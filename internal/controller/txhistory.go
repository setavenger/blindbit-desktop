package controller

import (
	"encoding/hex"
	"encoding/json"
)

// todo: outsource to blindbit-lib

type TxHistoryItem struct {
	BlockHeight int      `json:"block_height"`
	NetAmount   int      `json:"net_amount"`
	TxID        [32]byte `json:"txid"`
}

type TxHistoryItemJSON struct {
	BlockHeight int    `json:"block_height"`
	NetAmount   int    `json:"net_amount"`
	TxID        string `json:"txid"`
}

func (t *TxHistoryItem) MarshalJSON() ([]byte, error) {
	alias := TxHistoryItemJSON{
		BlockHeight: t.BlockHeight,
		NetAmount:   t.NetAmount,
		TxID:        hex.EncodeToString(t.TxID[:]),
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
	copy(t.TxID[:], txidBytes)

	return err
}
