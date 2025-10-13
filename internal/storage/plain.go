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
	fmt.Printf("SavePlain called with datadir: %s\n", datadir)
	
	binaryData, err := m.Serialise()
	if err != nil {
		logging.L.Err(err).Msg("failed to serialise wallet data")
		return err
	}
	fmt.Printf("Serialized data length: %d bytes\n", len(binaryData))

	// Write to file
	walletPath := filepath.Join(datadir, walletDataFilename)
	fmt.Printf("Writing to file: %s\n", walletPath)
	
	if err := os.WriteFile(walletPath, binaryData, 0600); err != nil {
		fmt.Printf("WriteFile error: %v\n", err)
		return fmt.Errorf("failed to write wallet file: %w", err)
	}
	
	fmt.Printf("Successfully wrote wallet file\n")
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
