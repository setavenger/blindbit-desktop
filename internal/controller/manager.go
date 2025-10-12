// Package controller is the interface between GUI and underlying data types handled outside
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/setavenger/blindbit-desktop/internal/configs"
	"github.com/setavenger/blindbit-lib/logging"
	"github.com/setavenger/blindbit-lib/networking/v2connect"
	"github.com/setavenger/blindbit-lib/scanning/scannerv2"
	"github.com/setavenger/blindbit-lib/wallet"
	"github.com/setavenger/go-bip352"
)

type Manager struct {
	Wallet          *wallet.Wallet `json:"wallet_data"`
	DataDir         string         `json:"-"`
	BirthHeight     int            `json:"birth_height"`
	DustLimit       int            `json:"dust_limit"`
	LabelCount      int            `json:"label_count"` // should always be 0
	MinChangeAmount uint64         `json:"min_change_amount"`
	OracleAddress   string         `json:"oracle_address"` // for now only gRPC possible will need a flag and options in future

	TransactionHistory []TxHistoryItem `json:"transaction_history"`

	// Scanner stufff
	Scanner *scannerv2.ScannerV2
}

func NewManager() *Manager {
	return &Manager{
		Wallet:    &wallet.Wallet{},
		DataDir:   "",
		DustLimit: configs.DefaultMinimumAmount, // default
		// LabelCount:      0,                    // default
		MinChangeAmount: configs.DefaultMinimumAmount, // default
		// OracleAddress:   "",
		Scanner: &scannerv2.ScannerV2{},
	}
}

func (m *Manager) ConstructScanner(ctx context.Context) error {
	if m.OracleAddress == "" {
		return errors.New("address is empty string")
	}
	oracleClient, err := v2connect.NewClient(ctx, m.OracleAddress)
	if err != nil {
		logging.L.Err(err).
			Str("address", m.OracleAddress).
			Msg("failed to constuct scanner")
		return err
	}

	// we only use change labels for now
	labels := []*bip352.Label{m.Wallet.GetLabel(0)}

	scanner := scannerv2.NewScannerV2(
		oracleClient,
		m.Wallet.SecretKeyScan,
		m.Wallet.PubKeySpend,
		labels,
	)

	m.Scanner = scanner

	return nil
}

/* DB preparations */

// Serialise creates byte data which can then be stored in an arbitrary way
func (m *Manager) Serialise() ([]byte, error) {
	// Marshal to JSON
	jsonData, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal wallet data: %w", err)
	}
	return jsonData, err
}

func (m *Manager) DeSerialise(data []byte) error {
	return json.Unmarshal(data, m)
}

/* Scanner stuff */
