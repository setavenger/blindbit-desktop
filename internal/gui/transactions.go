package gui

import (
	"encoding/hex"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/setavenger/blindbit-lib/wallet"
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
						label.SetText(fmt.Sprintf("%d", tx.ConfirmHeight))
					case 2:
						// Format amount with sign
						var amountText string
						netAmount := tx.NetAmount()
						if netAmount > 0 {
							amountText = fmt.Sprintf("+%d sats", netAmount)
						} else {
							amountText = fmt.Sprintf("%d sats", netAmount)
						}
						label.SetText(amountText)
					case 3:
						// Determine status
						var status string
						if tx.ConfirmHeight > 0 {
							status = "Confirmed"
						} else {
							status = "Pending"
						}
						label.SetText(status)
					}
				}
			}
		},
	)

	// Configure table column widths to prevent overlap
	table.SetColumnWidth(0, 120) // TXID column
	table.SetColumnWidth(1, 100) // Block Height column
	table.SetColumnWidth(2, 120) // Net Amount column
	table.SetColumnWidth(3, 100) // Status column

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

	// Create a scrollable container for the table to ensure it fills available space
	tableContainer := container.NewScroll(table)
	tableContainer.SetMinSize(fyne.NewSize(440, 300)) // Set minimum size

	// Main content using Border layout to fill available space
	content := container.NewBorder(
		nil, // top
		nil, // bottom
		nil, // left
		nil, // right
		container.NewVBox(
			instructionsText,
			widget.NewSeparator(),
			controlPanel,
			widget.NewSeparator(),
			tableContainer,
		),
	)

	return content
}

func (g *MainGUI) showTransactionHistoryDetails(tx *wallet.TxItem) {
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
		tx.ConfirmHeight,
		tx.NetAmount(),
		tx.Fees(),
		0, // todo: add fee rate
		func() string {
			if tx.ConfirmHeight > 0 {
				return "Confirmed"
			}
			return "Pending"
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
	// todo: remove this as it's no longer needed
	// or make this load data from database and update the view
	// Sync receive transactions from UTXOs

	// Sort the transaction history using the built-in Sort method
	g.manager.TransactionHistory.Sort()

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
