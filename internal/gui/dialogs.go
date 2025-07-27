package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// showCreateWalletDialog shows the create wallet dialog
func (g *MainGUI) showCreateWalletDialog() {
	// Create a dialog with options
	content := container.NewVBox(
		widget.NewLabel("Choose how to create your wallet:"),
		widget.NewSeparator(),
	)

	// Option 1: Generate new seed
	generateButton := widget.NewButtonWithIcon("Generate New Seed", theme.ContentAddIcon(), func() {
		g.showGenerateSeedDialog()
	})

	// Option 2: Enter existing seed
	enterButton := widget.NewButtonWithIcon("Enter Existing Seed", theme.DocumentIcon(), func() {
		g.showEnterSeedDialog()
	})

	content.Add(generateButton)
	content.Add(enterButton)

	dialog := dialog.NewCustom("Create Wallet", "Cancel", content, g.window)
	dialog.Resize(fyne.NewSize(400, 200))
	dialog.Show()
}

// showGenerateSeedDialog shows the dialog for generating a new seed
func (g *MainGUI) showGenerateSeedDialog() {
	// Generate new seed
	seed, err := g.walletManager.GenerateNewSeed()
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to generate seed: %v", err), g.window)
		return
	}

	// Create content with the generated seed
	seedLabel := widget.NewLabel("Your new seed phrase:")
	seedLabel.TextStyle = fyne.TextStyle{Bold: true}

	seedEntry := widget.NewEntry()
	seedEntry.SetText(seed)
	seedEntry.Disable() // Make it read-only

	warningLabel := widget.NewLabel("⚠️  IMPORTANT: Write down this seed phrase and keep it safe!")
	warningLabel.TextStyle = fyne.TextStyle{Bold: true}

	content := container.NewVBox(
		seedLabel,
		seedEntry,
		widget.NewSeparator(),
		warningLabel,
		widget.NewLabel("You will need this to recover your wallet if you lose access to this device."),
	)

	// Create dialog with confirm button
	dialog.ShowCustomConfirm("New Seed Generated", "Create Wallet", "Cancel", content, func(create bool) {
		if create {
			if err := g.walletManager.CreateWallet(seed); err != nil {
				dialog.ShowError(fmt.Errorf("failed to create wallet: %v", err), g.window)
				return
			}

			g.cachedAddress = "" // Clear cached address for new wallet
			g.content = g.createMainScreen()
			g.window.SetContent(g.content)
		}
	}, g.window)
}

// showEnterSeedDialog shows the dialog for entering an existing seed
func (g *MainGUI) showEnterSeedDialog() {
	seedEntry := widget.NewEntry()
	seedEntry.SetPlaceHolder("Enter your 24-word seed phrase")

	dialog.ShowForm("Import Wallet", "Import", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Seed Phrase", seedEntry),
		},
		func(shouldImport bool) {
			if !shouldImport {
				return
			}

			if err := g.walletManager.CreateWallet(seedEntry.Text); err != nil {
				dialog.ShowError(fmt.Errorf("failed to import wallet: %v", err), g.window)
				return
			}

			g.cachedAddress = "" // Clear cached address for imported wallet
			g.content = g.createMainScreen()
			g.window.SetContent(g.content)
		},
		g.window,
	)
}

// showImportWalletDialog shows the import wallet dialog
func (g *MainGUI) showImportWalletDialog() {
	// For now, redirect to the enter seed dialog
	g.showEnterSeedDialog()
}
