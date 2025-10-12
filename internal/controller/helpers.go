package controller

import "github.com/setavenger/blindbit-lib/types"

/* Helpers to avoid chaining to deep into sub structs */

func (m *Manager) GetNetwork() types.Network {
	return m.Wallet.Network
}
