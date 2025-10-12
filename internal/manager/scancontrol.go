package manager

import (
	"fmt"
	"time"
)

// StartScanning starts the scanning process
func (m *Manager) StartScanning() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.wallet == nil {
		return fmt.Errorf("no wallet loaded")
	}

	if m.scanner == nil {
		return fmt.Errorf("scanner not initialized")
	}

	// Check if scanner is already scanning
	if m.scanner.IsScanning() {
		return fmt.Errorf("scanning already in progress")
	}

	m.logger.Info().Msg("starting scan process")

	// Start the scanner (this launches the scanning goroutine)
	if err := m.scanner.Start(); err != nil {
		return fmt.Errorf("failed to start scanner: %w", err)
	}

	return nil
}

// StopScanning stops the scanning process
func (m *Manager) StopScanning() {
	m.mu.Lock()

	if m.scanner == nil {
		m.mu.Unlock()
		return
	}

	m.logger.Info().Uint64("current_height", m.scanHeight).Msg("stopping scan")
	m.scanner.Stop()
	m.logger.Info().Msg("stop signal sent to scanner")

	m.mu.Unlock()

	// Update UTXOs from scanner after stopping (without lock)
	go func() {
		// Wait a bit for scanner to finish
		time.Sleep(2 * time.Second)
		m.UpdateUTXOsFromScanner()
	}()
}

// IsScanning returns whether scanning is currently active
func (m *Manager) IsScanning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.scanner == nil {
		return false
	}

	return m.scanner.IsScanning()
}

// IsScannerReady returns whether the scanner is initialized and ready
func (m *Manager) IsScannerReady() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.scanner != nil
}

// UpdateScanHeight updates the scan height (for real-time UI updates)
func (m *Manager) UpdateScanHeight(height uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if height > m.scanHeight {
		m.scanHeight = height
	}
}

// GetScanHeight returns the current scan height
func (m *Manager) GetScanHeight() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.scanHeight
}
