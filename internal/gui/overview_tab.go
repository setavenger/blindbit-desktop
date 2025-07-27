package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createOverviewTab creates the overview tab
func (g *MainGUI) createOverviewTab() *fyne.Container {
	// Enhanced UTXO table with proper table widget and built-in headers
	g.utxoList = widget.NewTable(
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
	g.utxoList.ShowHeaderRow = true
	g.utxoList.CreateHeader = func() fyne.CanvasObject {
		headerLabel := widget.NewLabel("Header")
		headerLabel.TextStyle = fyne.TextStyle{Bold: true}
		return headerLabel
	}
	g.utxoList.UpdateHeader = func(id widget.TableCellID, template fyne.CanvasObject) {
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
	g.utxoList.SetColumnWidth(0, 200) // TxID:Vout column
	g.utxoList.SetColumnWidth(1, 120) // Amount column
	g.utxoList.SetColumnWidth(2, 100) // State column

	// Set row height to ensure proper table display
	g.utxoList.SetRowHeight(0, 40) // Set a reasonable row height for all rows

	// Scan controls with better UX
	scanButton := widget.NewButtonWithIcon("Start Scanning", theme.MediaPlayIcon(), func() {
		g.startScanning()
	})

	stopButton := widget.NewButtonWithIcon("Stop Scanning", theme.MediaStopIcon(), func() {
		g.stopScanning()
	})

	// Rescan button with dropdown for different options
	rescanButton := widget.NewButtonWithIcon("Rescan", theme.ViewRefreshIcon(), func() {
		g.showRescanDialog()
	})

	// Group scan controls
	scanControls := container.NewHBox(
		scanButton,
		stopButton,
		widget.NewSeparator(),
		rescanButton,
	)

	// Network info
	networkInfo := widget.NewLabel(fmt.Sprintf("Network: %s", string(g.walletManager.GetNetwork())))

	// Create info section
	infoSection := container.NewVBox(
		networkInfo,
		g.oracleInfoLabel,
		widget.NewSeparator(),
	)

	// UTXO section with proper layout - no need for separate headers now
	utxoSection := container.NewStack(
		widget.NewLabel("Recent UTXOs"),
		widget.NewSeparator(),
		g.utxoList, // Use NewStack to allow table to expand beyond MinSize
	)

	return container.NewBorder(
		container.NewVBox(
			infoSection,
			scanControls,
			widget.NewSeparator(),
		),
		nil, nil, nil,
		utxoSection,
	)
}

// showRescanDialog shows a dialog with rescan options
func (g *MainGUI) showRescanDialog() {
	// Create rescan options
	options := []string{
		"Rescan from current height (preserve UTXOs)",
		"Force rescan from current height (clear UTXOs)",
		"Rescan from birth height (preserve UTXOs)",
		"Force rescan from birth height (clear UTXOs)",
		"Rescan from specific height...",
	}

	// Create a select widget
	selectWidget := widget.NewSelect(options, nil)
	selectWidget.SetSelected(options[0]) // Default selection

	// Create the dialog content
	content := container.NewVBox(
		widget.NewLabel("Choose rescan option:"),
		selectWidget,
		widget.NewLabel("Note: Force rescan will clear all existing UTXOs"),
	)

	// Create the dialog
	dialog.ShowCustomConfirm("Rescan Options", "Rescan", "Cancel", content, func(confirmed bool) {
		if !confirmed {
			return
		}

		selected := selectWidget.Selected
		if selected == "" {
			return
		}

		// Handle different rescan options
		switch selected {
		case "Rescan from current height (preserve UTXOs)":
			g.rescanFromCurrentHeight(false)
		case "Force rescan from current height (clear UTXOs)":
			g.rescanFromCurrentHeight(true)
		case "Rescan from birth height (preserve UTXOs)":
			g.rescanFromBirthHeight(false)
		case "Force rescan from birth height (clear UTXOs)":
			g.rescanFromBirthHeight(true)
		case "Rescan from specific height...":
			g.showSpecificHeightDialog()
		}
	}, g.window)
}

// showSpecificHeightDialog shows a dialog to input a specific height
func (g *MainGUI) showSpecificHeightDialog() {
	heightEntry := widget.NewEntry()
	heightEntry.SetPlaceHolder("Enter block height (e.g., 840000)")

	// Create radio buttons for preserve/clear options
	preserveRadio := widget.NewRadioGroup([]string{"Preserve existing UTXOs", "Clear all UTXOs"}, nil)
	preserveRadio.SetSelected("Preserve existing UTXOs")

	content := container.NewVBox(
		widget.NewLabel("Enter block height to rescan from:"),
		heightEntry,
		widget.NewSeparator(),
		widget.NewLabel("UTXO handling:"),
		preserveRadio,
	)

	dialog.ShowCustomConfirm("Rescan from Height", "Rescan", "Cancel", content, func(confirmed bool) {
		if !confirmed {
			return
		}

		height := heightEntry.Text
		if height == "" {
			dialog.ShowError(fmt.Errorf("please enter a block height"), g.window)
			return
		}

		// Parse height
		var heightUint uint64
		if _, err := fmt.Sscanf(height, "%d", &heightUint); err != nil {
			dialog.ShowError(fmt.Errorf("invalid height: %s", height), g.window)
			return
		}

		// Perform rescan
		clearUTXOs := preserveRadio.Selected == "Clear all UTXOs"
		g.rescanFromHeight(heightUint, clearUTXOs)
	}, g.window)
}

// rescanFromCurrentHeight rescans from the current scan height
func (g *MainGUI) rescanFromCurrentHeight(clearUTXOs bool) {
	currentHeight := g.walletManager.GetScanHeight()
	if clearUTXOs {
		if err := g.walletManager.ForceRescanFromHeight(uint64(currentHeight)); err != nil {
			dialog.ShowError(fmt.Errorf("failed to force rescan: %v", err), g.window)
			return
		}
		dialog.ShowInformation("Rescan", "Force rescan started from current height", g.window)
	} else {
		if err := g.walletManager.RescanFromHeight(uint64(currentHeight)); err != nil {
			dialog.ShowError(fmt.Errorf("failed to rescan: %v", err), g.window)
			return
		}
		dialog.ShowInformation("Rescan", "Rescan started from current height", g.window)
	}

	// Update UI
	g.updateWalletInfo()
}

// rescanFromBirthHeight rescans from the birth height
func (g *MainGUI) rescanFromBirthHeight(clearUTXOs bool) {
	birthHeight := g.walletManager.GetBirthHeight()
	if clearUTXOs {
		if err := g.walletManager.ForceRescanFromHeight(birthHeight); err != nil {
			dialog.ShowError(fmt.Errorf("failed to force rescan: %v", err), g.window)
			return
		}
		dialog.ShowInformation("Rescan", "Force rescan started from birth height", g.window)
	} else {
		if err := g.walletManager.RescanFromHeight(birthHeight); err != nil {
			dialog.ShowError(fmt.Errorf("failed to rescan: %v", err), g.window)
			return
		}
		dialog.ShowInformation("Rescan", "Rescan started from birth height", g.window)
	}

	// Update UI
	g.updateWalletInfo()
}

// rescanFromHeight rescans from a specific height
func (g *MainGUI) rescanFromHeight(height uint64, clearUTXOs bool) {
	if clearUTXOs {
		if err := g.walletManager.ForceRescanFromHeight(height); err != nil {
			dialog.ShowError(fmt.Errorf("failed to force rescan: %v", err), g.window)
			return
		}
		dialog.ShowInformation("Rescan", fmt.Sprintf("Force rescan started from height %d", height), g.window)
	} else {
		if err := g.walletManager.RescanFromHeight(height); err != nil {
			dialog.ShowError(fmt.Errorf("failed to rescan: %v", err), g.window)
			return
		}
		dialog.ShowInformation("Rescan", fmt.Sprintf("Rescan started from height %d", height), g.window)
	}

	// Update UI
	g.updateWalletInfo()
}

// startScanning starts the scanning process
func (g *MainGUI) startScanning() {
	if err := g.walletManager.StartScanning(); err != nil {
		dialog.ShowError(fmt.Errorf("failed to start scanning: %v", err), g.window)
		return
	}

	// Update UI to show scanning status
	g.updateWalletInfo()
	dialog.ShowInformation("Scanning", "Scanning started successfully", g.window)
}

// stopScanning stops the scanning process
func (g *MainGUI) stopScanning() {
	g.walletManager.StopScanning()

	// Update UI to show stopped status
	g.updateWalletInfo()
	dialog.ShowInformation("Scanning", "Scanning stopped", g.window)
}
