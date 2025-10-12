package manager

import (
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

	scanner            *scanner.Scanner      // Add scanner field
	transactionHistory []*TransactionHistory // Transaction history
	saveMutex          sync.Mutex            // Mutex for file saving operations
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
	config, err := initializeConfig(dataDir)
	if err != nil {
		return nil, err
	}

	fmt.Println("Creating manager instance...")
	manager := &Manager{
		config:  config,
		dataDir: dataDir,
		utxos:   []*wallet.OwnedUTXO{},
		// logger:  zerolog.New(os.Stdout).With().Caller().Timestamp().Logger(),
		logger:             logging.L,
		transactionHistory: []*TransactionHistory{},
	}
	// initialize runtime fields from config
	manager.minChangeAmount = config.GetUint64("min_change_amount")
	return manager, nil
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
