package manager

import (
	"fmt"
	"time"

	"github.com/setavenger/blindbit-lib/wallet"
)

// reconcileTransactionHistory reconciles transaction history using UTXO set to fix self transfers and net amounts
func (m *Manager) reconcileTransactionHistory() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create UTXO map for efficient lookup
	utxoMap := make(map[string]*wallet.OwnedUTXO)
	for _, utxo := range m.utxos {
		txid := fmt.Sprintf("%x", utxo.Txid[:])
		key := fmt.Sprintf("%s:%d", txid, utxo.Vout)
		utxoMap[key] = utxo
	}

	// Group transactions by txid
	txGroups := make(map[string][]*TransactionHistory)
	for _, tx := range m.transactionHistory {
		txGroups[tx.TxID] = append(txGroups[tx.TxID], tx)
	}

	// Process each transaction group
	for txid, txs := range txGroups {
		var outgoingTx *TransactionHistory
		var incomingTxs []*TransactionHistory

		// Separate outgoing and incoming transactions
		for _, tx := range txs {
			if tx.Type == TransactionTypeOutgoing {
				outgoingTx = tx
			} else if tx.Type == TransactionTypeIncoming {
				incomingTxs = append(incomingTxs, tx)
			}
		}

		// If we have both outgoing and incoming transactions with same txid, it's a self transfer
		if outgoingTx != nil && len(incomingTxs) > 0 {
			// Mark outgoing transaction as self transfer
			outgoingTx.Type = TransactionTypeSelfTransfer

			// Mark incoming transactions as self transfers
			for _, incomingTx := range incomingTxs {
				incomingTx.Type = TransactionTypeSelfTransfer
				incomingTx.Description = "Self transfer (change output)"
			}

			// Calculate total change amount from UTXOs
			var totalChange int64
			for _, incomingTx := range incomingTxs {
				// Verify this UTXO exists in our UTXO set
				if utxo, exists := utxoMap[fmt.Sprintf("%s:%d", incomingTx.TxID, 0)]; exists {
					totalChange += int64(utxo.Amount)
				} else {
					// Fallback to transaction amount if UTXO not found
					totalChange += incomingTx.Amount
				}
			}

			// Update outgoing transaction with accurate net amount
			netAmountSent := outgoingTx.Amount + totalChange // outgoingTx.Amount is negative
			outgoingTx.Amount = netAmountSent
			outgoingTx.Description = fmt.Sprintf("Sent %d sats (change: %d sats)", -netAmountSent, totalChange)

			m.logger.Info().
				Str("txid", txid).
				Int64("net_amount", netAmountSent).
				Int64("change_amount", totalChange).
				Int("incoming_count", len(incomingTxs)).
				Msg("reconciled self transfer transaction using UTXO set")
		}
	}

	// Save reconciled history
	go func() {
		if err := m.saveTransactionHistory(); err != nil {
			m.logger.Error().Err(err).Msg("failed to save transaction history after reconciliation")
		}
	}()
}

// ReconcileTransactionHistory publicly exposes the reconciliation function
func (m *Manager) ReconcileTransactionHistory() {
	m.reconcileTransactionHistory()
}

// autoReconcileTransactionHistory automatically reconciles transaction history in the background
func (m *Manager) autoReconcileTransactionHistory() {
	// Wait a bit for any pending transactions to be added
	time.Sleep(2 * time.Second)

	// Perform reconciliation
	m.reconcileTransactionHistory()

	m.logger.Info().Msg("auto-reconciled transaction history")
}
