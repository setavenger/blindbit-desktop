package gui

import (
	"encoding/hex"
	"fmt"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/setavenger/blindbit-lib/types"
	"github.com/setavenger/blindbit-lib/wallet"
)

func (g *MainGUI) createTransactionsTab() fyne.CanvasObject {
	// Instructions
	instructionsText := widget.NewRichTextFromMarkdown(`
# Transaction History

This shows all transactions you have successfully broadcast.

Click on a transaction to view details.
`)

	// Create header labels with proper alignment and styling
	createHeaderLabel := func(text string) *widget.Label {
		label := widget.NewLabel(text)
		label.TextStyle.Bold = true
		label.Alignment = fyne.TextAlignLeading
		return label
	}

	// Create headers in a grid matching the list's 4 columns
	headers := container.NewGridWithColumns(4,
		createHeaderLabel("TXID"),
		createHeaderLabel("Block Height"),
		createHeaderLabel("Net Amount"),
		createHeaderLabel("Status"),
	)

	// Transaction list with proper columns
	txList := widget.NewList(
		func() int {
			return len(g.manager.TransactionHistory)
		},
		func() fyne.CanvasObject {
			// Create a container with 4 labels for each row
			return container.NewGridWithColumns(4,
				widget.NewLabel(""), // TXID
				widget.NewLabel(""), // Block Height
				widget.NewLabel(""), // Net Amount
				widget.NewLabel(""), // Status
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			history := g.manager.TransactionHistory
			if id < len(history) {
				tx := history[id]
				container := obj.(*fyne.Container)

				// Get the labels from the container
				txidLabel := container.Objects[0].(*widget.Label)
				heightLabel := container.Objects[1].(*widget.Label)
				amountLabel := container.Objects[2].(*widget.Label)
				statusLabel := container.Objects[3].(*widget.Label)

				// Set the data
				txidHex := hex.EncodeToString(tx.TxID[:])
				txidLabel.SetText(fmt.Sprintf("%.8s...", txidHex))
				heightLabel.SetText(FormatNumber(int64(tx.ConfirmHeight)))

				// Format amount with sign
				var amountText string
				netAmount := tx.NetAmount()
				if netAmount > 0 {
					amountText = "+" + FormatSatoshiUint64(uint64(netAmount))
				} else if netAmount < 0 {
					amountText = FormatSatoshi(int64(netAmount))
				} else {
					amountText = FormatSatoshiUint64(0)
				}
				amountLabel.SetText(amountText)

				// Determine status
				var status string
				if tx.ConfirmHeight > 0 {
					status = "Confirmed"
				} else {
					status = "Pending"
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

	// Create a scrollable container for the list (like UTXOs)
	scrollContainer := container.NewScroll(txList)
	scrollContainer.SetMinSize(fyne.NewSize(440, 300)) // Set minimum size

	// Main content using Border layout to fill available space
	// Put instructions and headers at top, list in center to make list fill remaining vertical space
	content := container.NewBorder(
		container.NewVBox(
			instructionsText,
			widget.NewSeparator(),
			headers,
			widget.NewSeparator(),
		), // top
		nil,             // bottom
		nil,             // left
		nil,             // right
		scrollContainer, // center - wrapped in scroll to match UTXOs pattern
	)

	return content
}

func (g *MainGUI) showTransactionHistoryDetails(tx *wallet.TxItem) {
	txidHex := hex.EncodeToString(tx.TxID[:])

	status := "Pending"
	if tx.ConfirmHeight > 0 {
		status = "Confirmed"
	}

	// TXID: Full 64 hex chars, monospace, allow word wrap (no truncation)
	txidLabel := widget.NewLabel("Transaction ID:")
	txidValue := widget.NewLabel(txidHex)
	txidValue.TextStyle.Monospace = true
	txidValue.Wrapping = fyne.TextWrapBreak // Break anywhere so long hex fits without scrolling

	// Transaction info (single-line labels for compactness)
	heightLine := widget.NewLabel("Block Height: " + FormatNumber(int64(tx.ConfirmHeight)))
	amountLine := widget.NewLabel("Total Amount: " + FormatSatoshi(int64(tx.NetAmount())))
	feeLine := widget.NewLabel("Fee: " + FormatSatoshi(int64(tx.Fees())))
	statusLine := widget.NewLabel("Status: " + status)

	// TODO: Take from TX History and
	// show both wallet internal and external UTXOs
	// // Find output UTXOs (belonging to wallet from this transaction)
	// var outputUTXOItems []fyne.CanvasObject
	// for _, utxo := range g.manager.Wallet.GetUTXOs() {
	// 	if hex.EncodeToString(utxo.Txid[:]) == txidHex {
	// 		utxoLine := widget.NewLabel(fmt.Sprintf("  vout %d — %s", utxo.Vout, FormatSatoshi(int64(utxo.Amount))))
	// 		outputUTXOItems = append(outputUTXOItems, utxoLine)
	// 	}
	// }

	// Try to find input UTXOs (sent/spent in this transaction)
	// var inputUTXOItems []fyne.CanvasObject
	// Try accessing input UTXOs via TxItem if available (check various patterns)
	// Note: Since TxItem comes from blindbit-lib, exact structure may vary
	// For now, we'll show output UTXOs which are definitive
	// Input UTXOs would require access to the original wire.MsgTx or stored input data
	// TODO: Implement input UTXO lookup when TxItem structure supports it

	// Build content sections
	contentItems := []fyne.CanvasObject{
		widget.NewLabelWithStyle("Transaction Details", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		txidLabel,
		txidValue,
		widget.NewSeparator(),
		heightLine,
		amountLine,
		feeLine,
		statusLine,
	}

	// Add output UTXOs section if any
	// if len(outputUTXOItems) > 0 {
	// 	contentItems = append(contentItems, widget.NewSeparator())
	// 	outputTitle := widget.NewLabelWithStyle("Outputs to your wallet", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	// 	contentItems = append(contentItems, outputTitle)
	// 	contentItems = append(contentItems, outputUTXOItems...)
	// }

	// Add input UTXOs section if any
	// if len(inputUTXOItems) > 0 {
	// 	contentItems = append(contentItems, widget.NewSeparator())
	// 	inputTitle := widget.NewLabelWithStyle("Inputs sent (UTXOs spent)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	// 	contentItems = append(contentItems, inputTitle)
	// 	contentItems = append(contentItems, inputUTXOItems...)
	// }

	// Buttons
	copyBtn := widget.NewButton("Copy TXID", func() {
		g.copyTxidToClipboard(txidHex)
	})

	explorerBtn := widget.NewButton("View in Explorer", func() {
		g.openInExplorer(txidHex)
	})

	innerContainer := container.NewHBox(copyBtn, explorerBtn)
	buttonLineContainer := container.NewHBox(innerContainer)
	buttonLineContainer.Layout = layout.NewCenterLayout()

	contentItems = append(contentItems,
		widget.NewSeparator(),
		buttonLineContainer,
	)

	content := container.NewVBox(contentItems...)
	// Increase dialog width so everything fits comfortably
	d := dialog.NewCustom("Transaction Details", "Close", content, g.window)
	d.Resize(fyne.NewSize(680, content.MinSize().Height))
	d.Show()
}

func (g *MainGUI) copyTxidToClipboard(text string) {
	// Copy to clipboard
	g.window.Clipboard().SetContent(text)

	// Show confirmation
	dialog.ShowInformation("Copied", "TXID copied to clipboard!", g.window)
}

func (g *MainGUI) openInExplorer(txid string) {
	// Open mempool.space explorer using Fyne's built-in OpenURL
	var urlStr string
	switch g.manager.GetNetwork() {
	case types.NetworkMainnet:
		urlStr = "https://mempool.space/tx/" + txid
	case types.NetworkTestnet:
		urlStr = "https://mempool.space/testnet/tx/" + txid
	case types.NetworkSignet:
		urlStr = "https://mempool.space/signet/tx/" + txid
	default:
		urlStr = "https://mempool.space/tx/" + txid
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to parse URL: %v", err), g.window)
		return
	}

	err = g.app.OpenURL(u)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to open URL: %v", err), g.window)
	}
}
