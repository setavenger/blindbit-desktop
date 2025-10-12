package manager

import (
	"fmt"

	"github.com/btcsuite/btcd/wire"
)

// formatSats formats satoshis as a human-readable string
func formatSats(sats int64) string {
	return fmt.Sprintf("%d sats", sats)
}

// addOutgoingTransactionToHistoryWithRecipientInfo adds an outgoing transaction to history using recipient information
func (m *Manager) addOutgoingTransactionToHistoryWithRecipientInfo(txid string, txInputs []*wire.TxIn, recipientInfo *RecipientInfo) {
	// Calculate total input amount from our UTXOs
	var totalInput int64
	var inputAddresses []string

	m.mu.RLock()
	utxoMap := make(map[string]uint64) // key: "txid:vout", value: amount
	for _, utxo := range m.utxos {
		key := fmt.Sprintf("%x:%d", utxo.Txid, utxo.Vout)
		utxoMap[key] = utxo.Amount
	}
	m.mu.RUnlock()

	for _, txIn := range txInputs {
		key := fmt.Sprintf("%s:%d", txIn.PreviousOutPoint.Hash.String(), txIn.PreviousOutPoint.Index)
		if amount, exists := utxoMap[key]; exists {
			totalInput += int64(amount)
			addr := fmt.Sprintf("%s:%d", txIn.PreviousOutPoint.Hash.String(), txIn.PreviousOutPoint.Index)
			inputAddresses = append(inputAddresses, addr)
		}
	}

	// Determine transaction type and description based on recipient information
	txType := TransactionTypeOutgoing
	description := "Outgoing transaction"

	if recipientInfo.IsSelfTransfer {
		txType = TransactionTypeSelfTransfer
		if recipientInfo.ChangeAmount > 0 {
			description = fmt.Sprintf("Self transfer (change: %s)", formatSats(recipientInfo.ChangeAmount))
		} else {
			description = "Self transfer"
		}
	}

	// Create output addresses list
	var outputAddresses []string
	for _, recipient := range recipientInfo.ExternalRecipients {
		outputAddresses = append(outputAddresses, recipient.Address)
	}
	if recipientInfo.ChangeRecipient != nil {
		outputAddresses = append(outputAddresses, recipientInfo.ChangeRecipient.Address)
	}

	// Create transaction history entry with accurate information from recipient info
	tx := &TransactionHistory{
		TxID:        txid,
		Type:        txType,
		Amount:      -recipientInfo.NetAmountSent, // Negative for outgoing
		Fee:         totalInput - recipientInfo.NetAmountSent - recipientInfo.ChangeAmount,
		BlockHeight: 0, // Unconfirmed initially
		Confirmed:   false,
		Description: description,
		Inputs:      inputAddresses,
		Outputs:     outputAddresses,
	}

	m.AddTransactionToHistory(tx)

	m.logger.Info().
		Str("txid", txid).
		Str("type", txType).
		Int64("net_amount", recipientInfo.NetAmountSent).
		Int64("change_amount", recipientInfo.ChangeAmount).
		Bool("is_self_transfer", recipientInfo.IsSelfTransfer).
		Msg("added outgoing transaction to history with recipient info")
}

// updateOutgoingTransactionForSelfTransfer updates an outgoing transaction when we detect it's a self transfer
func (m *Manager) updateOutgoingTransactionForSelfTransfer(txid string, changeAmount int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the outgoing transaction and update it with accurate net amount
	for _, tx := range m.transactionHistory {
		if tx.TxID == txid && tx.Type == TransactionTypeOutgoing {
			// Calculate the actual net amount sent (excluding change)
			// The current amount is negative, so we add the change amount to get the net
			netAmountSent := tx.Amount + changeAmount // tx.Amount is negative, changeAmount is positive

			tx.Amount = netAmountSent // Update to net amount sent
			tx.Description = fmt.Sprintf("Sent %d sats (change: %d sats)", -netAmountSent, changeAmount)

			m.logger.Info().
				Str("txid", txid).
				Int64("net_amount", netAmountSent).
				Int64("change_amount", changeAmount).
				Msg("updated outgoing transaction for self transfer")
			break
		}
	}

	// Save updated history
	go func() {
		if err := m.saveTransactionHistory(); err != nil {
			m.logger.Error().Err(err).Msg("failed to save transaction history after self transfer update")
		}
	}()
}
