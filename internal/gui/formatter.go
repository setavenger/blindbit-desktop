package gui

import (
	"strconv"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
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
