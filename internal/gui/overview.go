package gui

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/setavenger/blindbit-lib/wallet"
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

	// Pre-compute display order (see sortedTransactionHistory). Updated on each
	// refresh tick so updateItem can index into it.
	buildSortedHistory := func() []*wallet.TxItem {
		return sortedTransactionHistory(g.manager.TransactionHistory)
	}
	var mu sync.RWMutex
	orderedHistory := buildSortedHistory()

	// Show the most recent transactions (up to 10)
	recentTxList := widget.NewList(
		func() int {
			mu.RLock()
			n := len(orderedHistory)
			mu.RUnlock()
			if n > 10 {
				return 10
			}
			return n
		},
		func() fyne.CanvasObject {
			return newTxHistoryRowGrid()
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			mu.RLock()
			historyLen := len(orderedHistory)
			var tx *wallet.TxItem
			if id < historyLen {
				tx = orderedHistory[id]
			}
			mu.RUnlock()

			if id < historyLen && tx != nil {
				c := obj.(*fyne.Container)

				txidLabel := c.Objects[0].(*widget.Label)
				heightLabel := c.Objects[1].(*widget.Label)
				amountLabel := c.Objects[2].(*widget.Label)
				statusLabel := c.Objects[3].(*widget.Label)

				formatTxRowLabels(txidLabel, heightLabel, amountLabel, statusLabel, tx)
			}
		},
	)
	recentTxScroll := container.NewScroll(recentTxList)
	recentTxScroll.SetMinSize(fyne.NewSize(440, 200))

	recentTxSection := container.NewBorder(
		container.NewVBox(
			recentTxTitleLabel,
			widget.NewSeparator(),
			headers,
			widget.NewSeparator(),
		), // top
		nil,            // bottom
		nil,            // left
		nil,            // right
		recentTxScroll, // center - fills remaining space
	)

	// --- Periodic updates ---
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			updateBalance()
			newHistory := buildSortedHistory()
			mu.Lock()
			orderedHistory = newHistory
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
	content := container.NewBorder(
		container.NewVBox(
			balanceSection,
			widget.NewSeparator(),
			scanSection,
			widget.NewSeparator(),
		), // top
		nil,            // bottom
		nil,            // left
		nil,            // right
		recentTxSection, // center - fills remaining dashboard space
	)

	return content
}
