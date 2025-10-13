package controller

import (
	"sort"

	"github.com/setavenger/blindbit-lib/types"
	"github.com/setavenger/blindbit-lib/wallet"
)

/* Helpers to avoid chaining to deep into sub structs */

func (m *Manager) GetNetwork() types.Network {
	return m.Wallet.Network
}

// GetSilentPaymentAddress returns the main Silent Payment address
// Note: This is NOT the change address (label 0), but the main receiving address
func (m *Manager) GetSilentPaymentAddress() string {
	// TODO: Implement getting the main SP address from wallet
	// This would typically involve getting the address without any label
	return "sp1..." // Placeholder
}

// GetUTXOsSorted returns UTXOs sorted by block height (newest first)
func (m *Manager) GetUTXOsSorted() []*wallet.OwnedUTXO {
	utxos := m.Wallet.GetUTXOs()

	// Sort by height in descending order (newest first)
	sort.Slice(utxos, func(i, j int) bool {
		return utxos[i].Height > utxos[j].Height
	})

	return utxos
}
