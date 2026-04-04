package gui

import (
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (g *MainGUI) createOverviewTab() fyne.CanvasObject {
	// --- Balance section ---
	balanceTitleLabel := widget.NewLabel("Balance")
	balanceTitleLabel.TextStyle.Bold = true

	balanceLabel := widget.NewLabel("0 sats")
	balanceLabel.TextStyle.Bold = true

	// Update balance from unspent UTXOs
	updateBalance := func() {
		unspentUTXOs := g.manager.GetUnspentUTXOsSorted()
		var total uint64
		for _, utxo := range unspentUTXOs {
			total += utxo.Amount
		}
		balanceLabel.SetText(FormatSatoshiUint64(total))
	}
	updateBalance()

	balanceSection := container.NewVBox(
		balanceTitleLabel,
		balanceLabel,
	)

	// --- Scanning status section ---
	scanTitleLabel := widget.NewLabel("Sync Status")
	scanTitleLabel.TextStyle.Bold = true

	currentScanLabel := widget.NewLabel(
		"Scanned Height: " + FormatHeightUint64(g.manager.Wallet.LastScanHeight),
	)
	chainTipLabel := widget.NewLabel("Chain Tip: N/A")

	if currentHeight, err := g.manager.GetCurrentHeight(); err == nil {
		chainTipLabel.SetText("Chain Tip: " + FormatHeight(currentHeight))
	}

	scanSection := container.NewVBox(
		scanTitleLabel,
		currentScanLabel,
		chainTipLabel,
	)

	// --- Recent transactions section ---
	recentTxTitleLabel := widget.NewLabel("Recent Transactions")
	recentTxTitleLabel.TextStyle.Bold = true

	createHeaderLabel := func(text string) *widget.Label {
		label := widget.NewLabel(text)
		label.TextStyle.Bold = true
		label.Alignment = fyne.TextAlignLeading
		return label
	}

	headers := container.NewGridWithColumns(4,
		createHeaderLabel("TXID"),
		createHeaderLabel("Block Height"),
		createHeaderLabel("Net Amount"),
		createHeaderLabel("Status"),
	)

	// Pre-compute sorted indices (newest first, by confirm height) once.
	// This is updated on each refresh tick so updateItem can just index into it.
	buildSortedIndices := func() []int {
		history := g.manager.TransactionHistory
		indices := make([]int, len(history))
		for i := range indices {
			indices[i] = i
		}
		sort.Slice(indices, func(a, b int) bool {
			return history[indices[a]].ConfirmHeight > history[indices[b]].ConfirmHeight
		})
		return indices
	}
	sortedIndices := buildSortedIndices()

	// Show the most recent transactions (up to 10)
	recentTxList := widget.NewList(
		func() int {
			n := len(sortedIndices)
			if n > 10 {
				return 10
			}
			return n
		},
		func() fyne.CanvasObject {
			return container.NewGridWithColumns(4,
				widget.NewLabel(""), // TXID
				widget.NewLabel(""), // Block Height
				widget.NewLabel(""), // Net Amount
				widget.NewLabel(""), // Status
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			history := g.manager.TransactionHistory

			if id < len(sortedIndices) {
				tx := history[sortedIndices[id]]
				c := obj.(*fyne.Container)

				txidLabel := c.Objects[0].(*widget.Label)
				heightLabel := c.Objects[1].(*widget.Label)
				amountLabel := c.Objects[2].(*widget.Label)
				statusLabel := c.Objects[3].(*widget.Label)

				txidHex := hex.EncodeToString(tx.TxID[:])
				txidLabel.SetText(fmt.Sprintf("%.8s...", txidHex))
				heightLabel.SetText(FormatNumber(int64(tx.ConfirmHeight)))

				netAmount := tx.NetAmount()
				var amountText string
				if netAmount > 0 {
					amountText = "+" + FormatSatoshiUint64(uint64(netAmount))
				} else if netAmount < 0 {
					amountText = FormatSatoshi(int64(netAmount))
				} else {
					amountText = FormatSatoshiUint64(0)
				}
				amountLabel.SetText(amountText)

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
	recentTxScroll := container.NewScroll(recentTxList)
	recentTxScroll.SetMinSize(fyne.NewSize(440, 200))

	recentTxSection := container.NewVBox(
		recentTxTitleLabel,
		widget.NewSeparator(),
		headers,
		widget.NewSeparator(),
		recentTxScroll,
	)

	// --- Periodic updates ---
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			updateBalance()
			sortedIndices = buildSortedIndices()
			recentTxList.Refresh()
			currentScanLabel.SetText(
				"Scanned Height: " + FormatHeightUint64(g.manager.Wallet.LastScanHeight),
			)
			if currentHeight, err := g.manager.GetCurrentHeight(); err == nil {
				chainTipLabel.SetText("Chain Tip: " + FormatHeight(currentHeight))
			}
		}
	}()

	// --- Main layout ---
	content := container.NewVBox(
		balanceSection,
		widget.NewSeparator(),
		scanSection,
		widget.NewSeparator(),
		recentTxSection,
	)

	return content
}
