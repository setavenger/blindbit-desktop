package gui

import (
	"sort"
	"sync"
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
			ia, ib := indices[a], indices[b]
			ha, hb := history[ia].ConfirmHeight, history[ib].ConfirmHeight
			pa, pb := ha == 0, hb == 0 // matches FormatTxRow: Pending iff ConfirmHeight == 0
			if pa != pb {
				return pa // pending first
			}
			if ha != hb {
				return ha > hb // newest confirmed first
			}
			return ia > ib // same height: preserve newer-in-history first (higher index)
		})
		return indices
	}
	var mu sync.RWMutex
	sortedIndices := buildSortedIndices()

	// Show the most recent transactions (up to 10)
	recentTxList := widget.NewList(
		func() int {
			mu.RLock()
			n := len(sortedIndices)
			mu.RUnlock()
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

			mu.RLock()
			idxLen := len(sortedIndices)
			var txIdx int
			if id < idxLen {
				txIdx = sortedIndices[id]
			}
			mu.RUnlock()

			if id < idxLen {
				// Guard against the history slice being replaced with a shorter one.
				if txIdx >= len(history) {
					return
				}
				tx := history[txIdx]
				c := obj.(*fyne.Container)

				txidLabel := c.Objects[0].(*widget.Label)
				heightLabel := c.Objects[1].(*widget.Label)
				amountLabel := c.Objects[2].(*widget.Label)
				statusLabel := c.Objects[3].(*widget.Label)

				row := FormatTxRow(tx)
				txidLabel.SetText(row.TXID)
				heightLabel.SetText(row.Height)
				amountLabel.SetText(row.NetAmount)
				statusLabel.SetText(row.Status)
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
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			updateBalance()
			newIndices := buildSortedIndices()
			mu.Lock()
			sortedIndices = newIndices
			mu.Unlock()
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
