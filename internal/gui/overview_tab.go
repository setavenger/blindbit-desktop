package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createOverviewTab creates the overview tab
func (g *MainGUI) createOverviewTab() *fyne.Container {
	// Use the shared UTXO table component (unspent only for overview)
	g.utxoList = g.createUTXOTableWithFilter(func() []string {
		return []string{"unspent"} // Overview tab shows only unspent UTXOs
	})

	// Scan controls with better UX
	scanButton := widget.NewButtonWithIcon("Start Scanning", theme.MediaPlayIcon(), func() {
		g.startScanning()
	})

	stopButton := widget.NewButtonWithIcon("Stop Scanning", theme.MediaStopIcon(), func() {
		g.stopScanning()
	})

	// Rescan button with dropdown for different options
	rescanButton := widget.NewButtonWithIcon("Rescan", theme.ViewRefreshIcon(), func() {
		g.showRescanDialog()
	})

	// Group scan controls
	scanControls := container.NewHBox(
		scanButton,
		stopButton,
		widget.NewSeparator(),
		rescanButton,
	)

	// Network info
	networkInfo := widget.NewLabel(fmt.Sprintf("Network: %s", string(g.walletManager.GetNetwork())))

	// Create info section
	infoSection := container.NewVBox(
		networkInfo,
		g.oracleInfoLabel,
		widget.NewSeparator(),
	)

	// UTXO section with proper layout - no need for separate headers now
	utxoSection := container.NewStack(
		widget.NewLabel("Recent UTXOs"),
		widget.NewSeparator(),
		g.utxoList, // Use NewStack to allow table to expand beyond MinSize
	)

	return container.NewBorder(
		container.NewVBox(
			infoSection,
			scanControls,
			widget.NewSeparator(),
		),
		nil, nil, nil,
		utxoSection,
	)
}

// showRescanDialog shows a dialog with rescan options
func (g *MainGUI) showRescanDialog() {
	// Create rescan options
	options := []string{
		"Rescan from current height (preserve UTXOs)",
		"Force rescan from current height (clear UTXOs)",
		"Rescan from birth height (preserve UTXOs)",
		"Force rescan from birth height (clear UTXOs)",
		"Rescan from specific height...",
	}

	// Create a select widget
	selectWidget := widget.NewSelect(options, nil)
	selectWidget.SetSelected(options[0]) // Default selection

	// Create the dialog content
	content := container.NewVBox(
		widget.NewLabel("Choose rescan option:"),
		selectWidget,
		widget.NewLabel("Note: Force rescan will clear all existing UTXOs"),
	)

	// Create the dialog
	dialog.ShowCustomConfirm("Rescan Options", "Rescan", "Cancel", content, func(confirmed bool) {
		if !confirmed {
			return
		}

		selected := selectWidget.Selected
		if selected == "" {
			return
		}

		// Handle different rescan options
		switch selected {
		case "Rescan from current height (preserve UTXOs)":
			g.rescanFromCurrentHeight(false)
		case "Force rescan from current height (clear UTXOs)":
			g.rescanFromCurrentHeight(true)
		case "Rescan from birth height (preserve UTXOs)":
			g.rescanFromBirthHeight(false)
		case "Force rescan from birth height (clear UTXOs)":
			g.rescanFromBirthHeight(true)
		case "Rescan from specific height...":
			g.showSpecificHeightDialog()
		}
	}, g.window)
}

// showSpecificHeightDialog shows a dialog to input a specific height
func (g *MainGUI) showSpecificHeightDialog() {
	heightEntry := widget.NewEntry()
	heightEntry.SetPlaceHolder("Enter block height (e.g., 840000)")

	// Create radio buttons for preserve/clear options
	preserveRadio := widget.NewRadioGroup([]string{"Preserve existing UTXOs", "Clear all UTXOs"}, nil)
	preserveRadio.SetSelected("Preserve existing UTXOs")

	content := container.NewVBox(
		widget.NewLabel("Enter block height to rescan from:"),
		heightEntry,
		widget.NewSeparator(),
		widget.NewLabel("UTXO handling:"),
		preserveRadio,
	)

	dialog.ShowCustomConfirm("Rescan from Height", "Rescan", "Cancel", content, func(confirmed bool) {
		if !confirmed {
			return
		}

		height := heightEntry.Text
		if height == "" {
			dialog.ShowError(fmt.Errorf("please enter a block height"), g.window)
			return
		}

		// Parse height
		var heightUint uint64
		if _, err := fmt.Sscanf(height, "%d", &heightUint); err != nil {
			dialog.ShowError(fmt.Errorf("invalid height: %s", height), g.window)
			return
		}

		// Perform rescan
		clearUTXOs := preserveRadio.Selected == "Clear all UTXOs"
		g.rescanFromHeight(heightUint, clearUTXOs)
	}, g.window)
}

// rescanFromCurrentHeight rescans from the current scan height
func (g *MainGUI) rescanFromCurrentHeight(clearUTXOs bool) {
	currentHeight := g.walletManager.GetScanHeight()
	if clearUTXOs {
		if err := g.walletManager.ForceRescanFromHeight(uint64(currentHeight)); err != nil {
			dialog.ShowError(fmt.Errorf("failed to force rescan: %v", err), g.window)
			return
		}
		dialog.ShowInformation("Rescan", "Force rescan started from current height", g.window)
	} else {
		if err := g.walletManager.RescanFromHeight(uint64(currentHeight)); err != nil {
			dialog.ShowError(fmt.Errorf("failed to rescan: %v", err), g.window)
			return
		}
		dialog.ShowInformation("Rescan", "Rescan started from current height", g.window)
	}

	// Update scan height immediately to show the reset (fast, no network calls)
	g.updateScanHeightOnly()

	// Automatically start scanning after rescan
	if err := g.walletManager.StartScanning(); err != nil {
		dialog.ShowError(fmt.Errorf("failed to start scanning after rescan: %v", err), g.window)
		return
	}
	// Note: No need for second updateWalletInfo() call - the periodic updater will handle it
}

// rescanFromBirthHeight rescans from the birth height
func (g *MainGUI) rescanFromBirthHeight(clearUTXOs bool) {
	birthHeight := g.walletManager.GetBirthHeight()
	if clearUTXOs {
		if err := g.walletManager.ForceRescanFromHeight(birthHeight); err != nil {
			dialog.ShowError(fmt.Errorf("failed to force rescan: %v", err), g.window)
			return
		}
		dialog.ShowInformation("Rescan", "Force rescan started from birth height", g.window)
	} else {
		if err := g.walletManager.RescanFromHeight(birthHeight); err != nil {
			dialog.ShowError(fmt.Errorf("failed to rescan: %v", err), g.window)
			return
		}
		dialog.ShowInformation("Rescan", "Rescan started from birth height", g.window)
	}

	// Update scan height immediately to show the reset (fast, no network calls)
	g.updateScanHeightOnly()

	// Automatically start scanning after rescan
	if err := g.walletManager.StartScanning(); err != nil {
		dialog.ShowError(fmt.Errorf("failed to start scanning after rescan: %v", err), g.window)
		return
	}
	// Note: No need for second updateWalletInfo() call - the periodic updater will handle it
}

// rescanFromHeight rescans from a specific height
func (g *MainGUI) rescanFromHeight(height uint64, clearUTXOs bool) {
	if clearUTXOs {
		if err := g.walletManager.ForceRescanFromHeight(height); err != nil {
			dialog.ShowError(fmt.Errorf("failed to force rescan: %v", err), g.window)
			return
		}
		dialog.ShowInformation("Rescan", fmt.Sprintf("Force rescan started from height %d", height), g.window)
	} else {
		if err := g.walletManager.RescanFromHeight(height); err != nil {
			dialog.ShowError(fmt.Errorf("failed to rescan: %v", err), g.window)
			return
		}
		dialog.ShowInformation("Rescan", fmt.Sprintf("Rescan started from height %d", height), g.window)
	}

	// Update scan height immediately to show the reset (fast, no network calls)
	g.updateScanHeightOnly()

	// Automatically start scanning after rescan
	if err := g.walletManager.StartScanning(); err != nil {
		dialog.ShowError(fmt.Errorf("failed to start scanning after rescan: %v", err), g.window)
		return
	}
	// Note: No need for second updateWalletInfo() call - the periodic updater will handle it
}

// startScanning starts the scanning process
func (g *MainGUI) startScanning() {
	if err := g.walletManager.StartScanning(); err != nil {
		dialog.ShowError(fmt.Errorf("failed to start scanning: %v", err), g.window)
		return
	}

	// Update UI to show scanning status
	g.updateWalletInfo()
	dialog.ShowInformation("Scanning", "Scanning started successfully", g.window)
}

// stopScanning stops the scanning process
func (g *MainGUI) stopScanning() {
	g.walletManager.StopScanning()

	// Update UI to show stopped status
	g.updateWalletInfo()
	dialog.ShowInformation("Scanning", "Scanning stopped", g.window)
}
