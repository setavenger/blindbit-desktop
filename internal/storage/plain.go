// Package storage handles the saving and retrieving of the applications data
package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/setavenger/blindbit-desktop/internal/controller"
	"github.com/setavenger/blindbit-lib/logging"
)

const walletDataFilename = "wallet.dat"

func SavePlain(datadir string, m *controller.Manager) error {
	binaryData, err := m.Serialise()
	if err != nil {
		logging.L.Err(err).Msg("failed to serialise wallet data")
		return err
	}

	// Write to file
	walletPath := filepath.Join(datadir, walletDataFilename)
	if err := os.WriteFile(walletPath, binaryData, 0600); err != nil {
		return fmt.Errorf("failed to write wallet file: %w", err)
	}

	return nil
}

func LoadPlain(datadir string) (m *controller.Manager, err error) {

	data, err := os.ReadFile(filepath.Join(datadir, walletDataFilename))
	if err != nil {
		logging.L.Err(err).Msg("failed to load wallet file")
		return nil, err
	}

	m = new(controller.Manager)
	err = m.DeSerialise(data)
	if err != nil {
		logging.L.Err(err).Msg("failed to deserialise wallet data")
		return nil, err
	}

	return m, err
}
