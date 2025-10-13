package gui

import (
	"encoding/hex"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/setavenger/blindbit-lib/logging"
)

func (g *MainGUI) createUTXOsTab() fyne.CanvasObject {
	// Balance display
	balanceLabel := widget.NewLabel("Balance: 0 sats")
	balanceLabel.TextStyle.Bold = true

	// UTXO list
	utxoList := widget.NewList(
		func() int {
			return len(g.manager.GetUTXOsSorted())
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Amount"),
				widget.NewLabel("Outpoint"),
				widget.NewLabel("Height"),
				widget.NewLabel("Confirmations"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			utxos := g.manager.GetUTXOsSorted()
			if id < len(utxos) {
				utxo := utxos[id]
				container := obj.(*fyne.Container)

				// Update the labels with UTXO data
				amountLabel := container.Objects[0].(*widget.Label)
				outpointLabel := container.Objects[1].(*widget.Label)
				heightLabel := container.Objects[2].(*widget.Label)
				confirmationsLabel := container.Objects[3].(*widget.Label)

				amountLabel.SetText(fmt.Sprintf("%d sats", utxo.Amount))

				// Display txid:vout format
				txidHex := hex.EncodeToString(utxo.Txid[:])
				outpointLabel.SetText(fmt.Sprintf("%.8s...:%d", txidHex, utxo.Vout))

				heightLabel.SetText(fmt.Sprintf("%d", utxo.Height))

				// Calculate confirmations (current height - utxo height)
				// For now, show placeholder
				confirmationsLabel.SetText("N/A")
			}
		},
	)

	// Control buttons
	scanBtn := widget.NewButton("Scan", func() {
		g.startScanning()
	})

	rescanBtn := widget.NewButton("Rescan", func() {
		g.startRescanning()
	})

	refreshBtn := widget.NewButton("Refresh", func() {
		g.refreshUTXOs()
	})

	// Progress bar
	progressBar := widget.NewProgressBar()
	progressBar.Hide()

	// Update balance
	g.updateBalance(balanceLabel)

	// Control panel
	controlPanel := container.NewHBox(
		scanBtn,
		rescanBtn,
		refreshBtn,
	)

	// Main content
	content := container.NewVBox(
		balanceLabel,
		widget.NewSeparator(),
		controlPanel,
		progressBar,
		widget.NewSeparator(),
		utxoList,
	)

	return content
}

func (g *MainGUI) updateBalance(balanceLabel *widget.Label) {
	balance := g.manager.GetBalance()
	balanceLabel.SetText(fmt.Sprintf("Balance: %d sats", balance))
}

func (g *MainGUI) startScanning() {
	// TODO: Implement scanning with progress tracking
	dialog.ShowInformation("Scanning", "Scanning functionality will be implemented once scanner progress channel is available", g.window)
}

func (g *MainGUI) startRescanning() {
	// TODO: Implement rescanning
	dialog.ShowInformation("Rescanning", "Rescanning functionality will be implemented once scanner progress channel is available", g.window)
}

func (g *MainGUI) refreshUTXOs() {
	// Refresh the UTXO list
	// This would typically trigger a UI refresh
	logging.L.Info().Msg("Refreshing UTXO list")
}
