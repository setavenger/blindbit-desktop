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

	if err := m.setupScanner(); err != nil {
		return err
	}

	// Clear scanner UTXOs for new wallet
	if m.scanner != nil {
		m.scanner.ClearUTXOs()
	}

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
	fmt.Println("Starting LoadWallet...")
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if wallet exists
	walletPath := filepath.Join(m.dataDir, "wallet.json")
	fmt.Printf("Wallet path: %s\n", walletPath)
	if _, err := os.Stat(walletPath); os.IsNotExist(err) {
		fmt.Println("Wallet file does not exist")
		return fmt.Errorf("no wallet found")
	}

	// Load wallet data
	fmt.Println("Reading wallet file...")
	data, err := os.ReadFile(walletPath)
	if err != nil {
		fmt.Printf("Error reading wallet file: %v\n", err)
		return fmt.Errorf("failed to read wallet file: %w", err)
	}
	fmt.Printf("Wallet file read, size: %d bytes\n", len(data))

	var walletData wallet.Wallet
	fmt.Println("Unmarshaling wallet data...")
	if err := json.Unmarshal(data, &walletData); err != nil {
		fmt.Printf("Error unmarshaling wallet data: %v\n", err)
		return fmt.Errorf("failed to unmarshal wallet data: %w", err)
	}
	fmt.Println("Wallet data unmarshaled successfully")

	m.wallet = &walletData
	m.utxos = walletData.UTXOs
	m.scanHeight = walletData.LastScanHeight

	if err := m.setupScanner(); err != nil {
		return err
	}

	fmt.Printf("Wallet loaded: Network=%s, UTXOs=%d, ScanHeight=%d\n",
		m.wallet.Network, len(m.utxos), m.scanHeight)

	return nil
}
