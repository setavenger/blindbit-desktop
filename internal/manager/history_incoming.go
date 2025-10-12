package manager

import (
	"fmt"

	"github.com/setavenger/blindbit-lib/wallet"
)

// addIncomingTransactionToHistoryWithBlockHeight adds an incoming transaction to history with block height
func (m *Manager) addIncomingTransactionToHistoryWithBlockHeight(txid string, utxo *wallet.OwnedUTXO, blockHeight uint64) {
	// Check if this is a self transfer (change output from our own transaction)
	isSelfTransfer := m.isSelfTransfer(txid)

	var txType string
	var description string

	if isSelfTransfer {
		txType = TransactionTypeSelfTransfer
		description = "Self transfer (change output)"
	} else {
		txType = TransactionTypeIncoming
		description = "Incoming transaction"
	}

	tx := &TransactionHistory{
		TxID:        txid,
		Type:        txType,
		Amount:      int64(utxo.Amount),
		Fee:         0,           // No fee for incoming
		BlockHeight: blockHeight, // Set block height from scanner
		Confirmed:   true,        // Transactions from scanned blocks are confirmed
		Description: description,
		Inputs:      []string{}, // Not relevant for incoming
		Outputs:     []string{fmt.Sprintf("%x:%d", utxo.Txid, utxo.Vout)},
	}

	m.AddTransactionToHistory(tx)

	// If this is a self transfer, update the outgoing transaction to reflect accurate net amount
	if isSelfTransfer {
		go func() {
			m.updateOutgoingTransactionForSelfTransfer(txid, int64(utxo.Amount))
		}()
	}
}

// isSelfTransfer checks if a transaction is a self transfer (change output from our own transaction)
func (m *Manager) isSelfTransfer(txid string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if we have an outgoing transaction with this txid
	for _, tx := range m.transactionHistory {
		if tx.TxID == txid && tx.Type == TransactionTypeOutgoing {
			return true
		}
	}

	return false
}
