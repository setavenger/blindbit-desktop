package manager

import (
	"fmt"

	"github.com/setavenger/blindbit-lib/types"
)

// SetNetwork changes the network and refreshes the wallet
func (m *Manager) SetNetwork(network types.Network) error {
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
func (m *Manager) GetNetwork() types.Network {
	return types.Network(m.config.GetString("network"))
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
