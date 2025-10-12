package manager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AddTransactionToHistory adds a transaction to the history
func (m *Manager) AddTransactionToHistory(tx *TransactionHistory) {
	m.mu.Lock()

	// Check if transaction already exists
	for _, existing := range m.transactionHistory {
		if existing.TxID == tx.TxID {
			m.logger.Info().
				Str("txid", tx.TxID).
				Msg("transaction already exists in history, skipping")
			m.mu.Unlock()
			return // Already exists
		}
	}

	m.transactionHistory = append(m.transactionHistory, tx)
	m.mu.Unlock()

	m.logger.Info().
		Str("txid", tx.TxID).
		Str("type", tx.Type).
		Int64("amount", tx.Amount).
		Int("total_history_count", len(m.transactionHistory)).
		Msg("added transaction to history")

	// Save to disk (without lock to avoid deadlock)
	go func() {
		if err := m.saveTransactionHistory(); err != nil {
			m.logger.Error().Err(err).Msg("failed to save transaction history")
		}
	}()
}

// GetTransactionHistory returns the transaction history
func (m *Manager) GetTransactionHistory() []*TransactionHistory {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]*TransactionHistory, len(m.transactionHistory))
	copy(result, m.transactionHistory)
	return result
}

// GetTransactionHistoryForGUI returns transaction history formatted for GUI
func (m *Manager) GetTransactionHistoryForGUI() ([]*TransactionHistoryGUI, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var guiHistory []*TransactionHistoryGUI
	for _, tx := range m.transactionHistory {
		// Format amount with sign
		var amountStr string
		if tx.Amount > 0 {
			amountStr = fmt.Sprintf("+%d %s", tx.Amount, AmountUnitSats)
		} else {
			amountStr = fmt.Sprintf("%d %s", tx.Amount, AmountUnitSats)
		}

		// Format fee
		feeStr := fmt.Sprintf("0 %s", AmountUnitSats)
		if tx.Fee > 0 {
			feeStr = fmt.Sprintf("%d %s", tx.Fee, AmountUnitSats)
		}

		// Format block height
		blockHeightStr := TransactionStatusPending
		if tx.BlockHeight > 0 {
			blockHeightStr = fmt.Sprintf("%d", tx.BlockHeight)
		}

		// Format confirmed status
		confirmedStr := TransactionStatusPending
		if tx.Confirmed {
			confirmedStr = TransactionStatusConfirmed
		}

		guiTx := &TransactionHistoryGUI{
			TxID:        tx.TxID,
			Type:        tx.Type,
			Amount:      amountStr,
			Fee:         feeStr,
			BlockHeight: blockHeightStr,
			Confirmed:   confirmedStr,
			Description: tx.Description,
		}
		guiHistory = append(guiHistory, guiTx)
	}

	return guiHistory, nil
}

// saveTransactionHistory saves transaction history to disk
func (m *Manager) saveTransactionHistory() error {
	m.saveMutex.Lock()
	defer m.saveMutex.Unlock()

	historyPath := filepath.Join(m.dataDir, "transaction_history.json")

	// Get a copy of transaction history with proper locking
	m.mu.RLock()
	historyCopy := make([]*TransactionHistory, len(m.transactionHistory))
	copy(historyCopy, m.transactionHistory)
	m.mu.RUnlock()

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(historyCopy, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal transaction history: %w", err)
	}

	// Write to temporary file first, then rename (atomic operation)
	tempPath := historyPath + ".tmp"
	if err := os.WriteFile(tempPath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write transaction history temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, historyPath); err != nil {
		return fmt.Errorf("failed to rename transaction history file: %w", err)
	}

	return nil
}

// loadTransactionHistory loads transaction history from disk
func (m *Manager) loadTransactionHistory() error {
	historyPath := filepath.Join(m.dataDir, "transaction_history.json")

	// Check if file exists
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		// File doesn't exist, start with empty history
		// No locking needed during initialization - this is called before any concurrent access
		m.transactionHistory = []*TransactionHistory{}
		return nil
	}

	// Read file
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return fmt.Errorf("failed to read transaction history file: %w", err)
	}

	// Unmarshal JSON
	var history []*TransactionHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return fmt.Errorf("failed to unmarshal transaction history: %w", err)
	}

	// Set the loaded history - no locking needed during initialization
	m.transactionHistory = history

	return nil
}

// UpdateTransactionConfirmation updates a transaction's confirmation status
func (m *Manager) UpdateTransactionConfirmation(
	txid string, blockHeight uint64, confirmed bool,
) {
	m.mu.Lock()

	found := false
	for _, tx := range m.transactionHistory {
		if tx.TxID == txid {
			tx.BlockHeight = blockHeight
			tx.Confirmed = confirmed
			found = true
			break
		}
	}

	m.mu.Unlock()

	if found {
		// Save to disk (without lock to avoid deadlock)
		go func() {
			if err := m.saveTransactionHistory(); err != nil {
				m.logger.Error().Err(err).Msg("failed to save transaction history after confirmation update")
			}
		}()
	}
}
