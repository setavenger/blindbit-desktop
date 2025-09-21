package gui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/setavenger/blindbit-desktop/internal/manager"
	"github.com/setavenger/blindbit-lib/logging"
)

// TransactionDetailsTab represents the transaction details view
type TransactionDetailsTab struct {
	window        fyne.Window
	walletManager WalletManager
	result        *manager.TransactionResult
}

// WalletManager interface for wallet operations
type WalletManager interface {
	BroadcastTransaction(hex string) error
}

// NewTransactionDetailsTab creates a new transaction details tab
func NewTransactionDetailsTab(window fyne.Window, walletManager WalletManager, result *manager.TransactionResult) *TransactionDetailsTab {
	return &TransactionDetailsTab{
		window:        window,
		walletManager: walletManager,
		result:        result,
	}
}

// CreateTransactionDetailsView creates the transaction details view
func (td *TransactionDetailsTab) CreateTransactionDetailsView() *fyne.Container {
	// Transaction overview section
	overviewCard := td.createOverviewCard()

	// Inputs section
	inputsCard := td.createInputsCard()

	// Outputs section
	outputsCard := td.createOutputsCard()

	// Raw data section
	rawDataCard := td.createRawDataCard()

	// Action buttons
	actionButtons := td.createActionButtons()

	// Main container with scroll
	content := container.NewVBox(
		widget.NewLabelWithStyle("Transaction Details", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		overviewCard,
		widget.NewSeparator(),
		inputsCard,
		widget.NewSeparator(),
		outputsCard,
		widget.NewSeparator(),
		rawDataCard,
		widget.NewSeparator(),
		actionButtons,
	)

	// Create scrollable container
	scrollContainer := container.NewScroll(content)
	scrollContainer.SetMinSize(fyne.NewSize(600, 400))

	return container.NewPadded(scrollContainer)
}

// createOverviewCard creates the transaction overview card
func (td *TransactionDetailsTab) createOverviewCard() *fyne.Container {
	txidLabel := widget.NewLabel("Transaction ID:")
	// Use styled read-only entry with proper text color
	txidDisplay := createReadOnlyEntry(td.result.TxID, true)

	feeLabel := widget.NewLabel(fmt.Sprintf("Fee: %d sats", td.result.Fee))
	feeRateLabel := widget.NewLabel(fmt.Sprintf("Effective Fee Rate: %.2f sats/vB", td.result.EffectiveFeeRate))
	sizeLabel := widget.NewLabel(fmt.Sprintf("Size: %d bytes", td.result.Size))
	vsizeLabel := widget.NewLabel(fmt.Sprintf("VSize: %d vbytes", td.result.VSize))

	totalInputLabel := widget.NewLabel(fmt.Sprintf("Total Input: %d sats", td.result.TotalInput))
	totalOutputLabel := widget.NewLabel(fmt.Sprintf("Total Output: %d sats", td.result.TotalOutput))

	// Small, subtle copy button - just icon, no text
	copyTxidButton := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		td.window.Clipboard().SetContent(td.result.TxID)
	})

	// Create a horizontal layout for TxID with copy button
	// Use NewBorder to give the label most of the space
	txidRow := container.NewBorder(nil, nil, nil, copyTxidButton, txidDisplay)

	return container.NewVBox(
		widget.NewLabelWithStyle("Overview", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		txidLabel,
		txidRow,
		feeLabel,
		feeRateLabel,
		sizeLabel,
		vsizeLabel,
		totalInputLabel,
		totalOutputLabel,
	)
}

// createInputsCard creates the inputs card
func (td *TransactionDetailsTab) createInputsCard() *fyne.Container {
	inputsLabel := widget.NewLabelWithStyle("Inputs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	var inputItems []fyne.CanvasObject
	for i, input := range td.result.Inputs {
		inputText := fmt.Sprintf("Input %d:\n  TxID: %s\n  Vout: %d",
			i+1,
			input.PreviousOutPoint.Hash.String(),
			input.PreviousOutPoint.Index)

		inputLabel := widget.NewLabel(inputText)
		inputLabel.Wrapping = fyne.TextWrapWord
		inputItems = append(inputItems, inputLabel)
	}

	return container.NewVBox(append([]fyne.CanvasObject{inputsLabel}, inputItems...)...)
}

// createOutputsCard creates the outputs card
func (td *TransactionDetailsTab) createOutputsCard() *fyne.Container {
	outputsLabel := widget.NewLabelWithStyle("Outputs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	var outputItems []fyne.CanvasObject
	for i, output := range td.result.Outputs {
		outputText := fmt.Sprintf("Output %d:\n  Amount: %d sats\n  Script: %x",
			i+1,
			output.Value,
			output.PkScript)

		outputLabel := widget.NewLabel(outputText)
		outputLabel.Wrapping = fyne.TextWrapWord
		outputItems = append(outputItems, outputLabel)
	}

	return container.NewVBox(append([]fyne.CanvasObject{outputsLabel}, outputItems...)...)
}

// createRawDataCard creates the raw data card
func (td *TransactionDetailsTab) createRawDataCard() *fyne.Container {
	rawDataLabel := widget.NewLabelWithStyle("Raw Data", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Hex data
	hexLabel := widget.NewLabel("Transaction Hex:")
	// Use styled read-only multi-line entry with proper text color
	hexEntry := createReadOnlyMultiLineEntry(td.result.Hex, true)

	// Small, subtle copy button for hex - just icon
	copyHexButton := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		td.window.Clipboard().SetContent(td.result.Hex)
	})

	// Create horizontal layout for hex with copy button
	// Use NewBorder to give the entry field most of the space
	hexRow := container.NewBorder(nil, nil, nil, copyHexButton, hexEntry)

	// PSBT data (if available)
	var psbtContainer fyne.CanvasObject
	if td.result.PSBT != "" {
		psbtLabel := widget.NewLabel("PSBT:")
		// Use styled read-only multi-line entry with proper text color
		psbtEntry := createReadOnlyMultiLineEntry(td.result.PSBT, true)

		// Small, subtle copy button for PSBT - just icon
		copyPsbtButton := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
			td.window.Clipboard().SetContent(td.result.PSBT)
		})

		// Use NewBorder to give the entry field most of the space
		psbtRow := container.NewBorder(nil, nil, nil, copyPsbtButton, psbtEntry)

		psbtContainer = container.NewVBox(
			psbtLabel,
			psbtRow,
		)
	}

	hexContainer := container.NewVBox(
		hexLabel,
		hexRow,
	)

	if psbtContainer != nil {
		return container.NewVBox(rawDataLabel, hexContainer, psbtContainer)
	}

	return container.NewVBox(rawDataLabel, hexContainer)
}

// createActionButtons creates the action buttons
func (td *TransactionDetailsTab) createActionButtons() *fyne.Container {
	broadcastButton := widget.NewButtonWithIcon("Broadcast Transaction", theme.MailSendIcon(), func() {
		td.broadcastTransaction()
	})

	closeButton := widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() {
		td.window.Close()
	})

	return container.NewHBox(broadcastButton, closeButton)
}

// broadcastTransaction broadcasts the transaction
func (td *TransactionDetailsTab) broadcastTransaction() {
	// Show confirmation dialog
	dialog.ShowConfirm("Broadcast Transaction",
		"Are you sure you want to broadcast this transaction to the network?",
		func(confirmed bool) {
			if confirmed {
				err := td.walletManager.BroadcastTransaction(td.result.Hex)
				if err != nil {
					logging.L.Err(err).Msg("failed to broadcast transaction")
					dialog.ShowError(fmt.Errorf("failed to broadcast transaction: %v", err), td.window)
					return
				}

				dialog.ShowInformation("Transaction Broadcasted",
					fmt.Sprintf("Transaction %s has been successfully broadcasted to the network!", td.result.TxID),
					td.window)
			}
		}, td.window)
}

// formatAmount formats amount in sats
func formatAmount(amount int64) string {
	return strconv.FormatInt(amount, 10) + " sats"
}

// createReadOnlyEntry creates a styled read-only entry with proper text color
func createReadOnlyEntry(text string, monospace bool) *widget.Label {
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	if monospace {
		label.TextStyle = fyne.TextStyle{Monospace: true}
	}
	return label
}

// createReadOnlyMultiLineEntry creates a styled read-only multi-line entry with proper text color
func createReadOnlyMultiLineEntry(text string, monospace bool) *widget.Label {
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	if monospace {
		label.TextStyle = fyne.TextStyle{Monospace: true}
	}
	label.Resize(fyne.NewSize(0, 120))
	return label
}
