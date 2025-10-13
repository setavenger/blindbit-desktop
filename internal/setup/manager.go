package setup

import (
	"fmt"
	"os"

	"github.com/setavenger/blindbit-desktop/internal/configs"
	"github.com/setavenger/blindbit-desktop/internal/controller"
	"github.com/setavenger/blindbit-desktop/internal/storage"
	"github.com/setavenger/blindbit-lib/logging"
	"github.com/setavenger/blindbit-lib/utils"
)

// NewManagerWithDataDir creates a new wallet manager using the provided data directory.
// If dataDir is empty, it falls back to the default returned by getDataDir().
// Returns (manager, exists, error) where exists indicates if the wallet file already existed.
func NewManagerWithDataDir(dataDir string) (*controller.Manager, bool, error) {
	if dataDir == "" {
		dataDir = configs.DefaultDataDir()
	} else {
		dataDir = utils.ResolvePath(dataDir)
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("Error creating data directory: %v\n", err)
		return nil, false, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Check if wallet file exists
	walletPath := fmt.Sprintf("%s/wallet.dat", dataDir)
	if _, err := os.Stat(walletPath); os.IsNotExist(err) {
		// Wallet doesn't exist, return nil manager but no error
		return nil, false, nil
	}

	manager, err := storage.LoadPlain(dataDir)
	if err != nil {
		logging.L.Err(err).Msg("failed to load manager")
		return nil, true, err
	}

	return manager, true, nil
}
