package controller

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/btcsuite/btcd/wire"
	"github.com/setavenger/blindbit-lib/logging"
	"github.com/setavenger/blindbit-lib/types"
	"github.com/setavenger/blindbit-lib/utils"
	"github.com/setavenger/blindbit-lib/wallet"
)

func (m *Manager) PrepareTransaction(
	ctx context.Context,
	recipients []wallet.Recipient,
	feeRate uint32,
) (
	*wallet.TxMetadata, error,
) {
	txMetadata, err := m.Wallet.SendToRecipients(
		recipients,
		m.Wallet.GetUTXOs(),
		int64(feeRate),
		m.MinChangeAmount, // Minimum change amount
		false,             // Don't mark here! Wait until after successful broadcast
		false,             // Don't use unconfirmed spent todo: make optional in UI
	)
	if err != nil {
		return nil, err
	}

	return txMetadata, nil
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

// RecordSentTransaction records a sent transaction to history with proper net amount calculation
func (m *Manager) RecordSentTransaction(
	txMetadata *wallet.TxMetadata,
	recipients []wallet.Recipient,
) error {
	if txMetadata.Tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	// Use the TxItemFromTxMetadata function from blindbit-lib
	txItem, err := wallet.TxItemFromTxMetadata(m.Wallet, txMetadata)
	if err != nil {
		logging.L.Err(err).Msg("failed to map tx to tx history item")
		return err
	}

	// Add the transaction to history
	m.TransactionHistory = append(m.TransactionHistory, txItem)
	m.TransactionHistory.Sort()

	// Mark UTXOs as spent
	m.markUTXOsAsSpent(txMetadata.Tx)

	return nil
}

// markUTXOsAsSpent marks UTXOs as spent (unconfirmed spend) after successful broadcast
func (m *Manager) markUTXOsAsSpent(tx *wire.MsgTx) {
	for _, txIn := range tx.TxIn {
		// Find and mark the UTXO as spent
		for _, utxo := range m.Wallet.GetUTXOs() {
			isTxIDMatch := bytes.Equal(utxo.Txid[:], utils.ReverseBytesCopy(txIn.PreviousOutPoint.Hash[:]))
			isVoutMatch := utxo.Vout == txIn.PreviousOutPoint.Index
			if isTxIDMatch && isVoutMatch {
				// Mark UTXO as spent
				utxo.State = wallet.StateUnconfirmedSpent
				break
			}
		}
	}
}
