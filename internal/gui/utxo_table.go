package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// createUTXOTable creates a reusable UTXO table widget
func (g *MainGUI) createUTXOTable() *widget.Table {
	utxoList := widget.NewTable(
		func() (int, int) {
			// todo: add option to filter by state
			length := g.UtxoCount()
			return length, 5 // 5 columns: TxID:Vout, Amount, State, Label, Timestamp
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("wide content")
			label.Wrapping = fyne.TextWrapWord
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
				case 3: // Label
					label.SetText(utxo.Label)
				case 4: // Timestamp
					label.SetText(utxo.Timestamp)
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
		case 3:
			label.SetText("Label")
		case 4:
			label.SetText("Timestamp")
		}
	}

	// Set column widths for better layout
	utxoList.SetColumnWidth(0, 180) // TxID:Vout column
	utxoList.SetColumnWidth(1, 100) // Amount column
	utxoList.SetColumnWidth(2, 80)  // State column
	utxoList.SetColumnWidth(3, 60)  // Label column
	utxoList.SetColumnWidth(4, 140) // Timestamp column

	// Set row height to ensure proper table display
	utxoList.SetRowHeight(0, 40)

	return utxoList
}
