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

// Save encrypts the manager state with password and writes it to wallet.dat.
func Save(datadir string, m *controller.Manager, password []byte) error {
	logging.L.Trace().Str("datadir", datadir).Msg("saving encrypted wallet")
	plaintext, err := m.Serialise()
	if err != nil {
		logging.L.Err(err).Str("datadir", datadir).Msg("failed to serialise wallet data")
		return err
	}

	blob, err := Encrypt(plaintext, password)
	if err != nil {
		logging.L.Err(err).Str("datadir", datadir).Msg("failed to encrypt wallet data")
		return fmt.Errorf("failed to encrypt wallet: %w", err)
	}

	walletPath := filepath.Join(datadir, walletDataFilename)
	if err := os.WriteFile(walletPath, blob, 0600); err != nil {
		logging.L.Err(err).
			Str("datadir", datadir).
			Str("path", walletPath).
			Msg("failed to write encrypted wallet file")
		return fmt.Errorf("failed to write wallet file: %w", err)
	}

	logging.L.Info().Str("path", walletPath).Msg("successfully wrote encrypted wallet file")
	return nil
}

// Load reads wallet.dat, decrypts it with password, and deserialises the manager.
func Load(datadir string, password []byte) (*controller.Manager, error) {
	logging.L.Trace().Str("datadir", datadir).Msg("loading encrypted wallet")
	data, err := os.ReadFile(filepath.Join(datadir, walletDataFilename))
	if err != nil {
		logging.L.Err(err).Str("datadir", datadir).Msg("failed to read wallet file")
		return nil, err
	}

	plaintext, err := Decrypt(data, password)
	if err != nil {
		return nil, err
	}

	m := new(controller.Manager)
	m.Wallet = wallet.InitWallet()
	if err := m.DeSerialise(plaintext); err != nil {
		logging.L.Err(err).Str("datadir", datadir).Msg("failed to deserialise wallet data")
		return nil, err
	}

	logging.L.Info().Str("datadir", datadir).Msg("successfully loaded encrypted wallet")
	return m, nil
}

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

// SaveWithPassword saves the wallet encrypted when password is non-empty,
// or as plaintext when password is nil / empty.
func SaveWithPassword(datadir string, m *controller.Manager, password []byte) error {
	if len(password) == 0 {
		return SavePlain(datadir, m)
	}
	return Save(datadir, m, password)
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
