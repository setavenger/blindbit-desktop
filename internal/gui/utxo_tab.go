package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createUTXOOverviewTab creates the UTXO overview tab
func (g *MainGUI) createUTXOOverviewTab() *fyne.Container {
	// UTXO Statistics
	totalLabel := widget.NewLabel("Total UTXOs: 0")
	unspentLabel := widget.NewLabel("Unspent UTXOs: 0")
	spentLabel := widget.NewLabel("Spent UTXOs: 0")
	balanceLabel := widget.NewLabel("Total Balance: 0 sats")
	syncStatusLabel := widget.NewLabel("Sync Status: Loading...")

	// Update statistics function
	updateStats := func() {
		total, unspent, spent := g.walletManager.GetUTXOStats()
		balance := g.walletManager.GetBalance()

		totalLabel.SetText(fmt.Sprintf("Total UTXOs: %d", total))
		unspentLabel.SetText(fmt.Sprintf("Unspent UTXOs: %d", unspent))
		spentLabel.SetText(fmt.Sprintf("Spent UTXOs: %d", spent))
		balanceLabel.SetText(fmt.Sprintf("Total Balance: %d sats", balance))

		// Update sync status
		_, chainTip, syncPercentage := g.walletManager.GetSyncStatus()
		if chainTip > 0 {
			syncStatusLabel.SetText(fmt.Sprintf("Sync Status: %.1f%% (Chain Tip: %d)", syncPercentage, chainTip))
		} else {
			syncStatusLabel.SetText("Sync Status: Loading...")
		}

		totalLabel.Refresh()
		unspentLabel.Refresh()
		spentLabel.Refresh()
		balanceLabel.Refresh()
		syncStatusLabel.Refresh()
	}

	// Use the shared UTXO table component
	utxoList := g.createUTXOTable()

	// Control buttons
	refreshButton := widget.NewButtonWithIcon("Refresh UTXOs", theme.ViewRefreshIcon(), func() {
		g.refreshUTXOs()
		updateStats()
	})

	clearButton := widget.NewButtonWithIcon("Clear UTXOs", theme.DeleteIcon(), func() {
		dialog.ShowConfirm("Clear UTXOs", "Are you sure you want to clear all UTXOs? This action cannot be undone.", func(clear bool) {
			if clear {
				g.walletManager.ClearUTXOs()
				g.refreshUTXOs()
				updateStats()
				dialog.ShowInformation("UTXOs Cleared", "All UTXOs have been cleared.", g.window)
			}
		}, g.window)
	})

	scanButton := widget.NewButtonWithIcon("Start Scanning", theme.MediaPlayIcon(), func() {
		g.startScanning()
	})

	stopButton := widget.NewButtonWithIcon("Stop Scanning", theme.MediaStopIcon(), func() {
		g.stopScanning()
	})

	// Control sections
	statsSection := container.NewVBox(
		widget.NewLabel("UTXO Statistics"),
		widget.NewSeparator(),
		totalLabel,
		unspentLabel,
		spentLabel,
		balanceLabel,
		syncStatusLabel,
		widget.NewSeparator(),
	)

	controlsSection := container.NewVBox(
		widget.NewLabel("Controls"),
		widget.NewSeparator(),
		container.NewHBox(refreshButton, clearButton),
		container.NewHBox(scanButton, stopButton),
		widget.NewSeparator(),
	)

	fmt.Println("Utxolist MinSize:", utxoList.MinSize())

	// UTXO list section with proper layout - no need for separate headers now
	listSection := container.NewStack(
		widget.NewLabel("UTXO Details"),
		widget.NewSeparator(),
		utxoList, // This makes the table take up all available space
	)

	// Initial stats update
	updateStats()

	// Left panel with stats and controls
	leftPanel := container.NewVBox(
		statsSection,
		controlsSection,
	)

	// Main layout
	mainContainer := container.NewHSplit(
		leftPanel,
		listSection,
	)
	mainContainer.SetOffset(0.3) // 30% for left panel, 70% for UTXO list

	return container.NewPadded(mainContainer)
}
