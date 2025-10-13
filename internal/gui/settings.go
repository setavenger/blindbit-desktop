package gui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/setavenger/blindbit-desktop/internal/storage"
	"github.com/setavenger/blindbit-lib/logging"
	"github.com/setavenger/blindbit-lib/types"
)

func (g *MainGUI) createSettingsTab() fyne.CanvasObject {
	// Network selection
	networkLabel := widget.NewLabel("Network:")
	networkRadio := widget.NewRadioGroup([]string{"mainnet", "testnet", "signet"}, func(value string) {
		switch value {
		case "mainnet":
			g.manager.Wallet.Network = types.NetworkMainnet
		case "testnet":
			g.manager.Wallet.Network = types.NetworkTestnet
		case "signet":
			g.manager.Wallet.Network = types.NetworkSignet
		}
	})

	// Set current network
	switch g.manager.Wallet.Network {
	case types.NetworkMainnet:
		networkRadio.SetSelected("mainnet")
	case types.NetworkTestnet:
		networkRadio.SetSelected("testnet")
	case types.NetworkSignet:
		networkRadio.SetSelected("signet")
	default:
		networkRadio.SetSelected("testnet")
	}

	// Oracle address
	oracleLabel := widget.NewLabel("Oracle Address:")
	oracleEntry := widget.NewEntry()
	oracleEntry.SetText(g.manager.OracleAddress)

	// Birth height
	birthHeightLabel := widget.NewLabel("Birth Height:")
	birthHeightEntry := widget.NewEntry()
	birthHeightEntry.SetText(fmt.Sprintf("%d", g.manager.BirthHeight))

	// Dust limit
	dustLimitLabel := widget.NewLabel("Dust Limit (satoshis):")
	dustLimitEntry := widget.NewEntry()
	dustLimitEntry.SetText(fmt.Sprintf("%d", g.manager.DustLimit))

	// Min change amount
	minChangeLabel := widget.NewLabel("Min Change Amount (satoshis):")
	minChangeEntry := widget.NewEntry()
	minChangeEntry.SetText(fmt.Sprintf("%d", g.manager.MinChangeAmount))

	// Save button
	saveBtn := widget.NewButton("Save Settings", func() {
		g.saveSettings(oracleEntry.Text, birthHeightEntry.Text, dustLimitEntry.Text, minChangeEntry.Text)
	})

	// Reset button
	resetBtn := widget.NewButton("Reset to Defaults", func() {
		g.resetToDefaults(oracleEntry, birthHeightEntry, dustLimitEntry, minChangeEntry, networkRadio)
	})

	// Form layout
	form := container.NewVBox(
		widget.NewLabel("Wallet Settings"),
		widget.NewSeparator(),
		networkLabel,
		networkRadio,
		widget.NewSeparator(),
		oracleLabel,
		oracleEntry,
		widget.NewSeparator(),
		birthHeightLabel,
		birthHeightEntry,
		widget.NewSeparator(),
		dustLimitLabel,
		dustLimitEntry,
		widget.NewSeparator(),
		minChangeLabel,
		minChangeEntry,
		widget.NewSeparator(),
		container.NewHBox(resetBtn, saveBtn),
	)

	return form
}

func (g *MainGUI) saveSettings(oracleAddr, birthHeightStr, dustLimitStr, minChangeStr string) {
	// Parse birth height
	if birthHeightStr != "" {
		if height, err := strconv.Atoi(birthHeightStr); err == nil {
			g.manager.BirthHeight = height
		} else {
			dialog.ShowError(fmt.Errorf("invalid birth height: %v", err), g.window)
			return
		}
	}

	// Parse dust limit
	if dustLimit, err := strconv.Atoi(dustLimitStr); err == nil {
		g.manager.DustLimit = dustLimit
	} else {
		dialog.ShowError(fmt.Errorf("invalid dust limit: %v", err), g.window)
		return
	}

	// Parse min change amount
	if minChange, err := strconv.ParseUint(minChangeStr, 10, 64); err == nil {
		g.manager.MinChangeAmount = minChange
	} else {
		dialog.ShowError(fmt.Errorf("invalid min change amount: %v", err), g.window)
		return
	}

	// Set oracle address
	g.manager.OracleAddress = oracleAddr

	// Save the manager
	if err := storage.SavePlain(g.manager.DataDir, g.manager); err != nil {
		logging.L.Err(err).Msg("failed to save settings")
		dialog.ShowError(fmt.Errorf("failed to save settings: %v", err), g.window)
		return
	}

	// Show success message
	dialog.ShowInformation("Success", "Settings saved successfully!", g.window)
}

func (g *MainGUI) resetToDefaults(oracleEntry, birthHeightEntry, dustLimitEntry, minChangeEntry *widget.Entry, networkRadio *widget.RadioGroup) {
	// Reset to default values
	networkRadio.SetSelected("testnet")
	g.manager.Wallet.Network = types.NetworkTestnet

	oracleEntry.SetText("127.0.0.1:7001")
	g.manager.OracleAddress = "127.0.0.1:7001"

	birthHeightEntry.SetText("0")
	g.manager.BirthHeight = 0

	dustLimitEntry.SetText("546")
	g.manager.DustLimit = 546

	minChangeEntry.SetText("546")
	g.manager.MinChangeAmount = 546

	dialog.ShowInformation("Reset", "Settings reset to defaults", g.window)
}
