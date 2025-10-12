// Package controller is the interface between GUI and underlying data types handled outside
package controller

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/setavenger/blindbit-desktop/internal/configs"
	"github.com/setavenger/blindbit-desktop/internal/storage"
	"github.com/setavenger/blindbit-lib/logging"
	"github.com/setavenger/blindbit-lib/scanning/scannerv2"
	"github.com/setavenger/blindbit-lib/utils"
	"github.com/setavenger/blindbit-lib/wallet"
)

type Manager struct {
	Wallet          *wallet.Wallet `json:"wallet_data"`
	DataDir         string         `json:"-"`
	BirthHeight     int            `json:"birth_height"`
	DustLimit       int
	LabelCount      int // should always be 0
	MinChangeAmount uint64
	OracleAddress   string // for now only gRPC possible will need a flag and options in future

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

// NewManagerWithDataDir creates a new wallet manager using the provided data directory.
// If dataDir is empty, it falls back to the default returned by getDataDir().
func NewManagerWithDataDir(dataDir string) (*Manager, error) {
	if dataDir == "" {
		dataDir = configs.DefaultDataDir()
	} else {
		dataDir = utils.ResolvePath(dataDir)
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("Error creating data directory: %v\n", err)
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	manager, err := storage.LoadPlain(dataDir)
	if err != nil {
		logging.L.Err(err).Msg("failed to load manager")
		return nil, err
	}

	return manager, nil
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
