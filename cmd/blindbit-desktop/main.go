package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
	"github.com/rs/zerolog"
	"github.com/setavenger/blindbit-desktop/internal/gui"

	// "github.com/setavenger/blindbit-desktop/internal/manager"
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

	// Initialize wallet manager
	// walletManager, err := manager.NewManagerWithDataDir(dataDir)
	// if err != nil {
	// 	logging.L.Err(err).Msg("Failed to initialize wallet manager")
	// 	// Show error dialog
	// 	errorDialog := widget.NewModalPopUp(
	// 		widget.NewLabel("Failed to initialize wallet manager: "+err.Error()),
	// 		mainWindow.Canvas(),
	// 	)
	// 	errorDialog.Resize(fyne.NewSize(400, 100))
	// 	errorDialog.Show()
	// }

	// Create the main GUI
	mainGUI := gui.NewMainGUI(myApp, mainWindow, walletManager)

	// Set the main content
	mainWindow.SetContent(mainGUI.GetContent())

	// Set up cleanup when window is closed
	mainWindow.SetOnClosed(func() {
		mainGUI.Cleanup()
	})

	// Show and run the application
	mainWindow.ShowAndRun()
}
