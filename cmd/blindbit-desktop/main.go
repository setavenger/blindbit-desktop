package main

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"github.com/rs/zerolog"

	"github.com/setavenger/blindbit-desktop/internal/configs"
	"github.com/setavenger/blindbit-desktop/internal/controller"
	"github.com/setavenger/blindbit-desktop/internal/gui"
	"github.com/setavenger/blindbit-desktop/internal/setup"
	"github.com/setavenger/blindbit-desktop/internal/storage"
	"github.com/setavenger/blindbit-lib/logging"
	"github.com/setavenger/blindbit-lib/utils"
	"github.com/spf13/pflag"
)

var dataDir string

func init() {
	var debug bool
	pflag.BoolVar(&debug, "debug", false, "enable debug logging")
	pflag.StringVar(&dataDir, "datadir", "", "path to data directory for BlindBit Desktop")
	pflag.Parse()

	if debug {
		logging.SetLogLevel(zerolog.TraceLevel)
	} else {
		logging.SetLogLevel(zerolog.InfoLevel)
	}
}

func main() {
	// Create a new Fyne application
	myApp := app.New()

	myApp.SetIcon(myApp.Icon()) // You can replace this with a custom icon

	// Create the main window
	mainWindow := myApp.NewWindow("BlindBit Desktop")

	mainWindow.Resize(fyne.NewSize(1000, 750))
	mainWindow.CenterOnScreen()

	// Try to load existing wallet manager
	walletManager, exists, err := setup.NewManagerWithDataDir(dataDir)
	if err != nil {
		logging.L.Err(err).Msg("Failed to load existing wallet manager")
		// Show error dialog and exit
		dialog.ShowError(fmt.Errorf("failed to load wallet: %v", err), mainWindow)
		return
	}

	// Get the resolved data directory for consistency
	resolvedDataDir := dataDir
	if resolvedDataDir == "" {
		resolvedDataDir = configs.DefaultDataDir()
	} else {
		resolvedDataDir = utils.ResolvePath(resolvedDataDir)
	}

	if !exists {
		// No wallet exists, show setup wizard
		setupWizard := gui.NewSetupWizard(
			myApp, mainWindow, resolvedDataDir, func(manager *controller.Manager) {
				// Initialize scanner before showing main GUI
				if err := manager.ConstructScanner(context.TODO()); err != nil {
					logging.L.Err(err).Msg("failed to construct scanner after setup")
					dialog.ShowError(fmt.Errorf("failed to construct scanner: %v", err), mainWindow)
					return
				}

				// Start channel handling and background scanning
				manager.StartChannelHandling(context.TODO(), func() error {
					return storage.SavePlain(manager.DataDir, manager)
				})

				go func() {
					watchStartHeight := manager.Wallet.LastScanHeight
					if watchStartHeight == 0 {
						watchStartHeight = manager.Wallet.BirthHeight
					}
					if err := manager.Scanner.Watch(context.TODO(), uint32(watchStartHeight)); err != nil {
						logging.L.Err(err).Msg("failed to watch scanner")
						dialog.ShowError(fmt.Errorf("failed to watch scanner: %v", err), mainWindow)
						return
					}
				}()

				walletManager = manager
				// Setup completed, show main GUI
				mainGUI := gui.NewMainGUI(myApp, mainWindow, manager)
				mainWindow.SetContent(mainGUI.GetContent())
			},
		)
		setupWizard.Show()
	} else {
		// Set the DataDir on the loaded manager
		walletManager.DataDir = resolvedDataDir

		// Initialize scanner before showing main GUI
		if err := walletManager.ConstructScanner(context.TODO()); err != nil {
			logging.L.Err(err).Msg("failed to construct scanner")
			dialog.ShowError(fmt.Errorf("failed to construct scanner: %v", err), mainWindow)
			return
		}

		// Start channel handling and background scanning
		walletManager.StartChannelHandling(context.TODO(), func() error {
			return storage.SavePlain(walletManager.DataDir, walletManager)
		})

		go func() {
			err := walletManager.Scanner.Watch(
				context.TODO(), uint32(walletManager.Wallet.LastScanHeight),
			)
			if err != nil {
				logging.L.Err(err).Msg("failed to watch scanner")
				dialog.ShowError(fmt.Errorf("failed to watch scanner: %v", err), mainWindow)
				return
			}
		}()

		// Wallet loaded successfully, show main GUI
		mainGUI := gui.NewMainGUI(myApp, mainWindow, walletManager)
		mainWindow.SetContent(mainGUI.GetContent())
	}

	if walletManager != nil {
		defer storage.SavePlain(walletManager.DataDir, walletManager)
	}

	// Show and run the application
	mainWindow.ShowAndRun()
}
