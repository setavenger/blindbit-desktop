package manager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/setavenger/blindbit-lib/logging"
	"github.com/setavenger/blindbit-lib/types"
	"github.com/setavenger/blindbit-lib/wallet"
	"github.com/setavenger/go-bip352"
	"github.com/tyler-smith/go-bip39"
)

// CreateWallet creates a new wallet with the given seed
func (m *Manager) CreateWallet(seed string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	network := types.Network(m.config.GetString("network"))
	err := m.createWalletInternal(seed, network)
	if err != nil {
		return err
	}

	// Clear any existing UTXOs for new wallet
	m.utxos = []*wallet.OwnedUTXO{}

	// Initialize scan height to birth height for new wallets
	birthHeight := m.config.GetUint64("birth_height")
	m.scanHeight = birthHeight - 1 // Start scanning from birth height

	logging.L.Debug().Msgf(
		"Created new wallet with scan height: %d (birth height: %d)\n",
		m.scanHeight, birthHeight,
	)

	// Setup scanner in background to avoid blocking startup
	go func() {
		if err := m.setupScanner(); err != nil {
			m.logger.Error().Err(err).Msg("failed to setup scanner during wallet creation")
			return
		}

		// Clear scanner UTXOs for new wallet
		if m.scanner != nil {
			m.scanner.ClearUTXOs()
		}
	}()

	return m.saveWalletConfig()
}

// createWalletInternal creates a wallet with the given seed and network (assumes lock is already held)
func (m *Manager) createWalletInternal(seed string, network types.Network) error {
	// Validate mnemonic
	if !bip39.IsMnemonicValid(seed) {
		return fmt.Errorf("invalid mnemonic")
	}

	// Generate seed from mnemonic
	seedBytes := bip39.NewSeed(seed, "")

	// Get network parameters
	params, ok := types.NetworkParams[network]
	if !ok {
		return fmt.Errorf("unsupported network: %s", network)
	}

	// Create master key
	master, err := hdkeychain.NewMaster(seedBytes, params)
	if err != nil {
		return fmt.Errorf("failed to create master key: %w", err)
	}

	// Derive BIP352 keys
	scanSecret, spendSecret, err := bip352.DeriveKeysFromMaster(
		master, network == types.NetworkMainnet,
	)

	if err != nil {
		return fmt.Errorf("failed to derive keys: %w", err)
	}

	// Create wallet instance
	m.wallet = &wallet.Wallet{
		Network:        network,
		Mnemonic:       seed,
		SecretKeyScan:  scanSecret,
		SecretKeySpend: spendSecret,
		PubKeyScan:     *bip352.PubKeyFromSecKey(&scanSecret),
		PubKeySpend:    *bip352.PubKeyFromSecKey(&spendSecret),
	}

	// Save wallet configuration
	return m.saveWalletConfig()
}

// LoadWallet loads an existing wallet
func (m *Manager) LoadWallet() error {
	logging.L.Debug().Msg("Starting LoadWallet...")
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if wallet exists
	walletPath := filepath.Join(m.dataDir, "wallet.json")
	logging.L.Debug().Str("wallet_path", walletPath).Msg("Wallet path")
	if _, err := os.Stat(walletPath); os.IsNotExist(err) {
		logging.L.Debug().Msg("Wallet file does not exist")
		return fmt.Errorf("no wallet found")
	}

	// Load wallet data
	logging.L.Debug().Msg("Reading wallet file...")
	data, err := os.ReadFile(walletPath)
	if err != nil {
		logging.L.Debug().Err(err).Msg("Error reading wallet file")
		return fmt.Errorf("failed to read wallet file: %w", err)
	}
	logging.L.Debug().Int("wallet_size", len(data)).Msg("Wallet file read")

	var walletData wallet.Wallet
	logging.L.Debug().Msg("Unmarshaling wallet data...")
	if err := json.Unmarshal(data, &walletData); err != nil {
		logging.L.Debug().Err(err).Msg("Error unmarshaling wallet data")
		return fmt.Errorf("failed to unmarshal wallet data: %w", err)
	}
	logging.L.Debug().Msg("Wallet data unmarshaled successfully")

	m.wallet = &walletData
	m.utxos = walletData.UTXOs
	m.scanHeight = walletData.LastScanHeight

	// Load transaction history
	if err := m.loadTransactionHistory(); err != nil {
		m.logger.Error().Err(err).Msg("failed to load transaction history")
		// Continue without transaction history rather than failing
	} else {
		// Auto-reconcile transaction history to fix self transfers and net amounts
		// Run in background to avoid blocking initialization
		go m.autoReconcileTransactionHistory()
	}

	// Setup scanner in background to avoid blocking startup
	go func() {
		if err := m.setupScanner(); err != nil {
			m.logger.Error().Err(err).Msg("failed to setup scanner during startup")
		}
	}()

	logging.L.Debug().
		Str("network", string(m.wallet.Network)).
		Int("utxos", len(m.utxos)).
		Uint64("scan_height", m.scanHeight).
		Msg("Wallet loaded")

	return nil
}
