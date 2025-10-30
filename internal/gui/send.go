package gui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/setavenger/blindbit-desktop/internal/controller"
	"github.com/setavenger/blindbit-desktop/internal/storage"
	"github.com/setavenger/blindbit-lib/logging"
	"github.com/setavenger/blindbit-lib/utils"
	"github.com/setavenger/blindbit-lib/wallet"
)

func (g *MainGUI) createSendTab() fyne.CanvasObject {
	// Form fields
	recipientEntry := widget.NewEntry()
	recipientEntry.SetPlaceHolder("Enter recipient address...")

	amountEntry := widget.NewEntry()
	amountEntry.SetPlaceHolder("Amount in satoshis (e.g., 100000)")

	feeRateEntry := widget.NewEntry()
	feeRateEntry.SetPlaceHolder("Fee rate in sat/vB (e.g., 10)")

	// Labels
	recipientLabel := widget.NewLabel("Recipient Address:")
	amountLabel := widget.NewLabel("Amount (satoshis):")
	feeRateLabel := widget.NewLabel("Fee Rate (sat/vB):")

	// Preview button
	previewBtn := widget.NewButton("Send Transaction", func() {
		g.previewTransaction(recipientEntry.Text, amountEntry.Text, feeRateEntry.Text)
	})

	// Send button (initially disabled)
	// sendBtn := widget.NewButton("Send Transaction", func() {
	// 	g.sendTransaction(recipientEntry.Text, amountEntry.Text, feeRateEntry.Text)
	// })
	// sendBtn.Disable()

	// Form layout
	form := container.NewVBox(
		recipientLabel,
		recipientEntry,
		widget.NewSeparator(),
		amountLabel,
		amountEntry,
		widget.NewSeparator(),
		feeRateLabel,
		feeRateEntry,
		widget.NewSeparator(),
		container.NewHBox(
			previewBtn,
			// sendBtn,
		),
	)

	return form
}

func (g *MainGUI) previewTransaction(recipient, amountStr, feeRateStr string) {
	// Validate inputs
	if recipient == "" {
		dialog.ShowError(fmt.Errorf("recipient address is required"), g.window)
		return
	}

	_, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		dialog.ShowError(fmt.Errorf("invalid amount: %v", err), g.window)
		return
	}

	// Parse amount as uint64 (satoshis)
	amount, err := ParseFormattedUint64(amountStr)
	if err != nil {
		dialog.ShowError(fmt.Errorf("invalid amount: %v", err), g.window)
		return
	}

	_, err = strconv.ParseUint(feeRateStr, 10, 32)
	if err != nil {
		dialog.ShowError(fmt.Errorf("invalid fee rate: %v", err), g.window)
		return
	}

	// Parse fee rate as uint32
	feeRate, err := strconv.ParseUint(feeRateStr, 10, 32)
	if err != nil {
		dialog.ShowError(fmt.Errorf("invalid fee rate: %v", err), g.window)
		return
	}

	// Convert BTC to satoshis
	// amountSatoshis := uint64(amount * 100000000)

	// Create recipient using RecipientImpl
	recipients := []wallet.Recipient{
		&wallet.RecipientImpl{
			Address: recipient,
			Amount:  amount, // Already in satoshis
			Change:  false,
		},
	}

	// Prepare transaction
	ctx := context.Background()
	txMetadata, err := g.manager.PrepareTransaction(ctx, recipients, uint32(feeRate))
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to prepare transaction: %v", err), g.window)
		return
	}

	// Show transaction details
	g.showTransactionDetails(txMetadata, recipients)
}

func (g *MainGUI) showTransactionDetails(
	txMetadata *wallet.TxMetadata,
	recipients []wallet.Recipient,
) {
	// Calculate net amount (sum of recipient amounts)
	var netAmount int64
	var totalSent uint64
	for _, recipient := range recipients {
		totalSent += recipient.GetAmount()
		if recipient.IsChange() {
			logging.L.Debug().Msg("change detected")
			continue
		}
		if recipient.GetAddress() == g.manager.GetSilentPaymentAddress() {
			logging.L.Debug().Msg("self-transfer detected")
			continue
		}
		netAmount += int64(recipient.GetAmount())
	}

	// Calculate actual fee and fee rate
	var fee uint64
	var feeRate uint64
	var feeRateFloat float64
	if txMetadata.Tx != nil {
		// Calculate total output value
		var outputSum uint64
		for _, txOut := range txMetadata.Tx.TxOut {
			outputSum += uint64(txOut.Value)
		}

		// Calculate total input value from our UTXOs
		var inputSum uint64
		for _, txIn := range txMetadata.Tx.TxIn {
			var foundUtxo bool

			// Find the UTXO being spent
			for _, utxo := range g.manager.Wallet.GetUTXOs() {
				txidMatch := bytes.Equal(
					utils.ReverseBytesCopy(utxo.Txid[:]),
					txIn.PreviousOutPoint.Hash[:],
				)
				if txidMatch && utxo.Vout == txIn.PreviousOutPoint.Index {
					inputSum += utxo.Amount
					foundUtxo = true
					break
				}
			}
			if foundUtxo {
				continue
			}

			// we should never get here as we should have found the UTXO
			err := errors.New("input utxo not found")
			logging.L.Err(err).Str("input", txIn.PreviousOutPoint.String()).Msg("UTXO not found")
			return
		}

		logging.L.Info().Uint64("inputSum", inputSum).Uint64("outputSum", outputSum).Msg("input and output sums")
		// Calculate actual fee: inputs - outputs
		fee = controller.CalculateTxFee(inputSum, outputSum)

		// Calculate vbytes and fee rate
		vbytes := controller.CalculateTxVBytes(txMetadata.Tx)
		feeRateFloat = controller.CalculateFeeRate(fee, vbytes)
		feeRate = uint64(feeRateFloat)
	}
	logging.L.Info().
		Int64("netAmount", netAmount).
		Uint64("totalSent", totalSent).
		Uint64("fee", fee).
		Uint64("feeRate", feeRate).
		Msg("transaction details")

	// Build a clean two-column summary grid
	labels := []string{"Net Amount:", "Fee:", "Fee Rate:", "Total:"}
	values := []string{
		FormatSatoshi(netAmount),
		FormatSatoshiUint64(fee),
		fmt.Sprintf("%.2f sat/vB", feeRateFloat),
		FormatSatoshiUint64(totalSent + fee),
	}

	var gridObjects []fyne.CanvasObject
	for i := range labels {
		left := widget.NewLabel(labels[i])
		left.Alignment = fyne.TextAlignLeading
		right := widget.NewLabel(values[i])
		right.Alignment = fyne.TextAlignTrailing
		gridObjects = append(gridObjects, left, right)
	}

	title := widget.NewLabelWithStyle("Transaction Details", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	confirmBtn := widget.NewButton("Confirm & Broadcast", func() {
		g.broadcastTransaction(txMetadata, recipients)
	})

	grid := container.NewGridWithColumns(2, gridObjects...)

	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		grid,
		widget.NewSeparator(),
		container.NewHBox(layout.NewSpacer(), confirmBtn, layout.NewSpacer()),
	)

	dialog.ShowCustom("Transaction Preview", "Close", content, g.window)
}

func (g *MainGUI) broadcastTransaction(
	txMetadata *wallet.TxMetadata,
	recipients []wallet.Recipient,
) {
	// Serialize transaction to hex
	var txHex string
	if txMetadata.Tx != nil {
		var err error
		txHex, err = controller.SerializeTx(txMetadata.Tx)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to serialize transaction: %v", err), g.window)
			return
		}
	} else {
		dialog.ShowError(fmt.Errorf("no transaction to broadcast"), g.window)
		return
	}

	// Broadcast transaction
	err := g.manager.BroadcastTransaction(txHex, g.manager.GetNetwork())
	if err != nil {
		logging.L.Err(err).Str("tx_hex", txHex).Msg("failed to broadcast")
		dialog.ShowError(fmt.Errorf("failed to broadcast transaction: %v", err), g.window)
		return
	}

	// todo:mark inputs as spent

	// Record transaction to history
	err = g.manager.RecordSentTransaction(txMetadata, recipients)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to record transaction: %v", err), g.window)
		return
	}

	// Show success message
	dialog.ShowInformation("Success", "Transaction broadcast successfully!", g.window)

	// save wallet in the background
	go func() {
		err := storage.SavePlain(g.manager.DataDir, g.manager)
		if err != nil {
			logging.L.Err(err).Msg("failed to save wallet")
			return
		}
	}()

	// TODO: Clear form
}
