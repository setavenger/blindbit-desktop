package controller

import (
	"context"

	"github.com/setavenger/blindbit-lib/wallet"
)

func (m *Manager) PrepareTransaction(ctx context.Context, recipients []wallet.Recipient) (*wallet.TxMetadata, error) {
	tx, err := m.Wallet.SendToRecipients(
		recipients,
		m.Wallet.GetUTXOs(),
		int64(m.MinChangeAmount),
		uint64(m.DustLimit),
		false,
		false,
	)
	if err != nil {
		return nil, err
	}

	return tx, nil
}
