package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createWelcomeScreen creates the welcome screen for new users
func (g *MainGUI) createWelcomeScreen() *fyne.Container {
	title := widget.NewLabel("Welcome to BlindBit Desktop")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	description := widget.NewLabel("Create a new wallet or import an existing one to get started.")
	description.Alignment = fyne.TextAlignCenter

	createButton := widget.NewButtonWithIcon("Create New Wallet", theme.ContentAddIcon(), func() {
		g.showCreateWalletDialog()
	})

	importButton := widget.NewButtonWithIcon("Import Wallet", theme.FolderOpenIcon(), func() {
		g.showImportWalletDialog()
	})

	buttons := container.NewHBox(createButton, importButton)

	return container.NewVBox(
		title,
		description,
		widget.NewSeparator(),
		buttons,
	)
}
