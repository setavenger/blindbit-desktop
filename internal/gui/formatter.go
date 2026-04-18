package gui

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/setavenger/blindbit-lib/wallet"
)

// FormatNumber formats a number with thousand separators (commas) using golang.org/x/text
func FormatNumber(n int64) string {
	p := message.NewPrinter(language.English)
	return p.Sprintf("%d", n)
}

// FormatUint64 formats an unsigned integer with thousand separators (commas)
func FormatUint64(n uint64) string {
	p := message.NewPrinter(language.English)
	return p.Sprintf("%d", n)
}

// FormatUint32 formats an unsigned 32-bit integer with thousand separators (commas)
func FormatUint32(n uint32) string {
	p := message.NewPrinter(language.English)
	return p.Sprintf("%d", n)
}

// FormatSatoshi formats a satoshi amount with thousand separators and "sats" suffix
func FormatSatoshi(amount int64) string {
	p := message.NewPrinter(language.English)
	return p.Sprintf("%d sats", amount)
}

// FormatSatoshiUint64 formats a satoshi amount (uint64) with thousand separators and "sats" suffix
func FormatSatoshiUint64(amount uint64) string {
	p := message.NewPrinter(language.English)
	return p.Sprintf("%d sats", amount)
}

// FormatHeight formats a block height with thousand separators
func FormatHeight(height uint32) string {
	p := message.NewPrinter(language.English)
	return p.Sprintf("%d", height)
}

// FormatHeightUint64 formats a block height (uint64) with thousand separators
func FormatHeightUint64(height uint64) string {
	p := message.NewPrinter(language.English)
	return p.Sprintf("%d", height)
}

// ParseFormattedNumber parses a number string that may contain commas
func ParseFormattedNumber(str string) (int64, error) {
	// Remove commas
	cleanStr := strings.ReplaceAll(str, ",", "")
	return strconv.ParseInt(cleanStr, 10, 64)
}

// ParseFormattedUint64 parses a uint64 string that may contain commas
func ParseFormattedUint64(str string) (uint64, error) {
	// Remove commas
	cleanStr := strings.ReplaceAll(str, ",", "")
	return strconv.ParseUint(cleanStr, 10, 64)
}

// TxRowData holds the formatted strings for a single transaction row.
type TxRowData struct {
	TXID      string // truncated hex (8 chars + "...")
	Height    string // formatted block height
	NetAmount string // formatted net amount with sign
	Status    string // "Confirmed" or "Pending"
}

// FormatTxRow extracts the display strings for a transaction row, shared between
// the Dashboard's recent-transaction list and the full Transactions tab.
func FormatTxRow(tx *wallet.TxItem) TxRowData {
	txidHex := hex.EncodeToString(tx.TxID[:])

	netAmount := tx.NetAmount()
	var amountText string
	if netAmount > 0 {
		amountText = "+" + FormatSatoshiUint64(uint64(netAmount))
	} else if netAmount < 0 {
		amountText = FormatSatoshi(int64(netAmount))
	} else {
		amountText = FormatSatoshiUint64(0)
	}

	status := "Pending"
	if tx.ConfirmHeight > 0 {
		status = "Confirmed"
	}

	return TxRowData{
		TXID:      fmt.Sprintf("%.8s...", txidHex),
		Height:    FormatNumber(int64(tx.ConfirmHeight)),
		NetAmount: amountText,
		Status:    status,
	}
}
