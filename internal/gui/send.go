package gui

import (
	"context"
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

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
	feeRateEntry.SetText("10") // Default fee rate

	// Labels
	recipientLabel := widget.NewLabel("Recipient Address:")
	amountLabel := widget.NewLabel("Amount (satoshis):")
	feeRateLabel := widget.NewLabel("Fee Rate (sat/vB):")

	// Preview button
	previewBtn := widget.NewButton("Preview Transaction", func() {
		g.previewTransaction(recipientEntry.Text, amountEntry.Text, feeRateEntry.Text)
	})

	// Send button (initially disabled)
	sendBtn := widget.NewButton("Send Transaction", func() {
		g.sendTransaction(recipientEntry.Text, amountEntry.Text, feeRateEntry.Text)
	})
	sendBtn.Disable()

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
		container.NewHBox(previewBtn, sendBtn),
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
	amount, err := strconv.ParseUint(amountStr, 10, 64)
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

func (g *MainGUI) showTransactionDetails(txMetadata *wallet.TxMetadata, recipients []wallet.Recipient) {
	// Calculate net amount (sum of recipient amounts)
	var netAmount int64
	for _, recipient := range recipients {
		netAmount += int64(recipient.GetAmount())
	}

	detailsText := fmt.Sprintf(`
Transaction Details:

Recipients: %d
Net Amount: %d sats
Fee: %d sats (estimated)
Total: %d sats

Transaction prepared successfully
`,
		len(recipients),
		netAmount,
		// TODO: Calculate actual fee from transaction
		0,
		netAmount,
	)

	detailsLabel := widget.NewLabel(detailsText)
	detailsLabel.Wrapping = fyne.TextWrapWord

	confirmBtn := widget.NewButton("Confirm & Broadcast", func() {
		g.broadcastTransaction(txMetadata, netAmount)
	})

	cancelBtn := widget.NewButton("Cancel", func() {
		// Close dialog
	})

	content := container.NewVBox(
		detailsLabel,
		widget.NewSeparator(),
		container.NewHBox(cancelBtn, confirmBtn),
	)

	dialog.ShowCustom("Transaction Preview", "Close", content, g.window)
}

func (g *MainGUI) broadcastTransaction(txMetadata *wallet.TxMetadata, netAmount int64) {
	// Serialize transaction to hex
	var txHex string
	if txMetadata.Tx != nil {
		// TODO: Serialize the wire.MsgTx to hex
		// For now, show placeholder
		txHex = "placeholder_hex"
	} else {
		dialog.ShowError(fmt.Errorf("no transaction to broadcast"), g.window)
		return
	}

	// Broadcast transaction
	err := g.manager.BroadcastTransaction(txHex, g.manager.GetNetwork())
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to broadcast transaction: %v", err), g.window)
		return
	}

	// Add to history
	// TODO: Get actual transaction ID from the transaction
	// For now, create a placeholder
	var txid [32]byte
	g.manager.AddToHistory(txid, int(netAmount), 0) // Block height 0 for unconfirmed

	// Show success message
	dialog.ShowInformation("Success", "Transaction broadcast successfully!", g.window)

	// TODO: Clear form
}

func (g *MainGUI) sendTransaction(recipient, amountStr, feeRateStr string) {
	// This would be called after preview and confirmation
	// For now, just show a message
	dialog.ShowInformation("Send", "Send functionality will be implemented after transaction preview", g.window)
}
