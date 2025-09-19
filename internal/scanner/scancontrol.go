package scanner

import (
	"fmt"
	"time"
)

// Start begins the scanning process in a goroutine
func (s *Scanner) Start() error {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()

	if s.scanning {
		s.logger.Warn().Msg("scanning already in progress, ignoring start request")
		return fmt.Errorf("scanning already in progress")
	}

	s.scanning = true
	s.stopChan = make(chan struct{}) // Reset stop channel
	s.doneChan = nil                 // Clear any existing done channel

	s.logger.Info().Msg("starting scanner")

	go func() {
		defer func() {
			s.scanMu.Lock()
			s.scanning = false
			s.scanMu.Unlock()
			s.logger.Info().Msg("scanner stopped")

			// Signal that we're done if there's a done channel
			if s.doneChan != nil {
				s.logger.Debug().Msg("signaling done channel")
				close(s.doneChan)
				s.doneChan = nil
			} else {
				s.logger.Debug().Msg("no done channel to signal")
			}
		}()

		for {
			// Check if we should stop before starting a new scan cycle
			select {
			case <-s.stopChan:
				s.logger.Debug().Msg("received stop signal before scan cycle")
				return
			default:
			}

			err := s.SyncToTipWithProgress(func(height uint64) {
				// Check for stop signal during progress updates
				select {
				case <-s.stopChan:
					return
				default:
				}

				// Call the original progress callback if set
				if s.progressCallback != nil {
					s.progressCallback(height)
				}
			})

			// Check for stop signal immediately after sync completes
			select {
			case <-s.stopChan:
				s.logger.Debug().Msg("received stop signal after sync")
				return
			default:
			}

			if err != nil {
				s.logger.Error().Err(err).Msg("error during scanning")
				// Don't call s.Stop() here as it can cause issues
				// The goroutine will exit naturally and the defer will handle cleanup
			} else {
				// Final UTXO update when scanning completes successfully
				if s.progressCallback != nil {
					s.progressCallback(s.lastScanHeight)
				}
			}

			// Check if we should stop after scan cycle
			select {
			case <-s.stopChan:
				s.logger.Debug().Msg("received stop signal after scan cycle")
				return
			default:
				// Wait before next scan cycle with interruptible sleep
				s.logger.Debug().Msg("waiting before next scan cycle")
				select {
				case <-s.stopChan:
					s.logger.Debug().Msg("received stop signal during wait")
					return
				case <-time.After(30 * time.Second):
					// Continue to next iteration
				}
			}
		}
	}()

	return nil
}

// StopSync signals the scanner to stop and waits for it to actually stop
func (s *Scanner) StopSync() {
	s.scanMu.Lock()

	if !s.scanning {
		s.scanMu.Unlock()
		s.logger.Debug().Msg("scanner not running, nothing to stop")
		return
	}

	s.logger.Info().Msg("stopping scanner synchronously")

	// Only create a done channel if we don't already have one
	var doneChan chan struct{}
	if s.doneChan == nil {
		doneChan = make(chan struct{})
		s.doneChan = doneChan
		s.logger.Debug().Msg("created new done channel")
	} else {
		// If there's already a done channel, we'll wait on it
		doneChan = s.doneChan
		s.logger.Debug().Msg("reusing existing done channel")
	}

	// Close the stop channel to signal the scanning goroutine to stop
	// Use a safer approach to avoid closing an already closed channel
	if s.stopChan != nil {
		select {
		case <-s.stopChan:
			// Channel already closed, do nothing
			s.logger.Debug().Msg("stop channel already closed")
		default:
			// Channel not closed, close it
			close(s.stopChan)
			s.logger.Debug().Msg("stop channel closed")
		}
	}

	s.scanMu.Unlock()

	// Wait for the scanning goroutine to actually stop
	// We'll wait up to 10 seconds for it to stop gracefully
	select {
	case <-doneChan:
		s.logger.Debug().Msg("scanner stopped gracefully")
	case <-time.After(10 * time.Second):
		s.logger.Warn().Msg("scanner stop timeout, forcing stop")
		// Force the scanning state to false
		s.scanMu.Lock()
		s.scanning = false
		s.doneChan = nil // Clear the done channel reference
		s.scanMu.Unlock()
	}
}

// Stop signals the scanner to stop scanning
func (s *Scanner) Stop() {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()

	if !s.scanning {
		s.logger.Debug().Msg("scanner not running, nothing to stop")
		return
	}

	s.logger.Info().Msg("stopping scanner")

	// Close the stop channel to signal the scanning goroutine to stop
	// Use a safer approach to avoid closing an already closed channel
	if s.stopChan != nil {
		select {
		case <-s.stopChan:
			// Channel already closed, do nothing
			s.logger.Debug().Msg("stop channel already closed")
		default:
			// Channel not closed, close it
			close(s.stopChan)
			s.logger.Debug().Msg("stop channel closed")
		}
	}

	// Don't set scanning to false here - let the goroutine do it
	// This prevents race conditions where Start could be called before the goroutine finishes
}

// IsScanning returns whether the scanner is currently scanning
func (s *Scanner) IsScanning() bool {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	return s.scanning
}

// RescanFromHeight resets the last scan height and rescans from the specified height
func (s *Scanner) RescanFromHeight(height uint64) error {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()

	if s.scanning {
		s.logger.Warn().Msg("cannot rescan while scanning is in progress")
		return fmt.Errorf("cannot rescan while scanning is in progress")
	}

	oldHeight := s.lastScanHeight
	s.lastScanHeight = height - 1 // Set to height-1 so scanning starts from the specified height

	s.logger.Info().
		Uint64("old_height", oldHeight).
		Uint64("new_height", height).
		Msg("reset scan height for rescan")

	return nil
}

// ForceRescanFromHeight performs a complete rescan from the specified height, clearing all UTXOs
func (s *Scanner) ForceRescanFromHeight(height uint64) error {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()

	if s.scanning {
		s.logger.Warn().Msg("cannot force rescan while scanning is in progress")
		return fmt.Errorf("cannot force rescan while scanning is in progress")
	}

	// Clear all UTXOs
	oldUTXOCount := len(s.allOwnedUTXOs)
	s.allOwnedUTXOs = nil

	oldHeight := s.lastScanHeight
	s.lastScanHeight = height - 1 // Set to height-1 so scanning starts from the specified height

	s.logger.Info().
		Uint64("old_height", oldHeight).
		Uint64("new_height", height).
		Int("cleared_utxos", oldUTXOCount).
		Msg("force rescan: cleared UTXOs and reset scan height")

	return nil
}
