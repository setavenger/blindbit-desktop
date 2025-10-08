package gui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/setavenger/blindbit-lib/logging"
)

// SendTabFields holds references to the send tab form fields
type SendTabFields struct {
	AddressEntry *widget.Entry
	AmountEntry  *widget.Entry
	FeeRateEntry *widget.Entry
}

// createSendTab creates the send tab
func (g *MainGUI) createSendTab() *fyne.Container {
	addressEntry := widget.NewEntry()
	addressEntry.SetPlaceHolder("Enter recipient address")

	amountEntry := widget.NewEntry()
	amountEntry.SetPlaceHolder("Enter amount in sats")

	feeRateEntry := widget.NewEntry()
	feeRateEntry.SetPlaceHolder("Fee rate (sats/vB)")

	// Store references to the fields for later reset
	g.sendTabFields = &SendTabFields{
		AddressEntry: addressEntry,
		AmountEntry:  amountEntry,
		FeeRateEntry: feeRateEntry,
	}

	sendButton := widget.NewButtonWithIcon("Send", theme.MailSendIcon(), func() {
		g.sendTransaction(addressEntry.Text, amountEntry.Text, feeRateEntry.Text)
	})

	form := container.NewVBox(
		widget.NewLabel("Send Bitcoin"),
		addressEntry,
		amountEntry,
		feeRateEntry,
		sendButton,
	)

	return container.NewPadded(form)
}

// sendTransaction sends a transaction
func (g *MainGUI) sendTransaction(address, amountStr, feeRateStr string) {
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		dialog.ShowError(fmt.Errorf("invalid amount: %v", err), g.window)
		return
	}

	feeRate, err := strconv.ParseInt(feeRateStr, 10, 64)
	if err != nil {
		dialog.ShowError(fmt.Errorf("invalid fee rate: %v", err), g.window)
		return
	}

	result, err := g.walletManager.SendTransaction(address, amount, feeRate)
	if err != nil {
		logging.L.Err(err).Msg("failed to send transaction")
		dialog.ShowError(fmt.Errorf("failed to build transaction: %v", err), g.window)
		return
	}

	// Show transaction details in a new window
	g.ShowTransactionDetails(result)
}

// ResetSendTabFields clears all the send tab form fields
// This method implements the SendTabResetter interface
func (g *MainGUI) ResetSendTabFields() {
	if g.sendTabFields != nil {
		g.sendTabFields.AddressEntry.SetText("")
		g.sendTabFields.AmountEntry.SetText("")
		g.sendTabFields.FeeRateEntry.SetText("")
	}
}
