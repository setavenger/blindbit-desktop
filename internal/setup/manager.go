package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/setavenger/blindbit-desktop/internal/configs"
	"github.com/setavenger/blindbit-desktop/internal/controller"
	"github.com/setavenger/blindbit-desktop/internal/storage"
	"github.com/setavenger/blindbit-lib/logging"
	"github.com/setavenger/blindbit-lib/utils"
)

// PrepareDataDir resolves the data directory path, creates it if needed, and
// reports whether wallet.dat already exists. It does not load the wallet.
func PrepareDataDir(dataDir string) (resolved string, walletExists bool, err error) {
	if dataDir == "" {
		dataDir = configs.DefaultDataDir()
	} else {
		dataDir = utils.ResolvePath(dataDir)
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", false, fmt.Errorf("failed to create data directory: %w", err)
	}

	walletPath := filepath.Join(dataDir, "wallet.dat")
	_, statErr := os.Stat(walletPath)
	return dataDir, !os.IsNotExist(statErr), nil
}

// NewManagerWithDataDir loads the wallet from dataDir using password.
//
//   - If wallet.dat does not exist, returns (nil, false, nil).
//   - If wallet.dat is encrypted, decrypts it with password.
//   - If wallet.dat is plaintext (legacy), loads it and immediately re-saves it
//     encrypted with password (one-step migration).
//
// After loading, oracle settings are read from settings.json and applied.
func NewManagerWithDataDir(dataDir string, password []byte) (*controller.Manager, bool, error) {
	if dataDir == "" {
		dataDir = configs.DefaultDataDir()
	} else {
		dataDir = utils.ResolvePath(dataDir)
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, false, fmt.Errorf("failed to create data directory: %w", err)
	}

	walletPath := filepath.Join(dataDir, "wallet.dat")
	if _, err := os.Stat(walletPath); os.IsNotExist(err) {
		return nil, false, nil
	}

	raw, err := os.ReadFile(walletPath)
	if err != nil {
		return nil, true, fmt.Errorf("failed to read wallet file: %w", err)
	}

	var manager *controller.Manager

	if storage.IsEncrypted(raw) {
		manager, err = storage.Load(dataDir, password)
		if err != nil {
			logging.L.Err(err).Msg("failed to decrypt wallet")
			return nil, true, fmt.Errorf("incorrect password or corrupted wallet")
		}
	} else {
		// Legacy plaintext wallet — extract oracle settings from raw JSON before
		// they are lost (they are now json:"-" on Manager).
		var legacyOracle struct {
			OracleAddress string `json:"oracle_address"`
			OracleUseTLS  bool   `json:"oracle_use_tls"`
		}
		_ = json.Unmarshal(raw, &legacyOracle)

		manager, err = storage.LoadPlain(dataDir)
		if err != nil {
			logging.L.Err(err).Msg("failed to load plaintext wallet")
			return nil, true, err
		}

		// Restore oracle settings from the old JSON blob.
		manager.OracleAddress = legacyOracle.OracleAddress
		manager.OracleUseTLS = legacyOracle.OracleUseTLS

		// Persist oracle settings to settings.json as part of the migration.
		if saveErr := storage.SaveSettings(dataDir, &storage.Settings{
			OracleAddress: manager.OracleAddress,
			OracleUseTLS:  manager.OracleUseTLS,
		}); saveErr != nil {
			logging.L.Warn().Err(saveErr).Msg("failed to save oracle settings during migration")
		}

		if len(password) > 0 {
			if err := storage.Save(dataDir, manager, password); err != nil {
				logging.L.Err(err).Msg("failed to migrate wallet to encrypted format")
				return nil, true, fmt.Errorf("failed to migrate wallet to encrypted format: %w", err)
			}
			logging.L.Info().Msg("migrated plaintext wallet to encrypted format")
		}
	}

	// Populate oracle settings from settings.json (they are not stored in the encrypted blob).
	if settings, err := storage.LoadSettings(dataDir); err == nil && settings != nil {
		manager.OracleAddress = settings.OracleAddress
		manager.OracleUseTLS = settings.OracleUseTLS
	}

	return manager, true, nil
}
