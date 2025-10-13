package controller

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/setavenger/blindbit-lib/types"
	"github.com/setavenger/blindbit-lib/wallet"
)

func (m *Manager) PrepareTransaction(ctx context.Context, recipients []wallet.Recipient, feeRate uint32) (*wallet.TxMetadata, error) {
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
		url = "https://mempool.space/api/testnet/tx"
	case types.NetworkSignet:
		url = "https://mempool.space/api/signet/tx"
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
