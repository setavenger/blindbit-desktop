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
	log.Println("Starting BlindBit Desktop application...")

	// Create a new Fyne application
	myApp := app.New()
	log.Println("Fyne app created")

	myApp.SetIcon(theme.AccountIcon()) // You can replace this with a custom icon

	// Create the main window
	mainWindow := myApp.NewWindow("BlindBit Desktop")
	log.Println("Main window created")

	mainWindow.Resize(fyne.NewSize(800, 600))
	mainWindow.CenterOnScreen()

	// Initialize wallet manager
	log.Println("Initializing wallet manager...")
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
	} else {
		log.Println("Wallet manager initialized successfully")
	}

	// Create the main GUI
	log.Println("Creating main GUI...")
	mainGUI := gui.NewMainGUI(mainWindow, walletManager)
	log.Println("Main GUI created")

	// Set the main content
	log.Println("Setting main content...")
	mainWindow.SetContent(mainGUI.GetContent())
	log.Println("Main content set")

	// Set up cleanup when window is closed
	mainWindow.SetOnClosed(func() {
		log.Println("Window closing, cleaning up...")
		mainGUI.Cleanup()
	})

	// Show and run the application
	log.Println("Showing and running application...")
	mainWindow.ShowAndRun()
	log.Println("Application finished")
}
