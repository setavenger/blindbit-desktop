package gui

import (
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/setavenger/blindbit-lib/logging"
	"github.com/setavenger/blindbit-lib/wallet"
)

func (g *MainGUI) createUTXOsTab() fyne.CanvasObject {
	// Main title
	titleLabel := widget.NewLabel("UTXOs")
	titleLabel.TextStyle.Bold = true

	// Balance display
	balanceLabel := widget.NewLabel("Balance: 0 sats")
	balanceLabel.TextStyle.Bold = true

	// Debug info label
	debugLabel := widget.NewLabel("Debug: Loading UTXOs...")
	debugLabel.TextStyle.Italic = true

	// Filter checkbox for unspent UTXOs only
	unspentOnlyCheck := widget.NewCheck("Show only unspent UTXOs", nil)
	unspentOnlyCheck.SetChecked(true) // Default to showing only unspent

	// Create table headers with proper alignment
	headers := container.NewGridWithColumns(5,
		widget.NewLabel("Outpoint"),
		widget.NewLabel("Label"),
		widget.NewLabel("Value"),
		widget.NewLabel("Height"),
		widget.NewLabel("State"),
	)

	// UTXO list with proper columns
	utxoList := widget.NewList(
		func() int {
			utxos := g.getFilteredUTXOs(unspentOnlyCheck.Checked)
			return len(utxos)
		},
		func() fyne.CanvasObject {
			// Create a container with 5 labels for each row
			return container.NewGridWithColumns(5,
				widget.NewLabel(""), // Outpoint
				widget.NewLabel(""), // Label
				widget.NewLabel(""), // Value
				widget.NewLabel(""), // Height
				widget.NewLabel(""), // State
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			utxos := g.getFilteredUTXOs(unspentOnlyCheck.Checked)
			if id < len(utxos) {
				utxo := utxos[id]
				container := obj.(*fyne.Container)

				// Get the labels from the container
				outpointLabel := container.Objects[0].(*widget.Label)
				labelLabel := container.Objects[1].(*widget.Label)
				valueLabel := container.Objects[2].(*widget.Label)
				heightLabel := container.Objects[3].(*widget.Label)
				stateLabel := container.Objects[4].(*widget.Label)

				// Set the data
				txidHex := hex.EncodeToString(utxo.Txid[:])
				outpointLabel.SetText(fmt.Sprintf("%.8s...:%d", txidHex, utxo.Vout))

				if utxo.Label != nil {
					labelLabel.SetText(fmt.Sprintf("%d", utxo.Label.M))
				} else {
					labelLabel.SetText("-")
				}

				valueLabel.SetText(FormatSatoshiUint64(utxo.Amount))
				heightLabel.SetText(FormatHeight(utxo.Height))
				stateLabel.SetText(utxo.State.String())
			}
		},
	)

	// Refresh button
	refreshBtn := widget.NewButton("Refresh UTXOs", func() {
		g.refreshUTXOs(utxoList, debugLabel)
	})

	// Update initial values
	g.updateBalance(balanceLabel)
	g.updateDebugInfo(debugLabel)

	// Set up periodic updates
	go g.startPeriodicUTXOUpdates(balanceLabel, utxoList, debugLabel, unspentOnlyCheck)

	// Set up real-time updates from scanner
	go g.startRealTimeUTXOUpdates(balanceLabel, utxoList, debugLabel, unspentOnlyCheck)

	// Filter change handler
	unspentOnlyCheck.OnChanged = func(checked bool) {
		utxoList.Refresh()
		g.updateBalance(balanceLabel)
		g.updateDebugInfo(debugLabel)
	}

	// Main content
	content := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
		balanceLabel,
		widget.NewSeparator(),
		debugLabel,
		widget.NewSeparator(),
		container.NewHBox(unspentOnlyCheck, refreshBtn),
		widget.NewSeparator(),
		headers,
		widget.NewSeparator(),
	)

	// Create a scrollable container for the list
	scrollContainer := container.NewScroll(utxoList)
	scrollContainer.SetMinSize(fyne.NewSize(400, 300)) // Set minimum size

	// Add the scrollable list to the main content
	content.Add(scrollContainer)

	return content
}

func (g *MainGUI) updateBalance(balanceLabel *widget.Label) {
	// Calculate balance from unspent UTXOs only
	unspentUTXOs := g.manager.GetUnspentUTXOsSorted()
	var total uint64
	for _, utxo := range unspentUTXOs {
		total += utxo.Amount
	}
	balanceLabel.SetText("Balance: " + FormatSatoshiUint64(total))
}

// updateDebugInfo updates the debug label with current UTXO information
func (g *MainGUI) updateDebugInfo(debugLabel *widget.Label) {
	allUTXOs := g.manager.GetUTXOsSorted()
	unspentUTXOs := g.manager.GetUnspentUTXOsSorted()

	// Check if wallet is properly initialized
	walletInfo := "Wallet: "
	if g.manager.Wallet == nil {
		walletInfo += "NOT INITIALIZED"
	} else {
		walletInfo += "INITIALIZED"
	}

	// Debug: let's also check raw wallet data
	rawUTXOs := g.manager.Wallet.GetUTXOs()

	text := fmt.Sprintf(
		"Debug: %s | %d total UTXOs, %d unspent UTXOs (raw: %d)",
		walletInfo,
		len(allUTXOs),
		len(unspentUTXOs),
		len(rawUTXOs),
	)

	debugLabel.SetText(text)
}

// getFilteredUTXOs returns UTXOs based on the filter setting, sorted by height (descending)
func (g *MainGUI) getFilteredUTXOs(unspentOnly bool) []*wallet.OwnedUTXO {
	var utxos []*wallet.OwnedUTXO
	if unspentOnly {
		utxos = g.manager.Wallet.GetUTXOs(wallet.StateUnspent)
	} else {
		utxos = g.manager.Wallet.GetUTXOs()
	}

	// Sort by height in descending order (newest first)
	sort.Slice(utxos, func(i, j int) bool {
		return utxos[i].Height > utxos[j].Height
	})

	return utxos
}

// startPeriodicUTXOUpdates sets up periodic refresh of UTXO data
func (g *MainGUI) startPeriodicUTXOUpdates(
	balanceLabel *widget.Label,
	utxoList *widget.List,
	debugLabel *widget.Label,
	filterCheck *widget.Check,
) {
	ticker := time.NewTicker(60 * time.Second) // Update every 60 seconds (reduced frequency)
	defer ticker.Stop()

	for range ticker.C {
		// Update UI components
		g.updateBalance(balanceLabel)
		g.updateDebugInfo(debugLabel)
		utxoList.Refresh()
	}
}

// startRealTimeUTXOUpdates listens for new UTXOs from the scanner
// bug: cannot be used with owned chan.
// More than one receiver make things get lost on either receiver A or B
//
// Deprecated: we need a specific broadcast channel to fix this.
func (g *MainGUI) startRealTimeUTXOUpdates(
	balanceLabel *widget.Label,
	utxoList *widget.List,
	debugLabel *widget.Label,
	filterCheck *widget.Check,
) {
	// tickerChan := time.NewTicker(10 * time.Second)
	// // Listen to the manager's UTXO channel for real-time updates
	// if g.manager.OwnedUTXOsChan != nil {
	// 	for range tickerChan.C {
	// 		// New UTXO found, update UI immediately
	// 		g.updateBalance(balanceLabel)
	// 		g.updateDebugInfo(debugLabel)
	// 		utxoList.Refresh()
	// 		logging.L.Info().Msg("UTXO list updated due to new UTXO discovery")
	// 	}
	// }
}

func (g *MainGUI) refreshUTXOs(utxoList *widget.List, debugLabel *widget.Label) {
	// Refresh the UTXO list
	logging.L.Info().Msg("Refreshing UTXO list")
	g.updateDebugInfo(debugLabel)
	utxoList.Refresh()
}
