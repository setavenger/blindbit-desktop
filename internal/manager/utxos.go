package manager

import (
	"encoding/binary"
	"fmt"

	"github.com/btcsuite/btcd/wire"
	"github.com/setavenger/blindbit-lib/wallet"
)

// GetUTXOs returns all UTXOs from the scan wallet
func (m *Manager) GetUTXOs() ([]*wallet.OwnedUTXO, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// For now, return stored UTXOs
	// TODO: Integrate with actual blindbit-scan scanning
	return m.utxos, nil
}

// RefreshUTXOs manually refreshes UTXOs from the scanner
func (m *Manager) RefreshUTXOs() {
	m.UpdateUTXOsFromScanner()
}

// ClearUTXOs manually clears all UTXOs (use with caution)
func (m *Manager) ClearUTXOs() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.utxos = []*wallet.OwnedUTXO{}

	if m.scanner != nil {
		m.scanner.ClearUTXOs()
	}

	// Save the cleared state
	if err := m.saveWalletConfig(); err != nil {
		fmt.Printf("Error saving wallet config after clearing UTXOs: %v\n", err)
	} else {
		fmt.Println("Successfully cleared all UTXOs")
	}
}

// GetUTXOStats returns statistics about the current UTXOs
func (m *Manager) GetUTXOStats() (total int, unspent int, spent int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total = len(m.utxos)
	for _, utxo := range m.utxos {
		switch utxo.State {
		case wallet.StateUnspent:
			unspent++
		case wallet.StateSpent:
			spent++
		}
	}
	return total, unspent, spent
}

// GetBalance calculates the total balance from unspent UTXOs
func (m *Manager) GetBalance() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var balance uint64
	var unspentCount int
	for _, utxo := range m.utxos {
		if utxo.State == wallet.StateUnspent {
			balance += utxo.Amount
			unspentCount++
		}
	}
	// Removed debug logging to reduce log noise
	return balance
}

// MarkUTXOsAsSpent marks UTXOs as spent based on transaction inputs (outpoints)
func (m *Manager) MarkUTXOsAsSpent(txInputs []*wire.TxIn) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	markedCount := 0

	// Create a map of UTXOs for efficient lookup by txid:vout
	utxoMap := make(map[string]*wallet.OwnedUTXO)
	for _, utxo := range m.utxos {
		key := fmt.Sprintf("%x:%d", utxo.Txid, utxo.Vout)
		utxoMap[key] = utxo
	}

	// Mark UTXOs as spent based on transaction inputs
	for _, txIn := range txInputs {
		key := fmt.Sprintf("%s:%d", txIn.PreviousOutPoint.Hash.String(), txIn.PreviousOutPoint.Index)
		if utxo, exists := utxoMap[key]; exists && utxo.State == wallet.StateUnspent {
			utxo.State = wallet.StateSpent
			markedCount++
			m.logger.Info().
				Str("txid", fmt.Sprintf("%x", utxo.Txid[:])).
				Uint32("vout", utxo.Vout).
				Msg("marked UTXO as spent after broadcast")
		}
	}

	if markedCount > 0 {
		// Update scanner's UTXO state
		if m.scanner != nil {
			m.updateScannerUTXOStates()
		}

		// Save the updated state (without lock to avoid deadlock)
		go func() {
			if err := m.saveWalletConfig(); err != nil {
				m.logger.Error().Err(err).Msg("failed to save wallet config after marking UTXOs as spent")
			}
		}()
	}

	return markedCount
}

// updateScannerUTXOStates updates the scanner's UTXO states to match the manager's
func (m *Manager) updateScannerUTXOStates() {
	if m.scanner == nil {
		return
	}

	// Create a map of manager UTXOs for efficient lookup
	managerUTXOs := make(map[string]wallet.UTXOState)
	for _, utxo := range m.utxos {
		key := fmt.Sprintf("%x:%d", utxo.Txid, utxo.Vout)
		managerUTXOs[key] = utxo.State
	}

	// Update scanner UTXOs to match manager state
	scannerUTXOs := m.scanner.GetAllOwnedUTXOs()
	for _, scannerUTXO := range scannerUTXOs {
		key := fmt.Sprintf("%x:%d", scannerUTXO.Txid, scannerUTXO.Vout)
		if managerState, exists := managerUTXOs[key]; exists {
			scannerUTXO.State = wallet.UTXOState(managerState)
		}
	}
}

// UpdateUTXOsFromScanner updates the manager's UTXOs from the scanner's found UTXOs
func (m *Manager) UpdateUTXOsFromScanner() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.scanner == nil {
		return
	}

	// Get UTXOs from scanner (now returns wallet.OwnedUTXO directly)
	scannerUTXOs := m.scanner.GetAllOwnedUTXOs()

	// convert scan to lib utxos
	// todo: change in blindbit scan as well
	var scannedUtxos = make([]*wallet.OwnedUTXO, len(scannerUTXOs))
	for i := range scannedUtxos {
		v := scannerUTXOs[i]
		scannedUtxos[i] = &wallet.OwnedUTXO{
			Txid:         v.Txid,
			Vout:         v.Vout,
			Amount:       v.Amount,
			PrivKeyTweak: v.PrivKeyTweak,
			PubKey:       v.PubKey,
			Timestamp:    v.Timestamp,
			State:        wallet.UTXOState(v.State),
			Label:        v.Label,
		}
	}

	// Create a map of existing UTXOs for efficient lookup
	existingUTXOs := make(map[[36]byte]*wallet.OwnedUTXO)
	oldCount := len(m.utxos)
	for _, utxo := range m.utxos {
		var key [36]byte
		copy(key[:], utxo.Txid[:])
		binary.PutUvarint(key[32:], uint64(utxo.Vout))
		existingUTXOs[key] = utxo
	}

	// Collect new UTXOs for transaction history (to avoid deadlock)
	var newUTXOsForHistory []*wallet.OwnedUTXO

	// Merge new UTXOs with existing ones, updating existing ones
	for _, newUTXO := range scannedUtxos {
		// todo: optimize to byte array key
		var key [36]byte
		copy(key[:], newUTXO.Txid[:])
		binary.PutUvarint(key[32:], uint64(newUTXO.Vout))

		// Check if this is a new UTXO (incoming transaction)
		if _, exists := existingUTXOs[key]; !exists {
			// This is a new UTXO, collect it for transaction history
			newUTXOsForHistory = append(newUTXOsForHistory, newUTXO)
		}

		existingUTXOs[key] = newUTXO
	}

	// Convert back to slice
	m.utxos = make([]*wallet.OwnedUTXO, 0, len(existingUTXOs))
	for _, utxo := range existingUTXOs {
		m.utxos = append(m.utxos, utxo)
	}

	// Only log if there are actual changes
	if oldCount != len(m.utxos) {
		m.logger.Info().
			Int("old_count", oldCount).
			Int("new_count", len(m.utxos)).
			Msg("merged UTXOs from scanner")
	}

	// Save to disk (without lock to avoid deadlock)
	go func() {
		if err := m.saveWalletConfig(); err != nil {
			m.logger.Error().Err(err).Msg("failed to save wallet config after UTXO update")
		}
	}()

	// Add new transactions to history (without lock to avoid deadlock)
	if len(newUTXOsForHistory) > 0 {
		go func() {
			// Get the current scan height from scanner to mark transactions as confirmed
			// Note: This represents the height of the block that was just scanned
			currentScanHeight := m.scanner.GetLastScanHeight()
			for _, utxo := range newUTXOsForHistory {
				txid := fmt.Sprintf("%x", utxo.Txid[:])
				// Use the new method that sets confirmed=true and block height
				// Transactions found during scanning are confirmed by definition
				m.addIncomingTransactionToHistoryWithBlockHeight(txid, utxo, currentScanHeight)
			}
			// Auto-reconcile after adding new transactions
			m.autoReconcileTransactionHistory()
		}()
	}
}
