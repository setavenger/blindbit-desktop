// Package controller is the interface between GUI and underlying data types handled outside
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
	DustLimit       int            `json:"dust_limit"`
	LabelCount      int            `json:"label_count"` // should always be 0
	MinChangeAmount uint64         `json:"min_change_amount"`
	OracleAddress   string         `json:"oracle_address"` // for now only gRPC possible will need a flag and options in future

	TransactionHistory wallet.TxHistory        `json:"transaction_history"`
	OracleClient       *v2connect.OracleClient `json:"-"`
	Scanner            *scannerv2.ScannerV2    `json:"-"`

	// scnaner channels - internal use only
	// Deprecated: modify the blindbit-lib scanner
	// to properly expose these channels  for several listeners instead
	OwnedUTXOsChan     <-chan *wallet.OwnedUTXO `json:"-"`
	ProgressUpdateChan <-chan uint32            `json:"-"`

	// GUI update channels - for real-time UI updates
	GUIScanProgressChan chan uint32 `json:"-"` // todo: review sense of this channel logic
	StreamEndChan       chan bool   `json:"-"` // Signal when scanning streams end
}

func NewManager() *Manager {
	return &Manager{
		Wallet:              &wallet.Wallet{},
		DataDir:             "",
		DustLimit:           configs.DefaultMinimumAmount, // default
		LabelCount:          configs.DefaultLabelCount,    // default
		MinChangeAmount:     configs.DefaultMinimumAmount, // default
		OracleAddress:       configs.DefaultOracleAddress,
		TransactionHistory:  wallet.TxHistory{},     // Initialize empty TxHistory
		Scanner:             nil,                    // Don't initialize scanner until needed
		GUIScanProgressChan: make(chan uint32, 100), // Buffer for GUI updates
		StreamEndChan:       make(chan bool, 10),    // Buffer for stream end signals
	}
}

func (m *Manager) ConstructScanner(ctx context.Context) error {
	if m.Wallet == nil {
		return errors.New("wallet not initialized")
	}
	if m.OracleAddress == "" {
		return errors.New("address is empty string")
	}
	if m.OracleClient == nil {
		oracleClient, err := v2connect.NewClient(ctx, m.OracleAddress)
		if err != nil {
			logging.L.Err(err).
				Str("address", m.OracleAddress).
				Msg("failed to constuct scanner")
			return err
		}
		m.OracleClient = oracleClient
	}

	// we only use change labels for now
	labels := []*bip352.Label{m.Wallet.GetLabel(0)}

	scanner := scannerv2.NewScannerV2(
		m.OracleClient,
		m.Wallet.SecretKeyScan,
		m.Wallet.PubKeySpend,
		labels,
	)

	m.Scanner = scanner
	err := m.Scanner.AttachWallet(m.Wallet)
	if err != nil {
		logging.L.Err(err).Msg("failed to attach wallet to scanner")
		return err
	}

	m.OwnedUTXOsChan = m.Scanner.SubscribeOwnedUTXOs()
	m.ProgressUpdateChan = m.Scanner.ProgressUpdateChan()

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

// GetBalance returns the total balance of all UTXOs
func (m *Manager) GetBalance() uint64 {
	var total uint64
	utxos := m.Wallet.GetUTXOs()

	for _, utxo := range utxos {
		if utxo.State != wallet.StateUnspent {
			continue
		}
		total += utxo.Amount
	}

	return total
}

// GetBirthHeight returns the wallet's birth height
func (m *Manager) GetBirthHeight() uint64 {
	if m.Wallet == nil {
		return 0
	}
	return m.Wallet.BirthHeight
}

// SetBirthHeight sets the wallet's birth height and optionally LastScanHeight
func (m *Manager) SetBirthHeight(height uint64, setLastScanHeight bool) {
	if m.Wallet == nil {
		return
	}
	m.Wallet.BirthHeight = height
	if setLastScanHeight {
		m.Wallet.LastScanHeight = height
	}
}

// GetCurrentHeight queries the oracle for the current blockchain height
func (m *Manager) GetCurrentHeight() (uint32, error) {
	if m.OracleClient == nil {
		return 0, errors.New("oracle client not initialized")
	}
	resp, err := m.OracleClient.GetInfo(context.TODO())
	if err != nil {
		return 0, err
	}
	return uint32(resp.Height), nil
}

// SignalStreamEnd signals that a scanning stream has ended
func (m *Manager) SignalStreamEnd() {
	select {
	case m.StreamEndChan <- true:
		logging.L.Debug().Msg("stream end signal sent")
	default:
		logging.L.Debug().Msg("stream end channel full, skipping signal")
	}
}

// StartChannelHandling starts unified handling of scanner channels for background operations
func (m *Manager) StartChannelHandling(ctx context.Context, saveFunc func() error) {
	if m.OwnedUTXOsChan == nil || m.ProgressUpdateChan == nil {
		logging.L.Warn().Msg("scanner channels not initialized, skipping channel handling")
		return
	}

	logging.L.Info().Msg("starting unified channel handling for background scanning")

	// Channel for periodic saves
	saveTicker := time.NewTicker(15 * time.Second) // Save every 15 seconds
	defer saveTicker.Stop()

	// Channel for block-based saves
	blockSaveCounter := 0
	const blocksBetweenSaves = 100 // Save every 100 blocks

	// Handle progress updates and periodic saves
	go func() {
		for {
			select {
			case height := <-m.ProgressUpdateChan:
				// Update wallet's LastScanHeight
				m.Wallet.LastScanHeight = uint64(height)
				// logging.L.Debug().Uint32("scan_height", height).Msg("scan progress update")

				// Forward progress update to GUI channel for real-time updates
				select {
				case m.GUIScanProgressChan <- height:
					// Successfully sent to GUI
				default:
					// GUI channel is full, skip this update (non-blocking)
					// logging.L.Debug().Msg("GUI progress channel full, skipping update")
				}

				// Save every other block
				blockSaveCounter++
				if blockSaveCounter >= blocksBetweenSaves {
					if err := saveFunc(); err != nil {
						logging.L.Err(err).Msg("failed to save wallet after block progress")
					}
					blockSaveCounter = 0
				}

			case <-saveTicker.C:
				// Periodic save every 30 seconds
				if err := saveFunc(); err != nil {
					logging.L.Err(err).Msg("failed to save wallet periodically")
				} else {
					logging.L.Debug().Msg("wallet saved periodically")
				}

			case <-ctx.Done():
				logging.L.Info().Msg("stopping progress update handling")
				return
			}
		}
	}()

	// Handle new UTXOs
	go func() {
		for {
			select {
			case utxo := <-m.OwnedUTXOsChan:
				logging.L.Info().
					Str("txid", fmt.Sprintf("%x", utxo.Txid)).
					Uint32("vout", utxo.Vout).
					Uint64("amount", utxo.Amount).
					Uint32("height", utxo.Height).
					Msg("new UTXO discovered")

				err := m.TransactionHistory.AddOutUtxo(utxo)
				if err != nil {
					logging.L.Err(err).Msg("failed to add out UTXO to transaction history")
				}

				// Save wallet immediately when new UTXO is found
				if err := saveFunc(); err != nil {
					logging.L.Err(err).Msg("failed to save wallet after new UTXO found")
				} else {
					logging.L.Info().Msg("wallet saved after new UTXO discovery")
				}

			case <-ctx.Done():
				logging.L.Info().Msg("stopping UTXO handling")
				return
			}
		}
	}()
}
