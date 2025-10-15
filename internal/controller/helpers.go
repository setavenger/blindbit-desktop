package controller

import (
	"bytes"
	"encoding/hex"
	"sort"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/mempool"
	"github.com/btcsuite/btcd/wire"
	"github.com/setavenger/blindbit-lib/types"
	"github.com/setavenger/blindbit-lib/utils"
	"github.com/setavenger/blindbit-lib/wallet"
	"github.com/shopspring/decimal"
)

/* Helpers to avoid chaining to deep into sub structs */

func (m *Manager) GetNetwork() types.Network {
	return m.Wallet.Network
}

// GetSilentPaymentAddress returns the main Silent Payment address
// Note: This is NOT the change address (label 0), but the main receiving address
func (m *Manager) GetSilentPaymentAddress() string {
	return m.Wallet.Address()
}

// GetUTXOsSorted returns UTXOs sorted by block height (newest first)
func (m *Manager) GetUTXOsSorted() []*wallet.OwnedUTXO {
	utxos := m.Wallet.GetUTXOs()

	// Sort by height in descending order (newest first)
	sort.Slice(utxos, func(i, j int) bool {
		return utxos[i].Height > utxos[j].Height
	})

	return utxos
}

// GetUnspentUTXOsSorted returns only unspent UTXOs sorted by block height (newest first)
func (m *Manager) GetUnspentUTXOsSorted() []*wallet.OwnedUTXO {
	utxos := m.Wallet.GetUTXOs(wallet.StateUnspent)

	// Sort by height in descending order (newest first)
	sort.Slice(utxos, func(i, j int) bool {
		return utxos[i].Height > utxos[j].Height
	})

	return utxos
}

// GetTxID extracts transaction ID from wire.MsgTx
func GetTxID(tx *wire.MsgTx) [32]byte {
	txHash := tx.TxHash()
	var txid [32]byte
	copy(txid[:], txHash[:])
	utils.ReverseBytes(txid[:])
	return txid
}

// SerializeTx converts transaction to hex string for broadcasting
func SerializeTx(tx *wire.MsgTx) (string, error) {
	var buf bytes.Buffer
	err := tx.Serialize(&buf)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(buf.Bytes()), nil
}

// CalculateTxFee calculates fee from inputs and outputs
func CalculateTxFee(inputSum, outputSum uint64) uint64 {
	if inputSum > outputSum {
		return inputSum - outputSum
	}
	return 0
}

// CalculateTxVBytes calculates the virtual bytes (vbytes) of a transaction
func CalculateTxVBytes(tx *wire.MsgTx) uint64 {
	return uint64(mempool.GetTxVirtualSize(btcutil.NewTx(tx)))
}

// CalculateFeeRate calculates fee rate in sat/vB
func CalculateFeeRate(fee, vbytes uint64) float64 {
	if vbytes == 0 {
		return 0
	}

	out, _ := decimal.NewFromInt(int64(fee)).Div(decimal.NewFromInt(int64(vbytes))).Float64()
	return out

}
