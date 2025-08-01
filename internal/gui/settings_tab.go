package gui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/setavenger/blindbit-desktop/internal/wallet"
)

// createSettingsTab creates the settings tab
func (g *MainGUI) createSettingsTab() *fyne.Container {
	// Network selection
	networkSelect := widget.NewSelect([]string{"testnet", "mainnet", "signet", "regtest"}, func(value string) {
		if err := g.walletManager.SetNetwork(wallet.Network(value)); err != nil {
			dialog.ShowError(fmt.Errorf("failed to change network: %v", err), g.window)
			return
		}
		// Clear cached address when network changes
		g.cachedAddress = ""
		// Refresh wallet info after network change
		g.updateWalletInfo()
		// Update Oracle URL display as it might change with network
		g.updateOracleURLDisplay()
	})

	// Set the selected network after widget creation to avoid UI freezing
	selectedNetwork := string(g.walletManager.GetNetwork())
	networkSelect.SetSelected(selectedNetwork)

	// Oracle settings
	oracleURL := widget.NewEntry()
	oracleURL.SetText(g.walletManager.GetOracleURL())
	oracleURL.SetPlaceHolder("Oracle server URL")

	electrumURL := widget.NewEntry()
	electrumURL.SetText(g.walletManager.GetElectrumURL())
	electrumURL.SetPlaceHolder("Electrum server URL")

	useTor := widget.NewCheck("Use Tor", func(checked bool) {
		if err := g.walletManager.SetUseTor(checked); err != nil {
			dialog.ShowError(fmt.Errorf("failed to set Tor setting: %v", err), g.window)
		}
	})
	useTor.SetChecked(g.walletManager.GetUseTor())

	// Wallet settings
	dustLimit := widget.NewEntry()
	dustLimit.SetText(fmt.Sprintf("%d", g.walletManager.GetDustLimit()))
	dustLimit.SetPlaceHolder("Dust limit in sats")

	labelCount := widget.NewEntry()
	labelCount.SetText(fmt.Sprintf("%d", g.walletManager.GetLabelCount()))
	labelCount.SetPlaceHolder("Number of labels")

	birthHeight := widget.NewEntry()
	birthHeight.SetText(fmt.Sprintf("%d", g.walletManager.GetBirthHeight()))
	birthHeight.SetPlaceHolder("Birth height")

	// Save settings button
	saveButton := widget.NewButtonWithIcon("Save Settings", theme.DocumentSaveIcon(), func() {
		// Save oracle URL
		if err := g.walletManager.SetOracleURL(oracleURL.Text); err != nil {
			dialog.ShowError(fmt.Errorf("failed to save oracle URL: %v", err), g.window)
			return
		}

		// Save electrum URL
		if err := g.walletManager.SetElectrumURL(electrumURL.Text); err != nil {
			dialog.ShowError(fmt.Errorf("failed to save electrum URL: %v", err), g.window)
			return
		}

		// Save dust limit
		if dustLimitInt, err := strconv.ParseUint(dustLimit.Text, 10, 64); err == nil {
			if err := g.walletManager.SetDustLimit(dustLimitInt); err != nil {
				dialog.ShowError(fmt.Errorf("failed to save dust limit: %v", err), g.window)
				return
			}
		}

		// Save label count
		if labelCountInt, err := strconv.Atoi(labelCount.Text); err == nil {
			if err := g.walletManager.SetLabelCount(labelCountInt); err != nil {
				dialog.ShowError(fmt.Errorf("failed to save label count: %v", err), g.window)
				return
			}
		}

		// Save birth height
		if birthHeightInt, err := strconv.ParseUint(birthHeight.Text, 10, 64); err == nil {
			if err := g.walletManager.SetBirthHeight(birthHeightInt); err != nil {
				dialog.ShowError(fmt.Errorf("failed to save birth height: %v", err), g.window)
				return
			}
		}

		// Update the Oracle URL display in the main view
		g.updateOracleURLDisplay()

		dialog.ShowInformation("Settings", "Settings saved successfully", g.window)
	})

	// Create form sections
	networkSection := container.NewVBox(
		widget.NewLabel("Network Settings"),
		widget.NewForm(
			widget.NewFormItem("Network", networkSelect),
		),
	)

	oracleSection := container.NewVBox(
		widget.NewLabel("Oracle Settings"),
		widget.NewForm(
			widget.NewFormItem("Oracle URL", oracleURL),
			widget.NewFormItem("Electrum URL", electrumURL),
			widget.NewFormItem("Use Tor", useTor),
		),
	)

	walletSection := container.NewVBox(
		widget.NewLabel("Wallet Settings"),
		widget.NewForm(
			widget.NewFormItem("Dust Limit (sats)", dustLimit),
			widget.NewFormItem("Label Count", labelCount),
			widget.NewFormItem("Birth Height", birthHeight),
		),
	)

	mainContainer := container.NewVBox(
		networkSection,
		widget.NewSeparator(),
		oracleSection,
		widget.NewSeparator(),
		walletSection,
		widget.NewSeparator(),
		saveButton,
	)
	return mainContainer
}
