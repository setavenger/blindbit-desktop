package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/setavenger/blindbit-desktop/internal/controller"
)

type MainGUI struct {
	app     fyne.App
	window  fyne.Window
	manager *controller.Manager
	tabs    *container.AppTabs
}

func NewMainGUI(app fyne.App, window fyne.Window, manager *controller.Manager) *MainGUI {
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

func (g *MainGUI) Cleanup() {
	// Simple cleanup - avoid any operations that might cause issues during shutdown
	// The app will handle most cleanup automatically
}
