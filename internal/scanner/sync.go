package scanner

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/setavenger/blindbit-lib/wallet"
)

// At the top of the file, set CPU limit
func init() {
	// Use 75% of available cores (leaves some for other processes)
	runtime.GOMAXPROCS(runtime.NumCPU() * 3 / 4)
}

// SyncToTipWithProgress syncs from the last scan height to the current chain tip with progress callback
func (s *Scanner) SyncToTipWithProgress(progressCallback func(uint64)) error {
	syncStart := time.Now()
	defer func() {
		s.logger.Info().Dur("total_sync_duration", time.Since(syncStart)).Msg("sync completed")
	}()

	chainTip, err := s.Client.GetChainTip()
	if err != nil {
		return fmt.Errorf("failed to get chain tip: %w", err)
	}

	s.logger.Info().Uint64("chain_tip", chainTip).Msg("syncing to tip")

	startHeight := s.birthHeight
	if s.lastScanHeight >= startHeight {
		startHeight = s.lastScanHeight + 1
	}

	if startHeight > chainTip {
		return nil
	}

	if startHeight == 0 {
		startHeight = 1
	}

	var stopFlag bool

	dataCollector := make(map[uint64]*BlockData) // backlog to be collected
	errChan := make(chan error)
	dataChan := make(chan *BlockData, 100)

	finisherChan := make(chan *BlockData, 50)

	var mu sync.Mutex

	// fetch Routine
	go func() {
		semaphore := make(chan struct{}, 100) // Limit concurrent goroutines
		for i := startHeight; i <= chainTip && !stopFlag; i++ {
			semaphore <- struct{}{} // Acquire semaphore
			// Check for stop signal before scanning each block
			if stopFlag {
				s.logger.Info().Msg("stop flag called")
				return
			}
			go func(height uint64) {
				defer func() { <-semaphore }() // Release semaphore
				data, err := s.BlockFetcher(height)
				if err != nil {
					s.logger.Err(err).Uint64("height", height).Msg("failed fetching data")
					errChan <- err
					return
				}
				dataChan <- data
			}(i)
		}
	}()

	// processing in parallel as well finishing is done sequentially
	go func() {
		// semaphore := make(chan struct{}, 48) // Limit concurrent goroutines
		for !stopFlag {
			// semaphore <- struct{}{} // Acquire semaphore
			// s.logger.Debug().Msg("waiting for fetched blocks")
			select {
			case blockData := <-dataChan:
				processTiming := time.Now()
				func() {
					// defer func() { <-semaphore }() // Release semaphore
					// Time how long this block waited in the channel
					height := blockData.Height

					var ownedUTXOs []*wallet.OwnedUTXO
					ownedUTXOs, err = s.ProcessBlockData(blockData)
					if err != nil {
						s.logger.Err(err).Uint64("height", height).Msg("failed to process block data")
						errChan <- err
						return
					}
					blockData.OwnedUTXOs = ownedUTXOs
					finisherChan <- blockData
				}()
				processDuration := time.Since(processTiming)
				s.logger.Debug().Uint64("height", blockData.Height).
					Dur("process_duration", processDuration).
					Msg("block processed")
			}
		}
	}()

	// block finishing has to be done sequentially. Spent utxos will not be consistent... maybe?
	for !stopFlag {
		select {
		case <-s.stopChan:
			stopFlag = true
			s.logger.Info().Msg("scanning stopped during block scan")
			return nil
		case err := <-errChan:
			s.logger.Err(err).Msg("scanning failed")
			return err
		case blockData := <-finisherChan:
			finishingTime := time.Now()
			height := blockData.Height

			if height > s.lastScanHeight+1 {
				// store away as we need to process in order
				mu.Lock()
				dataCollector[height] = blockData
				mu.Unlock()
			}

			err = s.FinishBlock(blockData)
			if err != nil {
				s.logger.Err(err).Uint64("height", height).Msg("failed to finish block")
				return err
			}

			s.lastScanHeight = height

			// Report progress via callback
			progressCallback(height)

			// Save progress every 100 blocks
			if height%100 == 0 {
				s.logger.Debug().Uint64("height", height).Msg("saving progress")
			}

			// Log the time for just this individual block from channel to completion
			s.logger.Debug().Uint64("height", height).
				Dur("finishing_duration", time.Since(finishingTime)).
				Msg("individual block processed")

			var foundNextBlock bool = true
			for foundNextBlock {
				height++
				if blockData, foundNextBlock = dataCollector[height]; !foundNextBlock {
					continue
				}

				err = s.FinishBlock(blockData)
				if err != nil {
					s.logger.Err(err).Uint64("height", height).Msg("failed to finish block")
					return err
				}

				s.lastScanHeight = height

				// Report progress via callback
				progressCallback(height)

				// Save progress every 100 blocks
				if height%100 == 0 {
					s.logger.Debug().Uint64("height", height).Msg("saving progress")
				}

				// Log timing for this cascade block
				s.logger.Debug().Uint64("height", height).
					Msg("cascade block processed")
			}
		}
	}

	return nil
}
