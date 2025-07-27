package gui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createSendTab creates the send tab
func (g *MainGUI) createSendTab() *fyne.Container {
	addressEntry := widget.NewEntry()
	addressEntry.SetPlaceHolder("Enter recipient address")

	amountEntry := widget.NewEntry()
	amountEntry.SetPlaceHolder("Enter amount in sats")

	feeRateEntry := widget.NewEntry()
	feeRateEntry.SetPlaceHolder("Fee rate (sats/vB)")

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

	txid, err := g.walletManager.SendTransaction(address, amount, feeRate)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to send transaction: %v", err), g.window)
		return
	}

	dialog.ShowInformation("Success", "Transaction sent successfully\nTxID: "+txid, g.window)
}
