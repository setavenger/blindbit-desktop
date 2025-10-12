package manager

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/wire"
	"github.com/setavenger/blindbit-lib/logging"
	"github.com/setavenger/blindbit-lib/wallet"
)

// SendTransaction sends a transaction
func (m *Manager) SendTransaction(
	address string, amount int64, feeRate int64,
) (*TransactionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.wallet == nil {
		return nil, fmt.Errorf("no wallet loaded")
	}

	var recipientsImpl = []wallet.RecipientImpl{
		{
			Address:  address,
			Amount:   uint64(amount),
			PkScript: []byte{},
		},
	}

	recipients := make([]wallet.Recipient, len(recipientsImpl))
	for i := range recipientsImpl {
		recipients[i] = &recipientsImpl[i]
	}

	// TODO: Update to use enhanced SendToRecipients that returns recipient information
	// This will enable direct self-transfer detection and change recipient identification
	txMetadata, err := m.wallet.SendToRecipients(
		recipients, m.utxos, feeRate, m.minChangeAmount, true, false,
	)
	if err != nil {
		logging.L.Err(err).
			// Any("utxos", m.utxos).
			Msg("failed to build transaction")
		return nil, err
	}

	// Parse transaction to get detailed information
	result, err := m.parseTransactionDetails(txMetadata.Tx)
	if err != nil {
		logging.L.Err(err).Msg("failed to parse transaction details")
		return nil, fmt.Errorf("failed to parse transaction details: %w", err)
	}

	// Create RecipientInfo from txMetadata and transaction analysis
	result.RecipientInfo, err = m.createRecipientInfo(txMetadata, recipientsImpl, txMetadata.Tx)
	if err != nil {
		logging.L.Err(err).Msg("failed to create recipient info")
		return nil, fmt.Errorf("failed to create recipient info: %w", err)
	}

	return result, nil
}

// parseTransactionDetails parses transaction bytes and extracts detailed information
func (m *Manager) parseTransactionDetails(tx *wire.MsgTx) (*TransactionResult, error) {
	// Calculate txid
	txid := tx.TxHash().String()

	// Calculate transaction size and weight
	size := tx.SerializeSize()
	weight := tx.SerializeSizeStripped()*3 + size
	vsize := (weight + 3) / 4

	// Calculate total input and output amounts
	var totalInput, totalOutput int64

	// Create a map of UTXOs for efficient lookup
	utxoMap := make(map[string]uint64) // key: "txid:vout", value: amount
	for _, utxo := range m.utxos {
		key := fmt.Sprintf("%x:%d", utxo.Txid, utxo.Vout)
		utxoMap[key] = utxo.Amount
	}

	// Calculate total input from our UTXOs
	for _, txIn := range tx.TxIn {
		key := fmt.Sprintf("%s:%d", txIn.PreviousOutPoint.Hash.String(), txIn.PreviousOutPoint.Index)
		if amount, exists := utxoMap[key]; exists {
			totalInput += int64(amount)
		}
	}

	// Calculate total output
	for _, txOut := range tx.TxOut {
		totalOutput += txOut.Value
	}

	// Calculate fee and effective fee rate
	fee := totalInput - totalOutput
	effectiveFeeRate := float64(fee) / float64(vsize)

	// Note: We cannot create a PSBT from a raw transaction directly
	// PSBTs are created during the transaction building process, not from raw bytes
	// For now, we'll leave PSBT empty until we integrate proper PSBT creation
	var psbtHex string
	// TODO: Implement proper PSBT creation during transaction building
	// This would require modifying the wallet.SendToRecipients to return PSBT data

	var hexBuf bytes.Buffer
	err := tx.Serialize(&hexBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}

	return &TransactionResult{
		TxID:             txid,
		Hex:              hex.EncodeToString(hexBuf.Bytes()),
		PSBT:             psbtHex,
		EffectiveFeeRate: effectiveFeeRate,
		Size:             size,
		Weight:           weight,
		VSize:            vsize,
		Fee:              fee,
		TotalInput:       totalInput,
		TotalOutput:      totalOutput,
		Inputs:           tx.TxIn,
		Outputs:          tx.TxOut,
	}, nil
}

// createRecipientInfo creates RecipientInfo from txMetadata and transaction analysis
// NOTE: This function assumes the caller already holds the manager mutex
func (m *Manager) createRecipientInfo(txMetadata *wallet.TxMetadata, recipientsImpl []wallet.RecipientImpl, tx *wire.MsgTx) (*RecipientInfo, error) {
	// Get wallet address for ownership checks (no lock needed since caller holds it)
	if m.wallet == nil {
		return nil, fmt.Errorf("no wallet loaded")
	}
	walletAddress := m.wallet.Address()

	// Calculate total input amount from our UTXOs
	var totalInput int64
	utxoMap := make(map[string]uint64) // key: "txid:vout", value: amount
	for _, utxo := range m.utxos {
		key := fmt.Sprintf("%x:%d", utxo.Txid, utxo.Vout)
		utxoMap[key] = utxo.Amount
	}

	for _, txIn := range tx.TxIn {
		key := fmt.Sprintf("%s:%d", txIn.PreviousOutPoint.Hash.String(), txIn.PreviousOutPoint.Index)
		if amount, exists := utxoMap[key]; exists {
			totalInput += int64(amount)
		}
	}

	// Calculate total output amount
	var totalOutput int64
	for _, txOut := range tx.TxOut {
		totalOutput += txOut.Value
	}

	// Calculate fee (for validation, not used in return)
	_ = totalInput - totalOutput

	// Create external recipients from the original recipients
	var externalRecipients []RecipientDetail
	var netAmountSent int64
	var isSelfTransfer bool

	for _, recipient := range recipientsImpl {
		// Check if this is a self transfer (sending to our own address)
		if recipient.Address == walletAddress {
			isSelfTransfer = true
		}

		netAmountSent += int64(recipient.Amount)
		externalRecipients = append(externalRecipients, RecipientDetail{
			Address:      recipient.Address,
			Amount:       int64(recipient.Amount),
			IsOwnAddress: recipient.Address == walletAddress,
		})
	}

	// Handle change recipient from txMetadata
	var changeRecipient *RecipientDetail
	var changeAmount int64

	logging.L.Info().
		Bool("has_change_recipient", txMetadata.ChangeRecipient != nil).
		Msg("checking change recipient from txMetadata")

	if txMetadata.ChangeRecipient != nil {
		logging.L.Info().
			Str("change_address", txMetadata.ChangeRecipient.Address).
			Uint64("change_amount", txMetadata.ChangeRecipient.Amount).
			Msg("found change recipient in txMetadata")

		changeAmount = int64(txMetadata.ChangeRecipient.Amount)
		changeRecipient = &RecipientDetail{
			Address:      txMetadata.ChangeRecipient.Address,
			Amount:       int64(txMetadata.ChangeRecipient.Amount),
			IsOwnAddress: txMetadata.ChangeRecipient.Address == walletAddress,
		}
	} else {
		logging.L.Info().Msg("no change recipient in txMetadata")
	}

	recipientInfo := &RecipientInfo{
		IsSelfTransfer:     isSelfTransfer,
		ExternalRecipients: externalRecipients,
		ChangeRecipient:    changeRecipient,
		NetAmountSent:      netAmountSent,
		ChangeAmount:       changeAmount,
	}

	logging.L.Info().
		Int64("net_amount", recipientInfo.NetAmountSent).
		Int64("change_amount", recipientInfo.ChangeAmount).
		Bool("is_self_transfer", recipientInfo.IsSelfTransfer).
		Int("external_recipients", len(recipientInfo.ExternalRecipients)).
		Bool("has_change", recipientInfo.ChangeRecipient != nil).
		Msg("created recipient info")

	return recipientInfo, nil
}
