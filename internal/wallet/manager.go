package wallet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/rs/zerolog"
	"github.com/setavenger/go-bip352"
	"github.com/spf13/viper"
	"github.com/tyler-smith/go-bip39"
)

// Network represents the Bitcoin network type
type Network string

const (
	NetworkMainnet Network = "mainnet"
	NetworkTestnet Network = "testnet"
	NetworkSignet  Network = "signet"
	NetworkRegtest Network = "regtest"
)

// UTXO represents a UTXO for the GUI
type UTXO struct {
	TxID      string `json:"txid"`
	Vout      uint32 `json:"vout"`
	Amount    uint64 `json:"amount"`
	State     string `json:"state"`
	Timestamp int64  `json:"timestamp"`
	Address   string `json:"address,omitempty"`
	Label     *Label `json:"label,omitempty"` // Add label information
}

// Label represents a labeled address for the GUI
type Label struct {
	PubKey  string `json:"pub_key"`
	Tweak   string `json:"tweak"`
	Address string `json:"address"`
	M       uint32 `json:"m"`
}

// Wallet represents the core wallet data
type Wallet struct {
	Network     Network   `json:"network"`
	Mnemonic    string    `json:"mnemonic"`
	ScanSecret  []byte    `json:"scan_secret"`
	SpendSecret []byte    `json:"spend_secret"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WalletData represents the complete wallet data stored on disk
type WalletData struct {
	Wallet     Wallet `json:"wallet"`
	UTXOs      []UTXO `json:"utxos"`
	LastHeight int64  `json:"last_height"`
}

// Manager handles wallet operations and scanning
type Manager struct {
	config     *viper.Viper
	dataDir    string
	mu         sync.RWMutex
	scanHeight int64
	wallet     *Wallet
	utxos      []UTXO
	logger     zerolog.Logger // Add logger field

	scanner *Scanner // Add scanner field
}

// networkParams maps network names to chain parameters
var networkParams = map[Network]*chaincfg.Params{
	NetworkMainnet: &chaincfg.MainNetParams,
	NetworkTestnet: &chaincfg.TestNet3Params,
	NetworkSignet:  &chaincfg.SigNetParams,
	NetworkRegtest: &chaincfg.RegressionNetParams,
}

// NewManager creates a new wallet manager
func NewManager() (*Manager, error) {
	fmt.Println("Creating new wallet manager...")
	dataDir := getDataDir()

	// Ensure data directory exists
	fmt.Println("Creating data directory...")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("Error creating data directory: %v\n", err)
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	fmt.Println("Data directory created/verified")

	// Initialize configuration
	fmt.Println("Initializing configuration...")
	config := viper.New()
	config.SetConfigName("blindbit")
	config.SetConfigType("toml")
	config.AddConfigPath(dataDir)

	// Set default values
	fmt.Println("Setting default config values...")
	setDefaultConfig(config)

	// Load existing config if it exists
	fmt.Println("Loading existing config...")
	if err := config.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Printf("Error reading config: %v\n", err)
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config file doesn't exist, create default one
		fmt.Println("Creating default config file...")
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
		utxos:   []UTXO{},
		logger:  zerolog.New(os.Stdout).With().Caller().Timestamp().Logger(),
	}
	fmt.Println("Manager instance created successfully")
	return manager, nil
}

// getDataDir returns the appropriate data directory for the application
func getDataDir() string {
	fmt.Println("Getting data directory...")
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
	config.SetDefault("oracle_url", "https://oracle.testnet.blindbit.com")
	config.SetDefault("electrum_url", "ssl://electrum.blockstream.info:60002")
	config.SetDefault("use_tor", false)
	config.SetDefault("tor_host", "localhost")
	config.SetDefault("tor_port", 9050)
	config.SetDefault("http_port", 8080)
	config.SetDefault("private_mode", false)
	config.SetDefault("dust_limit", 1000)
	config.SetDefault("label_count", 21)
	config.SetDefault("birth_height", 840000)
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

// CreateWallet creates a new wallet with the given seed
func (m *Manager) CreateWallet(seed string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	network := Network(m.config.GetString("network"))
	err := m.createWalletInternal(seed, network)
	if err != nil {
		return err
	}

	// Clear any existing UTXOs for new wallet
	m.utxos = []UTXO{}

	// Initialize scan height to birth height for new wallets
	birthHeight := m.config.GetUint64("birth_height")
	m.scanHeight = int64(birthHeight - 1) // Start scanning from birth height

	fmt.Printf("Created new wallet with scan height: %d (birth height: %d)\n", m.scanHeight, birthHeight)

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
func (m *Manager) createWalletInternal(seed string, network Network) error {
	// Validate mnemonic
	if !bip39.IsMnemonicValid(seed) {
		return fmt.Errorf("invalid mnemonic")
	}

	// Generate seed from mnemonic
	seedBytes := bip39.NewSeed(seed, "")

	// Get network parameters
	params, ok := networkParams[network]
	if !ok {
		return fmt.Errorf("unsupported network: %s", network)
	}

	// Create master key
	master, err := hdkeychain.NewMaster(seedBytes, params)
	if err != nil {
		return fmt.Errorf("failed to create master key: %w", err)
	}

	// Derive BIP352 keys
	scanSecret, spendSecret, err := bip352.DeriveKeysFromMaster(master, network == NetworkMainnet)
	if err != nil {
		return fmt.Errorf("failed to derive keys: %w", err)
	}

	// Create wallet instance
	m.wallet = &Wallet{
		Network:     network,
		Mnemonic:    seed,
		ScanSecret:  scanSecret[:],
		SpendSecret: spendSecret[:],
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
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

	var walletData WalletData
	fmt.Println("Unmarshaling wallet data...")
	if err := json.Unmarshal(data, &walletData); err != nil {
		fmt.Printf("Error unmarshaling wallet data: %v\n", err)
		return fmt.Errorf("failed to unmarshal wallet data: %w", err)
	}
	fmt.Println("Wallet data unmarshaled successfully")

	m.wallet = &walletData.Wallet
	m.utxos = walletData.UTXOs
	m.scanHeight = walletData.LastHeight

	if err := m.setupScanner(); err != nil {
		return err
	}

	fmt.Printf("Wallet loaded: Network=%s, UTXOs=%d, ScanHeight=%d\n",
		m.wallet.Network, len(m.utxos), m.scanHeight)

	return nil
}

// GetAddress returns the current wallet address
func (m *Manager) GetAddress() (string, error) {
	fmt.Println("Starting GetAddress...")
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.wallet == nil {
		fmt.Println("No wallet loaded")
		return "", fmt.Errorf("no wallet loaded")
	}

	// Create address using BIP352
	network := m.wallet.Network
	mainnet := network == NetworkMainnet
	fmt.Printf("Network: %s, Mainnet: %v\n", network, mainnet)

	// Get scan and spend secrets
	fmt.Println("Getting scan and spend secrets...")
	scanSecret := [32]byte{}
	copy(scanSecret[:], m.wallet.ScanSecret)
	spendSecret := [32]byte{}
	copy(spendSecret[:], m.wallet.SpendSecret)

	// Derive scan public key from scan secret
	fmt.Println("Deriving scan public key...")
	_, scanPubKey := btcec.PrivKeyFromBytes(scanSecret[:])
	scanPubKeyBytes := scanPubKey.SerializeCompressed()
	fmt.Println("Scan public key derived")

	// Derive spend public key from spend secret
	fmt.Println("Deriving spend public key...")
	_, spendPubKey := btcec.PrivKeyFromBytes(spendSecret[:])
	spendPubKeyBytes := spendPubKey.SerializeCompressed()
	fmt.Println("Spend public key derived")

	// Create address
	fmt.Println("Creating BIP352 address...")
	address, err := bip352.CreateAddress(bip352.ConvertToFixedLength33(scanPubKeyBytes), bip352.ConvertToFixedLength33(spendPubKeyBytes), mainnet, 0)
	if err != nil {
		fmt.Printf("Error creating address: %v\n", err)
		return "", fmt.Errorf("failed to create address: %w", err)
	}

	fmt.Printf("Address created successfully: %s\n", address)
	return address, nil
}

// GetUTXOs returns all UTXOs from the scan wallet
func (m *Manager) GetUTXOs() ([]UTXO, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// For now, return stored UTXOs
	// TODO: Integrate with actual blindbit-scan scanning
	return m.utxos, nil
}

// SendTransaction sends a transaction
func (m *Manager) SendTransaction(address string, amount int64, feeRate int64) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.wallet == nil {
		return "", fmt.Errorf("no wallet loaded")
	}

	// For now, return a mock transaction ID
	// TODO: Integrate with actual blindbit-wallet-cli sending functionality
	return "mock_txid_1234567890abcdef", nil
}

// StartScanning starts the scanning process
func (m *Manager) StartScanning() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.wallet == nil {
		return fmt.Errorf("no wallet loaded")
	}

	if m.scanner == nil {
		return fmt.Errorf("scanner not initialized")
	}

	// Check if scanner is already scanning
	if m.scanner.IsScanning() {
		return fmt.Errorf("scanning already in progress")
	}

	m.logger.Info().Msg("starting scan process")

	// Start the scanner (this launches the scanning goroutine)
	if err := m.scanner.Start(); err != nil {
		return fmt.Errorf("failed to start scanner: %w", err)
	}

	return nil
}

// StopScanning stops the scanning process
func (m *Manager) StopScanning() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.scanner == nil {
		return
	}

	m.logger.Info().Int64("current_height", m.scanHeight).Msg("stopping scan")
	m.scanner.Stop()
	m.logger.Info().Msg("stop signal sent to scanner")

	// Update UTXOs from scanner after stopping
	go func() {
		// Wait a bit for scanner to finish
		time.Sleep(2 * time.Second)
		m.UpdateUTXOsFromScanner()
	}()
}

// IsScanning returns whether scanning is currently active
func (m *Manager) IsScanning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.scanner == nil {
		return false
	}

	return m.scanner.IsScanning()
}

// UpdateScanHeight updates the scan height (for real-time UI updates)
func (m *Manager) UpdateScanHeight(height int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if height > m.scanHeight {
		m.scanHeight = height
	}
}

// GetScanHeight returns the current scan height
func (m *Manager) GetScanHeight() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.scanHeight
}

// saveWalletConfig saves the wallet configuration
func (m *Manager) saveWalletConfig() error {
	if m.wallet == nil {
		return fmt.Errorf("no wallet to save")
	}

	// Create wallet data
	walletData := WalletData{
		Wallet:     *m.wallet,
		UTXOs:      m.utxos,
		LastHeight: m.scanHeight,
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(walletData, "", "  ")
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
	fmt.Println("Checking if wallet exists...")
	walletPath := filepath.Join(m.dataDir, "wallet.json")
	fmt.Printf("Wallet path: %s\n", walletPath)
	_, err := os.Stat(walletPath)
	exists := err == nil
	fmt.Printf("Wallet exists: %v\n", exists)
	return exists
}

// GetBalance calculates the total balance from unspent UTXOs
func (m *Manager) GetBalance() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var balance uint64
	var unspentCount int
	for _, utxo := range m.utxos {
		if utxo.State == "unspent" {
			balance += utxo.Amount
			unspentCount++
		}
	}
	// Removed debug logging to reduce log noise
	return balance
}

// SetNetwork changes the network and refreshes the wallet
func (m *Manager) SetNetwork(network Network) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update config
	m.config.Set("network", string(network))

	// Save config
	if err := m.config.WriteConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// If wallet exists, recreate it with new network
	if m.wallet != nil {
		// Store current mnemonic
		mnemonic := m.wallet.Mnemonic

		// Recreate wallet with new network (without acquiring lock again)
		if err := m.createWalletInternal(mnemonic, network); err != nil {
			return fmt.Errorf("failed to recreate wallet with new network: %w", err)
		}
	}

	return nil
}

// GetNetwork returns the current network
func (m *Manager) GetNetwork() Network {
	return Network(m.config.GetString("network"))
}

// SetOracleURL sets the oracle server URL
func (m *Manager) SetOracleURL(url string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config.Set("oracle_url", url)
	return m.config.WriteConfig()
}

// GetOracleURL returns the current oracle server URL
func (m *Manager) GetOracleURL() string {
	return m.config.GetString("oracle_url")
}

// SetElectrumURL sets the electrum server URL
func (m *Manager) SetElectrumURL(url string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config.Set("electrum_url", url)
	return m.config.WriteConfig()
}

// GetElectrumURL returns the current electrum server URL
func (m *Manager) GetElectrumURL() string {
	return m.config.GetString("electrum_url")
}

// SetUseTor sets whether to use Tor
func (m *Manager) SetUseTor(useTor bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config.Set("use_tor", useTor)
	return m.config.WriteConfig()
}

// GetUseTor returns whether Tor is enabled
func (m *Manager) GetUseTor() bool {
	return m.config.GetBool("use_tor")
}

// SetDustLimit sets the dust limit
func (m *Manager) SetDustLimit(limit uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config.Set("dust_limit", limit)
	return m.config.WriteConfig()
}

// GetDustLimit returns the current dust limit
func (m *Manager) GetDustLimit() uint64 {
	return m.config.GetUint64("dust_limit")
}

// SetLabelCount sets the label count
func (m *Manager) SetLabelCount(count int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config.Set("label_count", count)
	return m.config.WriteConfig()
}

// GetLabelCount returns the current label count
func (m *Manager) GetLabelCount() int {
	return m.config.GetInt("label_count")
}

// SetBirthHeight sets the birth height
func (m *Manager) SetBirthHeight(height uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config.Set("birth_height", height)
	return m.config.WriteConfig()
}

// GetBirthHeight returns the current birth height
func (m *Manager) GetBirthHeight() uint64 {
	return m.config.GetUint64("birth_height")
}

// UpdateUTXOsFromScanner updates the manager's UTXOs from the scanner's found UTXOs
func (m *Manager) UpdateUTXOsFromScanner() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.scanner == nil {
		return
	}

	// Get UTXOs from scanner
	scannerUTXOs := m.scanner.GetAllOwnedUTXOs()

	// Convert to GUI format and merge with existing UTXOs
	oldCount := len(m.utxos)
	newUTXOs := convertOwnedUTXOsToGUI(scannerUTXOs)

	// Create a map of existing UTXOs for efficient lookup
	existingUTXOs := make(map[string]UTXO)
	for _, utxo := range m.utxos {
		key := fmt.Sprintf("%s:%d", utxo.TxID, utxo.Vout)
		existingUTXOs[key] = utxo
	}

	// Merge new UTXOs with existing ones, updating existing ones
	for _, newUTXO := range newUTXOs {
		// todo: optimize to byte array key
		key := fmt.Sprintf("%s:%d", newUTXO.TxID, newUTXO.Vout)
		existingUTXOs[key] = newUTXO
	}

	// Convert back to slice
	m.utxos = make([]UTXO, 0, len(existingUTXOs))
	for _, utxo := range existingUTXOs {
		m.utxos = append(m.utxos, utxo)
	}

	// Only log if there are actual changes
	if oldCount != len(m.utxos) {
		m.logger.Info().
			Int("old_count", oldCount).
			Int("new_count", len(m.utxos)).
			Msg("merged UTXOs from scanner")
	}

	// Save to disk
	if err := m.saveWalletConfig(); err != nil {
		m.logger.Error().Err(err).Msg("failed to save wallet config after UTXO update")
	}
}

// Add this helper method to Manager:
func (m *Manager) setupScanner() error {
	oracleURL := m.config.GetString("oracle_url")
	electrumURL := m.config.GetString("electrum_url")
	birthHeight := m.config.GetUint64("birth_height")
	labelCount := m.config.GetInt("label_count") // Get label count from config

	m.logger.Info().
		Str("oracle_url", oracleURL).
		Str("electrum_url", electrumURL).
		Uint64("birth_height", birthHeight).
		Int("label_count", labelCount).
		Int64("saved_scan_height", m.scanHeight).
		Msg("setting up scanner")

	scanner, err := NewScanner(oracleURL, electrumURL, m.wallet, &m.logger, labelCount)
	if err != nil {
		return fmt.Errorf("failed to create scanner: %w", err)
	}

	// Set birth height from config
	scanner.SetBirthHeight(birthHeight)

	// Set last scan height from saved data
	if m.scanHeight > 0 {
		scanner.SetLastScanHeight(uint64(m.scanHeight))
		m.logger.Info().
			Int64("last_scan_height", m.scanHeight).
			Msg("restored last scan height")
	}

	// Load existing UTXOs into the scanner
	if len(m.utxos) > 0 {
		existingOwnedUTXOs := convertGUIUTXOsToOwned(m.utxos)
		scanner.LoadExistingUTXOs(existingOwnedUTXOs)
		m.logger.Info().
			Int("utxos_loaded", len(existingOwnedUTXOs)).
			Msg("loaded existing UTXOs into scanner")
	}

	// Set up progress callback for real-time updates
	scanner.SetProgressCallback(func(height uint64) {
		m.UpdateScanHeight(int64(height))

		// Update UTXOs from scanner every 100 blocks to reduce log noise
		if height%100 == 0 {
			m.UpdateUTXOsFromScanner()
		}
	})

	m.scanner = scanner
	m.logger.Info().Msg("scanner setup completed")
	return nil
}

// Add this helper function to convert scanner UTXOs to GUI UTXOs:
func convertOwnedUTXOsToGUI(owned []*OwnedUTXO) []UTXO {
	var guiUTXOs []UTXO
	for _, u := range owned {
		// Convert label if present
		var label *Label
		if u.Label != nil {
			label = &Label{
				PubKey:  fmt.Sprintf("%x", u.Label.PubKey[:]),
				Tweak:   fmt.Sprintf("%x", u.Label.Tweak[:]),
				Address: u.Label.Address,
				M:       u.Label.M,
			}
		}

		guiUTXOs = append(guiUTXOs, UTXO{
			TxID:      fmt.Sprintf("%x", u.Txid[:]),
			Vout:      u.Vout,
			Amount:    u.Amount,
			State:     u.State.String(),
			Timestamp: int64(u.Timestamp),
			Address:   "", // Optionally fill if you have address info
			Label:     label,
		})
	}
	return guiUTXOs
}

// Add this helper function to convert GUI UTXOs to owned UTXOs:
func convertGUIUTXOsToOwned(guiUTXOs []UTXO) []*OwnedUTXO {
	var ownedUTXOs []*OwnedUTXO
	for _, u := range guiUTXOs {
		var txid [32]byte
		if len(u.TxID) >= 64 {
			// Parse hex string to bytes
			for i := 0; i < 32; i++ {
				if i*2+1 < len(u.TxID) {
					// Simple hex parsing - in production you might want to use encoding/hex
					high := u.TxID[i*2]
					low := u.TxID[i*2+1]
					txid[i] = hexCharToByte(high)<<4 | hexCharToByte(low)
				}
			}
		}

		state := StateUnspent
		switch u.State {
		case "spent":
			state = StateSpent
		case "unconfirmed_spent":
			state = StateUnconfirmedSpent
		case "unconfirmed":
			state = StateUnconfirmed
		}

		// Convert label if present
		var label *bip352.Label
		if u.Label != nil {
			// Parse pubkey and tweak from hex strings
			var pubKeyBytes, tweakBytes []byte
			if len(u.Label.PubKey) >= 66 { // 33 bytes = 66 hex chars
				pubKeyBytes = make([]byte, 33)
				for i := 0; i < 33; i++ {
					if i*2+1 < len(u.Label.PubKey) {
						high := u.Label.PubKey[i*2]
						low := u.Label.PubKey[i*2+1]
						pubKeyBytes[i] = hexCharToByte(high)<<4 | hexCharToByte(low)
					}
				}
			}
			if len(u.Label.Tweak) >= 64 { // 32 bytes = 64 hex chars
				tweakBytes = make([]byte, 32)
				for i := 0; i < 32; i++ {
					if i*2+1 < len(u.Label.Tweak) {
						high := u.Label.Tweak[i*2]
						low := u.Label.Tweak[i*2+1]
						tweakBytes[i] = hexCharToByte(high)<<4 | hexCharToByte(low)
					}
				}
			}

			if len(pubKeyBytes) == 33 && len(tweakBytes) == 32 {
				label = &bip352.Label{
					PubKey:  bip352.ConvertToFixedLength33(pubKeyBytes),
					Tweak:   bip352.ConvertToFixedLength32(tweakBytes),
					Address: u.Label.Address,
					M:       u.Label.M,
				}
			}
		}

		ownedUTXOs = append(ownedUTXOs, &OwnedUTXO{
			Txid:      txid,
			Vout:      u.Vout,
			Amount:    u.Amount,
			Timestamp: uint64(u.Timestamp),
			State:     state,
			Label:     label,
		})
	}
	return ownedUTXOs
}

// Helper function to convert hex character to byte
func hexCharToByte(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 0
	}
}

// RefreshUTXOs manually refreshes UTXOs from the scanner
func (m *Manager) RefreshUTXOs() {
	m.UpdateUTXOsFromScanner()
}

// ClearUTXOs manually clears all UTXOs (use with caution)
func (m *Manager) ClearUTXOs() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.utxos = []UTXO{}

	if m.scanner != nil {
		m.scanner.ClearUTXOs()
	}

	// Save the cleared state
	if err := m.saveWalletConfig(); err != nil {
		fmt.Printf("Error saving wallet config after clearing UTXOs: %v\n", err)
	} else {
		fmt.Println("Successfully cleared all UTXOs")
	}
}

// GetUTXOStats returns statistics about the current UTXOs
func (m *Manager) GetUTXOStats() (total int, unspent int, spent int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total = len(m.utxos)
	for _, utxo := range m.utxos {
		switch utxo.State {
		case "unspent":
			unspent++
		case "spent":
			spent++
		}
	}
	return total, unspent, spent
}

// RescanFromHeight resets the last scanned height and triggers a rescan from the specified height
func (m *Manager) RescanFromHeight(height uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.wallet == nil {
		return fmt.Errorf("no wallet loaded")
	}

	if m.scanner == nil {
		return fmt.Errorf("scanner not initialized")
	}

	// Stop current scanning if running
	if m.scanner.IsScanning() {
		fmt.Println("[RescanFromHeight] Stopping current scan before rescan")
		m.scanner.StopSync() // Use synchronous stop
	}

	// Update the scan height
	oldHeight := m.scanHeight
	m.scanHeight = int64(height - 1) // Set to height-1 so scanning starts from the specified height

	fmt.Printf("[RescanFromHeight] Resetting scan height from %d to %d\n", oldHeight, m.scanHeight)

	// Update the scanner's last scan height
	m.scanner.SetLastScanHeight(height - 1)

	// Save the updated scan height
	if err := m.saveWalletConfig(); err != nil {
		return fmt.Errorf("failed to save wallet config: %w", err)
	}

	fmt.Printf("[RescanFromHeight] Successfully reset scan height to %d\n", height)
	return nil
}

// ForceRescanFromHeight performs a complete rescan from the specified height, clearing existing UTXOs
func (m *Manager) ForceRescanFromHeight(height uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.wallet == nil {
		return fmt.Errorf("no wallet loaded")
	}

	if m.scanner == nil {
		return fmt.Errorf("scanner not initialized")
	}

	// Stop current scanning if running
	if m.scanner.IsScanning() {
		fmt.Println("[ForceRescanFromHeight] Stopping current scan before force rescan")
		m.scanner.StopSync() // Use synchronous stop
	}

	// Clear all existing UTXOs
	oldUTXOCount := len(m.utxos)
	m.utxos = []UTXO{}
	m.scanner.ClearUTXOs()

	fmt.Printf("[ForceRescanFromHeight] Cleared %d existing UTXOs\n", oldUTXOCount)

	// Update the scan height
	oldHeight := m.scanHeight
	m.scanHeight = int64(height - 1) // Set to height-1 so scanning starts from the specified height

	fmt.Printf("[ForceRescanFromHeight] Resetting scan height from %d to %d\n", oldHeight, m.scanHeight)

	// Update the scanner's last scan height
	m.scanner.SetLastScanHeight(height - 1)

	// Save the updated state
	if err := m.saveWalletConfig(); err != nil {
		return fmt.Errorf("failed to save wallet config: %w", err)
	}

	fmt.Printf("[ForceRescanFromHeight] Successfully reset scan height to %d and cleared UTXOs\n", height)
	return nil
}

// RescanFromTip rescans from the current chain tip
func (m *Manager) RescanFromTip() error {
	if m.scanner == nil {
		return fmt.Errorf("scanner not initialized")
	}

	// Get current chain tip
	chainTip, err := m.scanner.client.GetChainTip()
	if err != nil {
		return fmt.Errorf("failed to get chain tip: %w", err)
	}

	fmt.Printf("[RescanFromTip] Rescanning from chain tip: %d\n", chainTip)
	return m.RescanFromHeight(chainTip)
}

// ForceRescanFromTip performs a complete rescan from the current chain tip, clearing existing UTXOs
func (m *Manager) ForceRescanFromTip() error {
	if m.scanner == nil {
		return fmt.Errorf("scanner not initialized")
	}

	// Get current chain tip
	chainTip, err := m.scanner.client.GetChainTip()
	if err != nil {
		return fmt.Errorf("failed to get chain tip: %w", err)
	}

	fmt.Printf("[ForceRescanFromTip] Force rescanning from chain tip: %d\n", chainTip)
	return m.ForceRescanFromHeight(chainTip)
}
