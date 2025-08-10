package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/setavenger/blindbit-desktop/internal/gui"
	"github.com/setavenger/blindbit-desktop/internal/wallet"
)

func main() {
	// Create a new Fyne application
	myApp := app.New()

	myApp.SetIcon(theme.AccountIcon()) // You can replace this with a custom icon

	// Create the main window
	mainWindow := myApp.NewWindow("BlindBit Desktop")

	mainWindow.Resize(fyne.NewSize(800, 600))
	mainWindow.CenterOnScreen()

	// Initialize wallet manager
	walletManager, err := wallet.NewManager()
	if err != nil {
		log.Printf("Failed to initialize wallet manager: %v", err)
		// Show error dialog
		errorDialog := widget.NewModalPopUp(
			widget.NewLabel("Failed to initialize wallet manager: "+err.Error()),
			mainWindow.Canvas(),
		)
		errorDialog.Resize(fyne.NewSize(400, 100))
		errorDialog.Show()
	}

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
