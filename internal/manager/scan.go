package manager

import (
	"fmt"

	"github.com/setavenger/blindbit-desktop/internal/scanner"
	"github.com/setavenger/blindbit-lib/wallet"
)

// Add this helper method to Manager:
func (m *Manager) setupScanner() error {
	oracleURL := m.config.GetString("oracle_url")
	electrumURL := m.config.GetString("electrum_url")
	birthHeight := m.config.GetUint64("birth_height")
	labelCount := m.config.GetInt("label_count") // Get label count from config

	m.logger.Info().
		Str("oracle_url", oracleURL).
		Str("electrum_url", electrumURL).
		Uint64("birth_height", birthHeight).
		Int("label_count", labelCount).
		Uint64("saved_scan_height", m.scanHeight).
		Msg("setting up scanner")

	scanManger, err := scanner.NewScanner(
		oracleURL, electrumURL, m.wallet, &m.logger, labelCount,
	)
	if err != nil {
		return fmt.Errorf("failed to create scanner: %w", err)
	}

	// Set birth height from config
	scanManger.SetBirthHeight(birthHeight)

	// Set last scan height from saved data
	if m.scanHeight > 0 {
		scanManger.SetLastScanHeight(uint64(m.scanHeight))
		m.logger.Info().
			Uint64("last_scan_height", m.scanHeight).
			Msg("restored last scan height")
	}

	// Load existing UTXOs into the scanner
	if len(m.utxos) > 0 {
		// The scanner now uses wallet.OwnedUTXO directly, so we can pass them directly
		scanManger.LoadExistingUTXOs(m.utxos)
		m.logger.Info().
			Int("utxos_loaded", len(m.utxos)).
			Msg("loaded existing UTXOs into scanner")
	}

	// Set up progress callback for real-time updates
	scanManger.SetProgressCallback(func(height uint64) {
		m.UpdateScanHeight(height)

		// Update UTXOs from scanner every 100 blocks to reduce log noise
		if height%100 == 0 {
			m.UpdateUTXOsFromScanner()
		}

		// Also update UTXOs immediately when we reach the chain tip
		// This ensures we get the latest data even if we don't hit the 100-block interval
		chainTip, err := scanManger.Client.GetChainTip()
		if err == nil && height >= chainTip {
			m.UpdateUTXOsFromScanner()
		}
	})

	// Set up UTXO update callback for immediate updates when new UTXOs are found
	scanManger.SetUTXOUpdateCallback(func() {
		m.UpdateUTXOsFromScanner()
	})

	m.scanner = scanManger
	m.logger.Info().Msg("scanner setup completed")
	return nil
}

// RescanFromTip rescans from the current chain tip
// Deprecated: Nonsense function from AI
func (m *Manager) RescanFromTip() error {
	if m.scanner == nil {
		return fmt.Errorf("scanner not initialized")
	}

	// Get current chain tip
	chainTip, err := m.scanner.Client.GetChainTip()
	if err != nil {
		return fmt.Errorf("failed to get chain tip: %w", err)
	}

	fmt.Printf("[RescanFromTip] Rescanning from chain tip: %d\n", chainTip)
	return m.RescanFromHeight(chainTip)
}

// RescanFromHeight resets the last scanned height
// and triggers a rescan from the specified height
func (m *Manager) RescanFromHeight(height uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.wallet == nil {
		return fmt.Errorf("no wallet loaded")
	}

	if m.scanner == nil {
		return fmt.Errorf("scanner not initialized")
	}

	// Stop current scanning if running
	if m.scanner.IsScanning() {
		fmt.Println("[RescanFromHeight] Stopping current scan before rescan")
		m.scanner.StopSync() // Use synchronous stop
	}

	// Update the scan height
	oldHeight := m.scanHeight
	m.scanHeight = height - 1 // Set to height-1 so scanning starts from the specified height

	fmt.Printf("[RescanFromHeight] Resetting scan height from %d to %d\n", oldHeight, m.scanHeight)

	// Update the scanner's last scan height
	m.scanner.SetLastScanHeight(height - 1)

	// Save the updated scan height
	if err := m.saveWalletConfig(); err != nil {
		return fmt.Errorf("failed to save wallet config: %w", err)
	}

	fmt.Printf("[RescanFromHeight] Successfully reset scan height to %d\n", height)
	return nil
}

// ForceRescanFromTip performs a complete rescan from the current chain tip, clearing existing UTXOs
func (m *Manager) ForceRescanFromTip() error {
	if m.scanner == nil {
		return fmt.Errorf("scanner not initialized")
	}

	// Get current chain tip
	chainTip, err := m.scanner.Client.GetChainTip()
	if err != nil {
		return fmt.Errorf("failed to get chain tip: %w", err)
	}

	fmt.Printf("[ForceRescanFromTip] Force rescanning from chain tip: %d\n", chainTip)
	return m.ForceRescanFromHeight(chainTip)
}

// ForceRescanFromHeight performs a complete rescan from the specified height, clearing existing UTXOs
func (m *Manager) ForceRescanFromHeight(height uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.wallet == nil {
		return fmt.Errorf("no wallet loaded")
	}

	if m.scanner == nil {
		return fmt.Errorf("scanner not initialized")
	}

	// Stop current scanning if running
	if m.scanner.IsScanning() {
		fmt.Println("[ForceRescanFromHeight] Stopping current scan before force rescan")
		m.scanner.StopSync() // Use synchronous stop
	}

	// Clear all existing UTXOs
	oldUTXOCount := len(m.utxos)
	m.utxos = []*wallet.OwnedUTXO{}
	m.scanner.ClearUTXOs()

	fmt.Printf("[ForceRescanFromHeight] Cleared %d existing UTXOs\n", oldUTXOCount)

	// Update the scan height
	oldHeight := m.scanHeight
	m.scanHeight = height - 1 // Set to height-1 so scanning starts from the specified height

	fmt.Printf("[ForceRescanFromHeight] Resetting scan height from %d to %d\n", oldHeight, m.scanHeight)

	// Update the scanner's last scan height
	m.scanner.SetLastScanHeight(height - 1)

	// Save the updated state
	if err := m.saveWalletConfig(); err != nil {
		return fmt.Errorf("failed to save wallet config: %w", err)
	}

	fmt.Printf("[ForceRescanFromHeight] Successfully reset scan height to %d and cleared UTXOs\n", height)
	return nil
}
