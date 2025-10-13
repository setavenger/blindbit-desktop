package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"github.com/rs/zerolog"

	"github.com/setavenger/blindbit-desktop/internal/configs"
	"github.com/setavenger/blindbit-desktop/internal/controller"
	"github.com/setavenger/blindbit-desktop/internal/gui"
	"github.com/setavenger/blindbit-desktop/internal/setup"
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
		logging.SetLogLevel(zerolog.DebugLevel)
	} else {
		logging.SetLogLevel(zerolog.InfoLevel)
	}
}

func main() {
	// Create a new Fyne application
	myApp := app.New()

	myApp.SetIcon(theme.AccountIcon()) // You can replace this with a custom icon

	// Create the main window
	mainWindow := myApp.NewWindow("BlindBit Desktop")

	mainWindow.Resize(fyne.NewSize(800, 600))
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
		setupWizard := gui.NewSetupWizard(myApp, mainWindow, resolvedDataDir, func(manager *controller.Manager) {
			// Setup completed, show main GUI
			mainGUI := gui.NewMainGUI(myApp, mainWindow, manager)
			mainWindow.SetContent(mainGUI.GetContent())
		})
		setupWizard.Show()
	} else {
		// Wallet loaded successfully, show main GUI
		mainGUI := gui.NewMainGUI(myApp, mainWindow, walletManager)
		mainWindow.SetContent(mainGUI.GetContent())
	}

	// Show and run the application
	mainWindow.ShowAndRun()
}
