package manager

import (
	"github.com/btcsuite/btcd/wire"
)

// LabelGUI represents a labeled address for the GUI (simplified version for display)
type LabelGUI struct {
	PubKey  string `json:"pub_key"`
	Tweak   string `json:"tweak"`
	Address string `json:"address"`
	M       uint32 `json:"m"`
}

// TransactionResult represents the result of a transaction
type TransactionResult struct {
	TxID             string         `json:"txid"`
	Hex              string         `json:"hex"`
	PSBT             string         `json:"psbt"`
	EffectiveFeeRate float64        `json:"effective_fee_rate"`
	Size             int            `json:"size"`
	Weight           int            `json:"weight"`
	VSize            int            `json:"vsize"`
	Fee              int64          `json:"fee"`
	TotalInput       int64          `json:"total_input"`
	TotalOutput      int64          `json:"total_output"`
	Inputs           []*wire.TxIn   `json:"inputs"`
	Outputs          []*wire.TxOut  `json:"outputs"`
	RecipientInfo    *RecipientInfo `json:"recipient_info,omitempty"`
}

// RecipientInfo contains information about transaction recipients from blindbit-lib
type RecipientInfo struct {
	IsSelfTransfer     bool              `json:"is_self_transfer"`
	ExternalRecipients []RecipientDetail `json:"external_recipients"`
	ChangeRecipient    *RecipientDetail  `json:"change_recipient,omitempty"`
	NetAmountSent      int64             `json:"net_amount_sent"`
	ChangeAmount       int64             `json:"change_amount"`
}

// RecipientDetail contains details about a specific recipient
type RecipientDetail struct {
	Address              string `json:"address"`
	Amount               int64  `json:"amount"`
	SilentPaymentAddress string `json:"silent_payment_address,omitempty"`
	IsOwnAddress         bool   `json:"is_own_address"`
}

// Transaction type constants
const (
	TransactionTypeIncoming     = "incoming"
	TransactionTypeOutgoing     = "outgoing"
	TransactionTypeSelfTransfer = "self_transfer"
)

// Transaction status constants
const (
	TransactionStatusPending   = "Pending"
	TransactionStatusConfirmed = "Confirmed"
)

// Amount unit constants
const (
	AmountUnitSats = "sats"
)

// TransactionHistory represents a transaction in the user's history
type TransactionHistory struct {
	TxID        string   `json:"txid"`
	Type        string   `json:"type"`         // TransactionTypeIncoming, TransactionTypeOutgoing, or TransactionTypeSelfTransfer
	Amount      int64    `json:"amount"`       // Net amount (positive for incoming, negative for outgoing)
	Fee         int64    `json:"fee"`          // Fee paid (0 for incoming)
	BlockHeight uint64   `json:"block_height"` // Block height (0 if unconfirmed)
	Confirmed   bool     `json:"confirmed"`    // Whether transaction is confirmed
	Description string   `json:"description"`  // Optional user description
	Inputs      []string `json:"inputs"`       // Input addresses/UTXOs (for display)
	Outputs     []string `json:"outputs"`      // Output addresses (for display)
}

// TransactionHistoryGUI represents a transaction for GUI display
type TransactionHistoryGUI struct {
	TxID        string `json:"txid"`
	Type        string `json:"type"`
	Amount      string `json:"amount"`       // Formatted amount with sign
	Fee         string `json:"fee"`          // Formatted fee
	BlockHeight string `json:"block_height"` // Formatted block height
	Confirmed   string `json:"confirmed"`    // "Confirmed" or "Pending"
	Description string `json:"description"`
}

// TransactionAnalysis represents the analysis of a transaction before broadcast
type TransactionAnalysis struct {
	TxID            string        `json:"txid"`
	IsSelfTransfer  bool          `json:"is_self_transfer"`
	NetAmountSent   int64         `json:"net_amount_sent"`
	ChangeAmount    int64         `json:"change_amount"`
	Fee             int64         `json:"fee"`
	ExternalOutputs []*wire.TxOut `json:"external_outputs"`
	ChangeOutputs   []*wire.TxOut `json:"change_outputs"`
	TotalInput      int64         `json:"total_input"`
}

// UTXOGUI represents a UTXO for the GUI display
type UTXOGUI struct {
	TxID         string    `json:"txid"`
	Vout         uint32    `json:"vout"`
	Amount       uint64    `json:"amount"`
	State        string    `json:"state"`
	Timestamp    int64     `json:"timestamp"`
	PrivKeyTweak string    `json:"priv_key_tweak"`
	PubKey       string    `json:"pub_key"`
	Label        *LabelGUI `json:"label,omitempty"`
}

// Network constants
const (
	NetworkTestnet = "testnet"
	NetworkMainnet = "mainnet"
	NetworkSignet  = "signet"
	NetworkRegtest = "regtest"
)

// Default network configuration
const DefaultNetwork = NetworkMainnet
