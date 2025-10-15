// Package storage handles the saving and retrieving of the applications data
package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/setavenger/blindbit-desktop/internal/controller"
	"github.com/setavenger/blindbit-lib/logging"
	"github.com/setavenger/blindbit-lib/wallet"
)

const walletDataFilename = "wallet.dat"

func SavePlain(datadir string, m *controller.Manager) error {
	logging.L.Trace().Str("datadir", datadir).Msg("saving wallet")
	binaryData, err := m.Serialise()
	if err != nil {
		logging.L.Err(err).
			Str("datadir", datadir).
			Msg("failed to serialise wallet data")
		return err
	}

	// Write to file
	walletPath := filepath.Join(datadir, walletDataFilename)

	if err := os.WriteFile(walletPath, binaryData, 0600); err != nil {
		logging.L.Err(err).
			Str("datadir", datadir).
			Str("path", walletPath).
			Msg("failed to write wallet file")
		return fmt.Errorf("failed to write wallet file: %w", err)
	}

	logging.L.Info().Str("path", walletPath).Msg("successfully wrote wallet file")
	return nil
}

func LoadPlain(datadir string) (m *controller.Manager, err error) {
	logging.L.Trace().Str("datadir", datadir).Msg("loading wallet")
	data, err := os.ReadFile(filepath.Join(datadir, walletDataFilename))
	if err != nil {
		logging.L.Err(err).
			Str("datadir", datadir).
			Str("path", filepath.Join(datadir, walletDataFilename)).
			Msg("failed to load wallet file")
		return nil, err
	}

	m = new(controller.Manager)
	m.Wallet = wallet.InitWallet()
	err = m.DeSerialise(data)
	if err != nil {
		logging.L.Err(err).
			Str("datadir", datadir).
			Str("path", filepath.Join(datadir, walletDataFilename)).
			Msg("failed to deserialise wallet data")
		return nil, err
	}

	logging.L.Info().Str("datadir", datadir).Msg("successfully loaded wallet")
	return m, err
}
