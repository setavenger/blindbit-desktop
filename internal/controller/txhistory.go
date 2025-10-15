package controller

import (
	"encoding/hex"
	"encoding/json"

	"github.com/setavenger/blindbit-lib/wallet"
)

// todo: outsource to blindbit-lib

type TxHistoryItem struct {
	BlockHeight int      `json:"block_height"`
	NetAmount   int      `json:"net_amount"`
	TxID        [32]byte `json:"txid"`
	Fee         uint64   `json:"fee,omitempty"`      // Fee in satoshis
	FeeRate     uint64   `json:"fee_rate,omitempty"` // Fee rate in sat/vB
}

type TxHistoryItemJSON struct {
	BlockHeight int    `json:"block_height"`
	NetAmount   int    `json:"net_amount"`
	TxID        string `json:"txid"`
	Fee         uint64 `json:"fee,omitempty"`
	FeeRate     uint64 `json:"fee_rate,omitempty"`
}

func (t *TxHistoryItem) MarshalJSON() ([]byte, error) {
	alias := TxHistoryItemJSON{
		BlockHeight: t.BlockHeight,
		NetAmount:   t.NetAmount,
		TxID:        hex.EncodeToString(t.TxID[:]),
		Fee:         t.Fee,
		FeeRate:     t.FeeRate,
	}
	return json.Marshal(alias)
}

func (t *TxHistoryItem) UnmarshalJSON(data []byte) error {
	var alias TxHistoryItemJSON
	err := json.Unmarshal(data, &alias)
	if err != nil {
		return err
	}

	var txidBytes []byte
	txidBytes, err = hex.DecodeString(alias.TxID)
	if err != nil {
		return err
	}

	t.BlockHeight = alias.BlockHeight
	t.NetAmount = alias.NetAmount
	t.Fee = alias.Fee
	t.FeeRate = alias.FeeRate
	copy(t.TxID[:], txidBytes)

	return err
}

// TransactionExists checks if a transaction already exists in history
func (m *Manager) TransactionExists(txid [32]byte) bool {
	for _, tx := range m.TransactionHistory {
		if tx.TxID == txid {
			return true
		}
	}
	return false
}

// SyncReceiveTransactions derives receive transactions from UTXOs
func (m *Manager) SyncReceiveTransactions() {
	// Group UTXOs by transaction ID
	txGroups := make(map[[32]byte][]*wallet.OwnedUTXO)
	for _, utxo := range m.Wallet.GetUTXOs() {
		txGroups[utxo.Txid] = append(txGroups[utxo.Txid], utxo)
	}

	// For each transaction not in history, add it
	for txid, utxos := range txGroups {
		if !m.TransactionExists(txid) {
			// Sum amounts and get earliest height
			var totalAmount uint64
			var height uint32

			for _, utxo := range utxos {
				totalAmount += utxo.Amount
				if height == 0 || utxo.Height < height {
					height = utxo.Height
				}
			}

			item := TxHistoryItem{
				TxID:        txid,
				BlockHeight: int(height),
				NetAmount:   int(totalAmount), // Positive for receives
			}
			m.TransactionHistory = append(m.TransactionHistory, item)
		}
	}
}

// UpdateTransactionHeight updates the block height when a transaction confirms
func (m *Manager) UpdateTransactionHeight(txid [32]byte, height uint32) {
	for i := range m.TransactionHistory {
		if m.TransactionHistory[i].TxID == txid {
			m.TransactionHistory[i].BlockHeight = int(height)
			break
		}
	}
}
