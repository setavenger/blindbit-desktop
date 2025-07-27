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

	// Update statistics function
	updateStats := func() {
		total, unspent, spent := g.walletManager.GetUTXOStats()
		balance := g.walletManager.GetBalance()

		totalLabel.SetText(fmt.Sprintf("Total UTXOs: %d", total))
		unspentLabel.SetText(fmt.Sprintf("Unspent UTXOs: %d", unspent))
		spentLabel.SetText(fmt.Sprintf("Spent UTXOs: %d", spent))
		balanceLabel.SetText(fmt.Sprintf("Total Balance: %d sats", balance))

		totalLabel.Refresh()
		unspentLabel.Refresh()
		spentLabel.Refresh()
		balanceLabel.Refresh()
	}

	// Enhanced UTXO list with proper table layout and built-in headers
	utxoList := widget.NewTable(
		func() (int, int) {
			length := len(g.utxoData)
			// fmt.Println("Length:", length)
			return length, 3 // 3 columns: TxID:Vout, Amount, State
		},
		func() fyne.CanvasObject {
			// Create a cell template with proper sizing
			label := widget.NewLabel("wide content")
			label.Wrapping = fyne.TextWrapWord
			// Create a container with padding to ensure proper cell sizing
			cellContainer := container.NewPadded(label)
			return cellContainer
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			cellContainer := o.(*fyne.Container)
			label := cellContainer.Objects[0].(*widget.Label)
			if i.Row < len(g.utxoData) {
				utxo := g.utxoData[i.Row]
				switch i.Col {
				case 0: // TxID:Vout combined
					txid := utxo.TxID
					if len(txid) > 8 {
						txid = txid[:8] + "..."
					}
					label.SetText(fmt.Sprintf("%s:%d", txid, utxo.Vout))
				case 1: // Amount
					label.SetText(utxo.Amount)
				case 2: // State
					label.SetText(utxo.State)
				}
			}
		},
	)

	// Enable header row and set up custom headers
	utxoList.ShowHeaderRow = true
	utxoList.CreateHeader = func() fyne.CanvasObject {
		headerLabel := widget.NewLabel("Header")
		headerLabel.TextStyle = fyne.TextStyle{Bold: true}
		return headerLabel
	}
	utxoList.UpdateHeader = func(id widget.TableCellID, template fyne.CanvasObject) {
		label := template.(*widget.Label)
		switch id.Col {
		case 0:
			label.SetText("Transaction ID:Vout")
		case 1:
			label.SetText("Amount (sats)")
		case 2:
			label.SetText("State")
		}
	}

	// Set column widths for better layout
	utxoList.SetColumnWidth(0, 200) // TxID:Vout column
	utxoList.SetColumnWidth(1, 120) // Amount column
	utxoList.SetColumnWidth(2, 100) // State column

	// Set row height to ensure proper table display
	utxoList.SetRowHeight(0, 40) // Set a reasonable row height for all rows

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
