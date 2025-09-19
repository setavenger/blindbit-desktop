package manager

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog"
	"github.com/setavenger/blindbit-desktop/internal/scanner"
	"github.com/setavenger/blindbit-lib/logging"
	"github.com/setavenger/blindbit-lib/utils"
	"github.com/setavenger/blindbit-lib/wallet"
	"github.com/spf13/viper"
	"github.com/tyler-smith/go-bip39"
	// Import blindbit-scan structures
)

// LabelGUI represents a labeled address for the GUI (simplified version for display)
type LabelGUI struct {
	PubKey  string `json:"pub_key"`
	Tweak   string `json:"tweak"`
	Address string `json:"address"`
	M       uint32 `json:"m"`
}

// Manager handles wallet operations and scanning
type Manager struct {
	config          *viper.Viper
	dataDir         string
	mu              sync.RWMutex
	scanHeight      uint64
	wallet          *wallet.Wallet
	utxos           []*wallet.OwnedUTXO
	logger          zerolog.Logger // Add logger field
	minChangeAmount uint64

	scanner *scanner.Scanner // Add scanner field
}

// NewManager creates a new wallet manager using the default data directory
func NewManager() (*Manager, error) {
	return NewManagerWithDataDir("")
}

// NewManagerWithDataDir creates a new wallet manager using the provided data directory.
// If dataDir is empty, it falls back to the default returned by getDataDir().
func NewManagerWithDataDir(dataDir string) (*Manager, error) {
	if dataDir == "" {
		dataDir = getDataDir()
	} else {
		dataDir = utils.ResolvePath(dataDir)
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("Error creating data directory: %v\n", err)
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Initialize configuration
	config := viper.New()
	config.SetConfigName("blindbit")
	config.SetConfigType("toml")
	config.AddConfigPath(dataDir)

	// Set default values
	setDefaultConfig(config)

	// Load existing config if it exists
	if err := config.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config file doesn't exist, create default one
		if err := config.WriteConfigAs(filepath.Join(dataDir, "blindbit.toml")); err != nil {
			fmt.Printf("Error writing default config: %v\n", err)
			return nil, fmt.Errorf("failed to write default config: %w", err)
		}
		fmt.Println("Default config file created")
	} else {
		fmt.Println("Existing config loaded")
	}

	fmt.Println("Creating manager instance...")
	manager := &Manager{
		config:  config,
		dataDir: dataDir,
		utxos:   []*wallet.OwnedUTXO{},
		// logger:  zerolog.New(os.Stdout).With().Caller().Timestamp().Logger(),
		logger: logging.L,
	}
	// initialize runtime fields from config
	manager.minChangeAmount = config.GetUint64("min_change_amount")
	return manager, nil
}

// getDataDir returns the appropriate data directory for the application
func getDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		homeDir = "."
	}
	dataDir := filepath.Join(homeDir, ".blindbit-desktop")
	fmt.Printf("Data directory: %s\n", dataDir)
	return dataDir
}

// setDefaultConfig sets default configuration values
func setDefaultConfig(config *viper.Viper) {
	config.SetDefault("network", "testnet")
	config.SetDefault("oracle_url", "https://silentpayments.dev/blindbit/mainnet")
	// config.SetDefault("http_port", 8080)
	// todo: this is probably not relevant anymore
	// config.SetDefault("private_mode", false)
	config.SetDefault("dust_limit", 546)
	config.SetDefault("label_count", 0)
	// todo: change to chain-tip minus a couple blocks
	config.SetDefault("birth_height", 900000)
	config.SetDefault("min_change_amount", 546)
}

// GenerateNewSeed generates a new BIP39 seed phrase
func (m *Manager) GenerateNewSeed() (string, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", fmt.Errorf("failed to generate entropy: %w", err)
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate mnemonic: %w", err)
	}

	return mnemonic, nil
}

// GetAddress returns the current wallet address
func (m *Manager) GetAddress() (addr string, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.wallet == nil {
		err = fmt.Errorf("no wallet loaded")
		logging.L.Err(nil).Msg("No wallet loaded")
		return "", err
	}

	addr = m.wallet.Address()

	return addr, nil
}

// GetUTXOs returns all UTXOs from the scan wallet
func (m *Manager) GetUTXOs() ([]*wallet.OwnedUTXO, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// For now, return stored UTXOs
	// TODO: Integrate with actual blindbit-scan scanning
	return m.utxos, nil
}

// GetUTXOsForGUI returns UTXOs in a GUI-friendly format
func (m *Manager) GetUTXOsForGUI() ([]*UTXOGUI, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var guiUTXOs []*UTXOGUI
	for _, utxo := range m.utxos {
		// Convert label if present
		var label *LabelGUI
		if utxo.Label != nil {
			label = &LabelGUI{
				PubKey:  fmt.Sprintf("%x", utxo.Label.PubKey[:]),
				Tweak:   fmt.Sprintf("%x", utxo.Label.Tweak[:]),
				Address: utxo.Label.Address,
				M:       utxo.Label.M,
			}
		}

		guiUTXO := &UTXOGUI{
			TxID:         fmt.Sprintf("%x", utxo.Txid[:]),
			Vout:         utxo.Vout,
			Amount:       utxo.Amount,
			State:        utxo.State.String(),
			Timestamp:    int64(utxo.Timestamp),
			PrivKeyTweak: fmt.Sprintf("%x", utxo.PrivKeyTweak[:]),
			PubKey:       fmt.Sprintf("%x", utxo.PubKey[:]),
			Label:        label,
		}
		guiUTXOs = append(guiUTXOs, guiUTXO)
	}

	return guiUTXOs, nil
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

// SendTransaction sends a transaction
func (m *Manager) SendTransaction(
	address string, amount int64, feeRate int64,
) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.wallet == nil {
		return "", fmt.Errorf("no wallet loaded")
	}

	var recipientsImpl = []wallet.RecipientImpl{
		{
			Address:  address,
			Amount:   uint64(amount),
			PkScript: []byte{},
		},
	}

	recipients := make([]wallet.Recipient, len(recipientsImpl))
	for i := range recipientsImpl {
		recipients[i] = &recipientsImpl[i]
	}

	txBytes, err := m.wallet.SendToRecipients(
		recipients, m.utxos, feeRate, m.minChangeAmount, true, false,
	)
	if err != nil {
		logging.L.Err(err).
			Any("utxos", m.utxos).
			Msg("failed to build transaction")
		return "", err
	}

	// For now, return a mock transaction ID
	// TODO: Integrate with actual blindbit-wallet-cli sending functionality
	return hex.EncodeToString(txBytes), err
}

// saveWalletConfig saves the wallet configuration
func (m *Manager) saveWalletConfig() error {
	if m.wallet == nil {
		return fmt.Errorf("no wallet to save")
	}

	m.wallet.UTXOs = m.utxos
	m.wallet.LastScanHeight = m.scanHeight

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(m.wallet, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal wallet data: %w", err)
	}

	// Write to file
	walletPath := filepath.Join(m.dataDir, "wallet.json")
	if err := os.WriteFile(walletPath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write wallet file: %w", err)
	}

	return nil
}

// HasWallet returns whether a wallet exists
func (m *Manager) HasWallet() bool {
	walletPath := filepath.Join(m.dataDir, "wallet.json")
	_, err := os.Stat(walletPath)
	exists := err == nil
	return exists
}

// GetChainTip returns the current chain tip
func (m *Manager) GetChainTip() (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.scanner == nil {
		return 0, fmt.Errorf("scanner not initialized")
	}

	return m.scanner.Client.GetChainTip()
}

// GetSyncStatus returns the sync status as a percentage
func (m *Manager) GetSyncStatus() (uint64, uint64, float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.scanner == nil {
		return 0, 0, 0.0
	}

	chainTip, err := m.scanner.Client.GetChainTip()
	if err != nil {
		return uint64(m.scanHeight), 0, 0.0
	}

	if chainTip == 0 {
		return uint64(m.scanHeight), 0, 0.0
	}

	percentage := float64(m.scanHeight) / float64(chainTip) * 100.0
	if percentage > 100.0 {
		percentage = 100.0
	}

	return uint64(m.scanHeight), chainTip, percentage
}
