package gui

import (
	"fmt"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/setavenger/blindbit-desktop/internal/manager"
)

// TransactionOverviewTab represents the transaction overview tab
type TransactionOverviewTab struct {
	walletManager *manager.Manager
	table         *widget.Table
	data          []*manager.TransactionHistoryGUI
	refreshButton *widget.Button
}

// NewTransactionOverviewTab creates a new transaction overview tab
func NewTransactionOverviewTab(walletManager *manager.Manager) *TransactionOverviewTab {
	tab := &TransactionOverviewTab{
		walletManager: walletManager,
		data:          []*manager.TransactionHistoryGUI{},
	}

	tab.createContent()
	return tab
}

// createContent creates the transaction overview content
func (t *TransactionOverviewTab) createContent() {
	// Create refresh button
	t.refreshButton = widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		t.RefreshTransactions()
	})

	// Create transaction table
	t.table = widget.NewTable(
		func() (int, int) {
			return len(t.data), 6 // 6 columns: Type, Amount, Fee, Status, Block Height, Description
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			if id.Row >= len(t.data) {
				return
			}

			tx := t.data[id.Row]
			label := obj.(*widget.Label)

			switch id.Col {
			case 0: // Type
				label.SetText(tx.Type)
				if tx.Type == manager.TransactionTypeIncoming {
					label.TextStyle = fyne.TextStyle{Bold: true}
				} else if tx.Type == manager.TransactionTypeSelfTransfer {
					label.TextStyle = fyne.TextStyle{Italic: true}
				} else {
					label.TextStyle = fyne.TextStyle{}
				}
			case 1: // Amount
				label.SetText(tx.Amount)
				if tx.Type == manager.TransactionTypeIncoming {
					label.TextStyle = fyne.TextStyle{Bold: true}
				} else if tx.Type == manager.TransactionTypeSelfTransfer {
					label.TextStyle = fyne.TextStyle{Italic: true}
				} else {
					label.TextStyle = fyne.TextStyle{}
				}
			case 2: // Fee
				label.SetText(tx.Fee)
			case 3: // Status
				label.SetText(tx.Confirmed)
				if tx.Confirmed == "Confirmed" {
					label.TextStyle = fyne.TextStyle{Bold: true}
				} else {
					label.TextStyle = fyne.TextStyle{}
				}
			case 4: // Block Height
				label.SetText(tx.BlockHeight)
			case 5: // Description
				label.SetText(tx.Description)
			}
		})

	// Enable header row and set up custom headers
	t.table.ShowHeaderRow = true
	t.table.CreateHeader = func() fyne.CanvasObject {
		headerLabel := widget.NewLabel("Header")
		headerLabel.TextStyle = fyne.TextStyle{Bold: true}
		return headerLabel
	}
	t.table.UpdateHeader = func(id widget.TableCellID, template fyne.CanvasObject) {
		label := template.(*widget.Label)
		switch id.Col {
		case 0:
			label.SetText("Type")
		case 1:
			label.SetText("Amount")
		case 2:
			label.SetText("Fee")
		case 3:
			label.SetText("Status")
		case 4:
			label.SetText("Block Height")
		case 5:
			label.SetText("Description")
		}
	}

	// Set column widths
	t.table.SetColumnWidth(0, 80)  // Type
	t.table.SetColumnWidth(1, 120) // Amount
	t.table.SetColumnWidth(2, 80)  // Fee
	t.table.SetColumnWidth(3, 100) // Status
	t.table.SetColumnWidth(4, 100) // Block Height
	t.table.SetColumnWidth(5, 200) // Description

	// Set row height to ensure proper table display
	t.table.SetRowHeight(0, 40)

	// Add click handler for transaction details
	t.table.OnSelected = func(id widget.TableCellID) {
		if id.Row >= 0 && id.Row < len(t.data) {
			t.showTransactionDetails(t.data[id.Row])
		}
	}

	// Initial refresh
	t.RefreshTransactions()
}

// GetContent returns the transaction overview content
func (t *TransactionOverviewTab) GetContent() *fyne.Container {
	return container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Transaction History"),
			widget.NewSeparator(),
		),
		container.NewHBox(
			t.refreshButton,
			widget.NewLabel("Click on a transaction to view details"),
		),
		nil, nil,
		t.table,
	)
}

// RefreshTransactions refreshes the transaction data
func (t *TransactionOverviewTab) RefreshTransactions() {
	history, err := t.walletManager.GetTransactionHistoryForGUI()
	if err != nil {
		fmt.Printf("Error getting transaction history: %v\n", err)
		return
	}

	// Sort transactions by block height (highest first)
	sort.Slice(history, func(i, j int) bool {
		// Parse block heights for comparison
		blockHeightI := history[i].BlockHeight
		blockHeightJ := history[j].BlockHeight

		// If both are pending (empty string), sort by txid
		if blockHeightI == "Pending" && blockHeightJ == "Pending" {
			return history[i].TxID > history[j].TxID
		}

		// If one is pending, put it at the end
		if blockHeightI == "Pending" {
			return false
		}
		if blockHeightJ == "Pending" {
			return true
		}

		// Both have block heights, sort by height (descending)
		return blockHeightI > blockHeightJ
	})

	t.data = history
	t.table.Refresh()
}

// showTransactionDetails shows details for a specific transaction
func (t *TransactionOverviewTab) showTransactionDetails(tx *manager.TransactionHistoryGUI) {
	// Create a new window for transaction details
	txWindow := fyne.CurrentApp().NewWindow("Transaction Details")
	txWindow.Resize(fyne.NewSize(600, 500))
	txWindow.SetFixedSize(false)

	// Create content with better layout
	content := container.NewVBox()

	// Header
	header := widget.NewLabel("Transaction Details")
	header.TextStyle = fyne.TextStyle{Bold: true}
	content.Add(header)
	content.Add(widget.NewSeparator())

	// Transaction ID with copy button
	txidContainer := container.NewHBox(
		widget.NewLabel("Transaction ID:"),
		widget.NewLabel(tx.TxID),
		widget.NewButtonWithIcon("Copy", theme.ContentCopyIcon(), func() {
			txWindow.Clipboard().SetContent(tx.TxID)
		}),
	)
	content.Add(txidContainer)

	// Type
	typeContainer := container.NewHBox(
		widget.NewLabel("Type:"),
		widget.NewLabel(tx.Type),
	)
	content.Add(typeContainer)

	// Amount
	amountContainer := container.NewHBox(
		widget.NewLabel("Amount:"),
		widget.NewLabel(tx.Amount),
	)
	content.Add(amountContainer)

	// Fee
	feeContainer := container.NewHBox(
		widget.NewLabel("Fee:"),
		widget.NewLabel(tx.Fee),
	)
	content.Add(feeContainer)

	// Status
	statusContainer := container.NewHBox(
		widget.NewLabel("Status:"),
		widget.NewLabel(tx.Confirmed),
	)
	content.Add(statusContainer)

	// Block Height
	blockHeightContainer := container.NewHBox(
		widget.NewLabel("Block Height:"),
		widget.NewLabel(tx.BlockHeight),
	)
	content.Add(blockHeightContainer)

	// Description
	descriptionContainer := container.NewVBox(
		widget.NewLabel("Description:"),
		widget.NewLabel(tx.Description),
	)
	content.Add(descriptionContainer)

	content.Add(widget.NewSeparator())

	// Action buttons
	buttons := container.NewHBox(
		widget.NewButtonWithIcon("Copy TXID", theme.ContentCopyIcon(), func() {
			txWindow.Clipboard().SetContent(tx.TxID)
		}),
		widget.NewButtonWithIcon("Copy Amount", theme.ContentCopyIcon(), func() {
			txWindow.Clipboard().SetContent(tx.Amount)
		}),
		widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() {
			txWindow.Close()
		}),
	)
	content.Add(buttons)

	// Add scroll container
	scroll := container.NewScroll(content)
	scroll.SetMinSize(fyne.NewSize(580, 480))

	txWindow.SetContent(scroll)
	txWindow.Show()
}
