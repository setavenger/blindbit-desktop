package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// TrayManager handles system tray functionality
type TrayManager struct {
	app     fyne.App
	window  fyne.Window
	visible bool
}

// NewTrayManager creates a new tray manager
func NewTrayManager(app fyne.App, window fyne.Window) *TrayManager {
	tm := &TrayManager{
		app:     app,
		window:  window,
		visible: true,
	}

	tm.setupTray()
	return tm
}

// setupTray initializes the system tray
func (tm *TrayManager) setupTray() {
	// Handle window close events - hide instead of close
	tm.window.SetCloseIntercept(func() {
		// Hide window instead of closing when red button is clicked
		tm.hideWindow()
	})

	// Set up system tray menu if desktop support is available
	if desk, ok := tm.app.(desktop.App); ok {
		menu := fyne.NewMenu("BlindBit",
			fyne.NewMenuItem("Show/Hide", func() {
				tm.toggleWindow()
			}),
			fyne.NewMenuItem("Quit", func() {
				tm.quitApp()
			}),
		)
		desk.SetSystemTrayMenu(menu)
	}
}

// toggleWindow shows or hides the main window
func (tm *TrayManager) toggleWindow() {
	if tm.visible {
		tm.hideWindow()
	} else {
		tm.showWindow()
	}
}

// showWindow shows the main window
func (tm *TrayManager) showWindow() {
	tm.window.Show()
	tm.window.RequestFocus() // Bring window to front on macOS
	tm.visible = true
}

// hideWindow hides the main window
func (tm *TrayManager) hideWindow() {
	tm.window.Hide()
	tm.visible = false
}

// quitApp quits the entire application
func (tm *TrayManager) quitApp() {
	tm.app.Quit()
}

// IsVisible returns whether the window is currently visible
func (tm *TrayManager) IsVisible() bool {
	return tm.visible
}

// ShowNotification shows a notification in the system tray
func (tm *TrayManager) ShowNotification(title, message string) {
	// Fyne doesn't have built-in system notifications yet
	// This could be implemented with platform-specific code if needed
}

// UpdateTrayIcon updates the tray icon with new icon data
func (tm *TrayManager) UpdateTrayIcon(iconBytes []byte) {
	// For Fyne's native system tray, we need to use the desktop.App interface
	// This is a placeholder for future implementation
}

// UpdateTrayTooltip updates the tray tooltip text
func (tm *TrayManager) UpdateTrayTooltip(tooltip string) {
	// Fyne's native system tray doesn't support tooltips
	// This is a placeholder for future implementation
}

// UpdateTrayTitle updates the tray title
func (tm *TrayManager) UpdateTrayTitle(title string) {
	// Fyne's native system tray doesn't support dynamic title updates
	// This is a placeholder for future implementation
}
