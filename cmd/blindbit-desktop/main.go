package main

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/rs/zerolog"

	"github.com/setavenger/blindbit-desktop/internal/controller"
	"github.com/setavenger/blindbit-desktop/internal/gui"
	"github.com/setavenger/blindbit-desktop/internal/setup"
	"github.com/setavenger/blindbit-desktop/internal/storage"
	"github.com/setavenger/blindbit-lib/logging"
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
	myApp := app.New()
	myApp.SetIcon(myApp.Icon())

	mainWindow := myApp.NewWindow("BlindBit Desktop")
	mainWindow.Resize(fyne.NewSize(1000, 750))
	mainWindow.CenterOnScreen()

	// Resolve and create the data directory; check if a wallet already exists.
	resolvedDataDir, walletExists, err := setup.PrepareDataDir(dataDir)
	if err != nil {
		logging.L.Err(err).Msg("failed to prepare data directory")
		dialog.ShowError(fmt.Errorf("failed to prepare data directory: %v", err), mainWindow)
		return
	}

	if err = logging.EnableFileLogging(resolvedDataDir, "debug.log"); err != nil {
		fmt.Println("base_dir:", resolvedDataDir)
		logging.L.Fatal().Err(err).Msg("error setting log file")
	}

	// walletManager and sessionPassword are set once the user completes setup or
	// unlock. The deferred save uses them to write an encrypted wallet on exit.
	var walletManager *controller.Manager
	var sessionPassword []byte

	defer func() {
		if walletManager != nil {
			if err := storage.SaveWithPassword(walletManager.DataDir, walletManager, sessionPassword); err != nil {
				logging.L.Err(err).Msg("failed to save wallet on exit")
			}
		}
	}()

	// startMainGUI wires up the scanner and shows the main window. It is called
	// from both the setup-wizard callback and the unlock-screen callback.
	startMainGUI := func(manager *controller.Manager, password []byte) {
		sessionPassword = password
		walletManager = manager
		walletManager.DataDir = resolvedDataDir

		if err := manager.ConstructScanner(context.TODO()); err != nil {
			logging.L.Err(err).Msg("failed to construct scanner")
			dialog.ShowError(fmt.Errorf("failed to construct scanner: %v", err), mainWindow)
			return
		}

		manager.StartChannelHandling(context.TODO(), func() error {
			return storage.SaveWithPassword(manager.DataDir, manager, password)
		})

		go func() {
			watchStartHeight := manager.Wallet.LastScanHeight
			if watchStartHeight == 0 {
				watchStartHeight = manager.Wallet.BirthHeight
			}
			if err := manager.Scanner.Watch(context.TODO(), uint32(watchStartHeight)); err != nil {
				logging.L.Err(err).Msg("failed to watch scanner")
				dialog.ShowError(fmt.Errorf("failed to watch scanner: %v", err), mainWindow)
			}
		}()

		mainGUI := gui.NewMainGUI(myApp, mainWindow, manager, password)
		mainWindow.SetContent(mainGUI.GetContent())
		// Re-apply the main window size after SetContent so layout changes made
		// during setup/unlock do not leave the window at the wrong dimensions.
		mainWindow.Resize(fyne.NewSize(1000, 750))
	}

	if !walletExists {
		setupWizard := gui.NewSetupWizard(
			myApp, mainWindow, resolvedDataDir,
			func(manager *controller.Manager, password []byte) {
				startMainGUI(manager, password)
			},
		)
		setupWizard.Show()
	} else {
		unlockScreen := gui.NewUnlockScreen(
			myApp, mainWindow, resolvedDataDir,
			func(password []byte) {
				manager, _, err := setup.NewManagerWithDataDir(resolvedDataDir, password)
				if err != nil {
					logging.L.Err(err).Msg("failed to load wallet after unlock")
					dialog.ShowError(fmt.Errorf("failed to load wallet: %v", err), mainWindow)
					return
				}
				startMainGUI(manager, password)
			},
		)
		unlockScreen.Show()
	}

	// Tray settings
	if desk, ok := myApp.(desktop.App); ok {
		logging.L.Debug().Msg("Is desktop app")
		m := fyne.NewMenu("BlindBit",
			fyne.NewMenuItem("Show", func() {
				mainWindow.Show()
			}))
		desk.SetSystemTrayMenu(m)
		if myApp.Metadata().Icon != nil {
			logging.L.Trace().Msg("have metadata icon")
			desk.SetSystemTrayIcon(myApp.Metadata().Icon)
		} else {
			logging.L.Trace().Msg("failed to find metadata icon")
		}
	}

	mainWindow.SetCloseIntercept(func() { mainWindow.Hide() })

	mainWindow.ShowAndRun()
}
