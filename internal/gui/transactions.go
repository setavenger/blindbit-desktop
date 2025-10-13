package gui

import (
	"encoding/hex"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/setavenger/blindbit-desktop/internal/controller"
)

func (g *MainGUI) createTransactionsTab() fyne.CanvasObject {
	// Transaction list
	txList := widget.NewList(
		func() int {
			return len(g.manager.TransactionHistory)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("TXID"),
				widget.NewLabel("Block Height"),
				widget.NewLabel("Net Amount"),
				widget.NewLabel("Status"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			history := g.manager.TransactionHistory
			if id < len(history) {
				tx := history[id]
				container := obj.(*fyne.Container)

				// Update the labels with transaction data
				txidLabel := container.Objects[0].(*widget.Label)
				heightLabel := container.Objects[1].(*widget.Label)
				amountLabel := container.Objects[2].(*widget.Label)
				statusLabel := container.Objects[3].(*widget.Label)

				// Truncate TXID for display
				txidHex := hex.EncodeToString(tx.TxID[:])
				txidLabel.SetText(fmt.Sprintf("%.8s...", txidHex))

				heightLabel.SetText(fmt.Sprintf("%d", tx.BlockHeight))

				// Format amount with sign
				var amountText string
				if tx.NetAmount > 0 {
					amountText = fmt.Sprintf("+%d sats", tx.NetAmount)
				} else {
					amountText = fmt.Sprintf("%d sats", tx.NetAmount)
				}
				amountLabel.SetText(amountText)

				// Determine status
				var status string
				if tx.BlockHeight > 0 {
					status = "Confirmed"
				} else {
					status = "Unconfirmed"
				}
				statusLabel.SetText(status)
			}
		},
	)

	// Set up click handler for transaction details
	txList.OnSelected = func(id widget.ListItemID) {
		if id < len(g.manager.TransactionHistory) {
			tx := g.manager.TransactionHistory[id]
			g.showTransactionHistoryDetails(tx)
		}
	}

	// Refresh button
	refreshBtn := widget.NewButton("Refresh", func() {
		g.refreshTransactions()
	})

	// Control panel
	controlPanel := container.NewHBox(
		refreshBtn,
	)

	// Instructions
	instructionsText := widget.NewRichTextFromMarkdown(`
# Transaction History

This shows all transactions you have successfully broadcast.

Click on a transaction to view details.
`)

	// Main content
	content := container.NewVBox(
		instructionsText,
		widget.NewSeparator(),
		controlPanel,
		widget.NewSeparator(),
		txList,
	)

	return content
}

func (g *MainGUI) showTransactionHistoryDetails(tx controller.TxHistoryItem) {
	txidHex := hex.EncodeToString(tx.TxID[:])

	detailsText := fmt.Sprintf(`
Transaction Details:

Transaction ID: %s
Block Height: %d
Net Amount: %d sats
Status: %s

Click "View in Explorer" to see this transaction on mempool.space
`,
		txidHex,
		tx.BlockHeight,
		tx.NetAmount,
		func() string {
			if tx.BlockHeight > 0 {
				return "Confirmed"
			}
			return "Unconfirmed"
		}(),
	)

	detailsLabel := widget.NewLabel(detailsText)
	detailsLabel.Wrapping = fyne.TextWrapWord

	copyBtn := widget.NewButton("Copy TXID", func() {
		g.copyTxidToClipboard(txidHex)
	})

	explorerBtn := widget.NewButton("View in Explorer", func() {
		g.openInExplorer(txidHex)
	})

	closeBtn := widget.NewButton("Close", func() {
		// Close dialog
	})

	content := container.NewVBox(
		detailsLabel,
		widget.NewSeparator(),
		container.NewHBox(copyBtn, explorerBtn, closeBtn),
	)

	dialog.ShowCustom("Transaction Details", "Close", content, g.window)
}

func (g *MainGUI) refreshTransactions() {
	// Refresh the transaction list
	// This would typically trigger a UI refresh
	fmt.Println("Refreshing transaction history")
}

func (g *MainGUI) copyTxidToClipboard(text string) {
	// Copy to clipboard
	g.window.Clipboard().SetContent(text)

	// Show confirmation
	dialog.ShowInformation("Copied", "TXID copied to clipboard!", g.window)
}

func (g *MainGUI) openInExplorer(txid string) {
	// TODO: Open mempool.space explorer
	// This would typically open the default browser
	dialog.ShowInformation("Explorer", "Opening mempool.space explorer...", g.window)
}
