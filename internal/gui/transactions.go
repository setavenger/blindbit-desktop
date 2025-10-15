package gui

import (
	"encoding/hex"
	"fmt"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/setavenger/blindbit-desktop/internal/controller"
)

func (g *MainGUI) createTransactionsTab() fyne.CanvasObject {
	// Create table with headers
	table := widget.NewTable(
		func() (int, int) {
			return len(g.manager.TransactionHistory) + 1, 4 // +1 for header row
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)

			if id.Row == 0 {
				// Header row
				switch id.Col {
				case 0:
					label.SetText("TXID")
				case 1:
					label.SetText("Block Height")
				case 2:
					label.SetText("Net Amount")
				case 3:
					label.SetText("Status")
				}
				label.TextStyle.Bold = true
			} else {
				// Data rows
				history := g.manager.TransactionHistory
				if id.Row-1 < len(history) {
					tx := history[id.Row-1]

					switch id.Col {
					case 0:
						// Truncate TXID for display
						txidHex := hex.EncodeToString(tx.TxID[:])
						label.SetText(fmt.Sprintf("%.8s...", txidHex))
					case 1:
						label.SetText(fmt.Sprintf("%d", tx.BlockHeight))
					case 2:
						// Format amount with sign
						var amountText string
						if tx.NetAmount > 0 {
							amountText = fmt.Sprintf("+%d sats", tx.NetAmount)
						} else {
							amountText = fmt.Sprintf("%d sats", tx.NetAmount)
						}
						label.SetText(amountText)
					case 3:
						// Determine status
						var status string
						if tx.BlockHeight > 0 {
							status = "Confirmed"
						} else {
							status = "Unconfirmed"
						}
						label.SetText(status)
					}
				}
			}
		},
	)

	// Set up click handler for transaction details
	table.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 && id.Row-1 < len(g.manager.TransactionHistory) {
			tx := g.manager.TransactionHistory[id.Row-1]
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
		table,
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
Fee: %d sats
Fee Rate: %d sat/vB
Status: %s

Click "View in Explorer" to see this transaction on mempool.space
`,
		txidHex,
		tx.BlockHeight,
		tx.NetAmount,
		tx.Fee,
		tx.FeeRate,
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
	// Sync receive transactions from UTXOs
	g.manager.SyncReceiveTransactions()

	// Sort by block height descending (newest first)
	sort.Slice(g.manager.TransactionHistory, func(i, j int) bool {
		return g.manager.TransactionHistory[i].BlockHeight > g.manager.TransactionHistory[j].BlockHeight
	})

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
