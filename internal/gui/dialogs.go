package gui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/setavenger/blindbit-desktop/internal/manager"
	"github.com/setavenger/blindbit-lib/types"
)

// showCreateWalletDialog shows the create/import wallet dialog
func (g *MainGUI) showCreateWalletDialog() {
	// Create a dialog with options
	content := container.NewVBox(
		widget.NewLabel("Choose how to set up your wallet:"),
		widget.NewSeparator(),
	)

	// Option 1: Generate new seed
	generateButton := widget.NewButtonWithIcon("Generate New Seed", theme.ContentAddIcon(), func() {
		g.showGenerateSeedDialog()
	})

	// Option 2: Enter existing seed
	enterButton := widget.NewButtonWithIcon("Import Existing Seed", theme.DocumentIcon(), func() {
		g.showEnterSeedDialog()
	})

	content.Add(generateButton)
	content.Add(enterButton)

	dialog := dialog.NewCustom("Create/Import Wallet", "Cancel", content, g.window)
	dialog.Resize(fyne.NewSize(500, 300))
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

	// Use multi-line entry that's read-only but allows copying
	seedEntry := widget.NewMultiLineEntry()
	seedEntry.SetText(seed)
	seedEntry.Disable() // Make it read-only but still copyable
	seedEntry.Wrapping = fyne.TextWrapWord

	warningLabel := widget.NewLabel("⚠️  IMPORTANT: Write down this seed phrase and keep it safe!")
	warningLabel.TextStyle = fyne.TextStyle{Bold: true}

	copyHintLabel := widget.NewLabel("💡 Tip: You can copy the seed phrase above by selecting and copying the text")
	copyHintLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Network selection
	networkLabel := widget.NewLabel("Select Network:")
	networkSelect := widget.NewSelect([]string{manager.NetworkTestnet, manager.NetworkMainnet, manager.NetworkSignet, manager.NetworkRegtest}, nil)
	networkSelect.SetSelected(manager.DefaultNetwork) // Default to mainnet

	content := container.NewVBox(
		seedLabel,
		seedEntry,
		copyHintLabel,
		widget.NewSeparator(),
		networkLabel,
		networkSelect,
		widget.NewSeparator(),
		warningLabel,
		widget.NewLabel("You will need this to recover your wallet if you lose access to this device."),
	)

	// Create dialog with confirm button
	dialog := dialog.NewCustomConfirm("New Seed Generated", "Create Wallet", "Cancel", content, func(create bool) {
		if create {
			// Set network before creating wallet
			if err := g.walletManager.SetNetwork(types.Network(networkSelect.Selected)); err != nil {
				dialog.ShowError(fmt.Errorf("failed to set network: %v", err), g.window)
				return
			}

			if err := g.walletManager.CreateWallet(seed); err != nil {
				dialog.ShowError(fmt.Errorf("failed to create wallet: %v", err), g.window)
				return
			}

			g.cachedAddress = "" // Clear cached address for new wallet
			g.content = g.createMainScreen()
			g.window.SetContent(g.content)

			// Start scanning for the newly created wallet
			if err := g.walletManager.StartScanning(); err != nil {
				dialog.ShowError(fmt.Errorf("failed to start scanner: %v", err), g.window)
			}
		}
	}, g.window)
	dialog.Resize(fyne.NewSize(600, 500))
	dialog.Show()
}

// showEnterSeedDialog shows the dialog for entering an existing seed
func (g *MainGUI) showEnterSeedDialog() {
	seedEntry := widget.NewMultiLineEntry()
	seedEntry.SetPlaceHolder("Enter your 24-word seed phrase")
	seedEntry.Wrapping = fyne.TextWrapWord

	// Network selection
	networkLabel := widget.NewLabel("Select Network:")
	networkSelect := widget.NewSelect([]string{manager.NetworkTestnet, manager.NetworkMainnet, manager.NetworkSignet, manager.NetworkRegtest}, nil)
	networkSelect.SetSelected(manager.DefaultNetwork) // Default to mainnet

	// Birth height input
	birthHeightLabel := widget.NewLabel("Birth Height (optional):")
	birthHeightEntry := widget.NewEntry()
	birthHeightEntry.SetPlaceHolder("Leave empty to use default")

	// Function to update birth height based on network
	updateBirthHeight := func(network string) {
		switch network {
		case manager.NetworkSignet:
			birthHeightEntry.SetText("240000")
		case manager.NetworkMainnet:
			birthHeightEntry.SetText("900000")
		default:
			birthHeightEntry.SetText("0")
		}
	}

	// Set initial birth height based on default network
	updateBirthHeight(manager.DefaultNetwork)

	// Update birth height when network changes
	networkSelect.OnChanged = func(selected string) {
		updateBirthHeight(selected)
	}

	content := container.NewVBox(
		widget.NewLabel("Import Existing Wallet"),
		widget.NewSeparator(),
		widget.NewLabel("Seed Phrase:"),
		seedEntry,
		widget.NewSeparator(),
		networkLabel,
		networkSelect,
		widget.NewSeparator(),
		birthHeightLabel,
		birthHeightEntry,
		widget.NewLabel("Birth height is the block height when you first used this wallet. Leave empty to scan from a default height."),
	)

	// Create dialog with import button
	dialog := dialog.NewCustomConfirm("Import Wallet", "Import", "Cancel", content, func(shouldImport bool) {
		if !shouldImport {
			return
		}

		// Set network before creating wallet
		if err := g.walletManager.SetNetwork(types.Network(networkSelect.Selected)); err != nil {
			dialog.ShowError(fmt.Errorf("failed to set network: %v", err), g.window)
			return
		}

		// Set birth height if provided
		if birthHeightEntry.Text != "" {
			if height, err := strconv.ParseUint(birthHeightEntry.Text, 10, 64); err == nil {
				if err := g.walletManager.SetBirthHeight(height); err != nil {
					dialog.ShowError(fmt.Errorf("failed to set birth height: %v", err), g.window)
					return
				}
			} else {
				dialog.ShowError(fmt.Errorf("invalid birth height: %v", err), g.window)
				return
			}
		}

		if err := g.walletManager.CreateWallet(seedEntry.Text); err != nil {
			dialog.ShowError(fmt.Errorf("failed to import wallet: %v", err), g.window)
			return
		}

		g.cachedAddress = "" // Clear cached address for imported wallet
		g.content = g.createMainScreen()
		g.window.SetContent(g.content)

		// Start scanning for the newly imported wallet
		if err := g.walletManager.StartScanning(); err != nil {
			dialog.ShowError(fmt.Errorf("failed to start scanner: %v", err), g.window)
		}
	}, g.window)
	dialog.Resize(fyne.NewSize(600, 600))
	dialog.Show()
}
