package gui

import (
	"os"
	"os/exec"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/setavenger/blindbit-desktop/internal/controller"
	"github.com/setavenger/blindbit-desktop/internal/storage"
	"github.com/setavenger/blindbit-lib/logging"
)

type MainGUI struct {
	app     fyne.App
	window  fyne.Window
	manager *controller.Manager
	tabs    *container.AppTabs
}

func NewMainGUI(
	app fyne.App,
	window fyne.Window,
	manager *controller.Manager,
) *MainGUI {
	gui := &MainGUI{
		app:     app,
		window:  window,
		manager: manager,
	}

	gui.setupTabs()
	return gui
}

func (g *MainGUI) setupTabs() {
	g.tabs = container.NewAppTabs(
		container.NewTabItem("Scanning", g.createScanningTab()),
		container.NewTabItem("UTXOs", g.createUTXOsTab()),
		container.NewTabItem("Send", g.createSendTab()),
		container.NewTabItem("Receive", g.createReceiveTab()),
		container.NewTabItem("Transactions", g.createTransactionsTab()),
		container.NewTabItem("Settings", g.createSettingsTab()),
	)
}

func (g *MainGUI) GetContent() fyne.CanvasObject {
	return g.tabs
}

// CleanupAndExit exits the program with status 0
// Before that it:
// - saves the data to file
func (g *MainGUI) CleanupAndExit() {
	// Simple cleanup - avoid any operations that might cause issues during shutdown
	// The app will handle most cleanup automatically
	if err := storage.SavePlain(g.manager.DataDir, g.manager); err != nil {
		logging.L.Err(err).Msg("error during shutdown")
		os.Exit(1)
	}

	os.Exit(0)
}

// restartApplication attempts to re-execute the current binary with the same arguments,
// then exits the current process.
//
// Deprecated: Discoured use.
// tends to have problems when application is called from the terminal
func (g *MainGUI) restartApplication() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	// Preserve original args (excluding the current process name; os.Args[0] is the path)
	args := os.Args[1:]

	cmd := exec.Command(exePath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Start the new process and exit the current one
	if err := cmd.Start(); err != nil {
		return err
	}
	// Close the current app window and exit
	if g.window != nil {
		g.window.Close()
	}

	// Cleanup and Exit
	g.CleanupAndExit()
	return nil
}
