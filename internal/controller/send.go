package controller

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/btcsuite/btcd/wire"
	"github.com/setavenger/blindbit-lib/types"
	"github.com/setavenger/blindbit-lib/wallet"
)

func (m *Manager) PrepareTransaction(
	ctx context.Context,
	recipients []wallet.Recipient,
	feeRate uint32,
) (
	*wallet.TxMetadata, error,
) {
	tx, err := wallet.SendToRecipients(
		m.Wallet,
		recipients,
		feeRate,
	)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// BroadcastTransaction broadcasts a transaction to mempool.space
func (m *Manager) BroadcastTransaction(txHex string, network types.Network) error {
	var url string
	switch network {
	case types.NetworkMainnet:
		url = "https://mempool.space/api/tx"
	case types.NetworkTestnet:
		url = "https://mempool.space/testnet/api/tx"
	case types.NetworkSignet:
		url = "https://mempool.space/signet/api/tx"
	default:
		return fmt.Errorf("unsupported network: %v", network)
	}

	resp, err := http.Post(url, "text/plain", strings.NewReader(txHex))
	if err != nil {
		return fmt.Errorf("failed to broadcast transaction: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("broadcast failed with status: %d", resp.StatusCode)
	}

	return nil
}

// AddToHistory adds a transaction to the transaction history
func (m *Manager) AddToHistory(txid [32]byte, netAmount int, blockHeight int) {
	item := TxHistoryItem{
		TxID:        txid,
		NetAmount:   netAmount,
		BlockHeight: blockHeight,
	}
	m.TransactionHistory = append(m.TransactionHistory, item)
}

// RecordSentTransaction records a sent transaction to history with proper net amount calculation
func (m *Manager) RecordSentTransaction(txMetadata *wallet.TxMetadata, recipients []wallet.Recipient) error {
	if txMetadata.Tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	// Extract TXID
	txid := GetTxID(txMetadata.Tx)

	// Calculate total sent to external recipients (exclude change)
	var totalSent uint64
	for _, recipient := range recipients {
		if !recipient.IsChange() {
			totalSent += recipient.GetAmount()
		}
	}

	// Calculate total output value
	var outputSum uint64
	for _, txOut := range txMetadata.Tx.TxOut {
		outputSum += uint64(txOut.Value)
	}

	// Calculate total input value from our UTXOs
	var inputSum uint64
	for _, txIn := range txMetadata.Tx.TxIn {
		// Find the UTXO being spent
		for _, utxo := range m.Wallet.GetUTXOs() {
			if utxo.Txid == txIn.PreviousOutPoint.Hash && utxo.Vout == txIn.PreviousOutPoint.Index {
				inputSum += utxo.Amount
				break
			}
		}
	}

	// Calculate actual fee: inputs - outputs
	fee := CalculateTxFee(inputSum, outputSum)

	// Calculate vbytes for fee rate
	vbytes := CalculateTxVBytes(txMetadata.Tx)
	feeRate := CalculateFeeRate(fee, vbytes)

	// Net amount is negative for sends: what we sent to external recipients + fee
	// This represents the total amount that left our wallet
	netAmount := -int64(totalSent + fee)

	// Create history item
	item := TxHistoryItem{
		TxID:        txid,
		NetAmount:   int(netAmount),
		BlockHeight: 0, // Unconfirmed initially
		Fee:         fee,
		FeeRate:     uint64(feeRate),
	}

	m.TransactionHistory = append(m.TransactionHistory, item)

	// Mark UTXOs as spent
	m.markUTXOsAsSpent(txMetadata.Tx)

	return nil
}

// markUTXOsAsSpent marks UTXOs as spent after successful broadcast
func (m *Manager) markUTXOsAsSpent(tx *wire.MsgTx) {
	for _, txIn := range tx.TxIn {
		// Find and mark the UTXO as spent
		for _, utxo := range m.Wallet.GetUTXOs() {
			if utxo.Txid == txIn.PreviousOutPoint.Hash && utxo.Vout == txIn.PreviousOutPoint.Index {
				// Mark UTXO as spent
				utxo.State = wallet.StateSpent
				break
			}
		}
	}
}
