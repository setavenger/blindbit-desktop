package manager

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/btcsuite/btcd/wire"
	"github.com/setavenger/blindbit-lib/logging"
)

// BroadcastTransaction broadcasts a transaction to the network via mempool.space API
// It posts the raw hex as the request body to the appropriate endpoint based on network.
// After successful broadcast, marks the transaction inputs as spent.
// DEPRECATED: Use BroadcastTransactionWithInfo instead
func (m *Manager) BroadcastTransaction(txHex string, txInputs []*wire.TxIn) error {
	return fmt.Errorf("BroadcastTransaction is deprecated, use BroadcastTransactionWithInfo with RecipientInfo")
}

// BroadcastTransactionWithInfo broadcasts a transaction with recipient information
func (m *Manager) BroadcastTransactionWithInfo(txHex string, txInputs []*wire.TxIn, recipientInfo *RecipientInfo) error {
	m.mu.RLock()
	network := string(m.GetNetwork())
	m.mu.RUnlock()

	// Determine base URL by network
	var baseURL string
	switch network {
	case "mainnet", "":
		baseURL = "https://mempool.space"
	case "testnet":
		baseURL = "https://mempool.space/testnet"
	case "signet":
		baseURL = "https://mempool.space/signet"
	default:
		return fmt.Errorf("broadcast not supported for network: %s", network)
	}

	url := baseURL + "/api/tx"

	// Prepare request
	body := strings.TrimSpace(txHex)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request failed: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept", "text/plain")

	// Execute
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("broadcast request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		// mempool.space returns error text in body
		return fmt.Errorf("broadcast failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	// On success, body contains the txid
	txid := strings.TrimSpace(string(respBody))
	logging.L.Info().Str("txid", txid).Msg("transaction broadcasted successfully")

	// Add outgoing transaction to history using recipient information
	if recipientInfo == nil {
		return fmt.Errorf("RecipientInfo is required for transaction broadcast")
	}

	logging.L.Info().
		Str("txid", txid).
		Int64("net_amount", recipientInfo.NetAmountSent).
		Int64("change_amount", recipientInfo.ChangeAmount).
		Bool("is_self_transfer", recipientInfo.IsSelfTransfer).
		Int("external_recipients", len(recipientInfo.ExternalRecipients)).
		Msg("adding transaction to history with recipient info")

	// Use enhanced recipient information for accurate transaction history
	m.addOutgoingTransactionToHistoryWithRecipientInfo(txid, txInputs, recipientInfo)

	// Mark UTXOs as spent after successful broadcast
	if len(txInputs) > 0 {
		markedCount := m.MarkUTXOsAsSpent(txInputs)
		logging.L.Info().
			Str("txid", txid).
			Int("marked_utxos", markedCount).
			Msg("marked UTXOs as spent after successful broadcast")
	}

	return nil
}
