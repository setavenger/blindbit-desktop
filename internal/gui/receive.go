package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (g *MainGUI) createReceiveTab() fyne.CanvasObject {
	// Get the Silent Payment address
	address := g.manager.GetSilentPaymentAddress()

	// Address display
	addressLabel := widget.NewLabel("Your Silent Payment Address:")
	addressEntry := widget.NewEntry()
	addressEntry.SetText(address)
	addressEntry.Disable() // Read-only

	// Copy button
	copyBtn := widget.NewButton("Copy to Clipboard", func() {
		g.copyToClipboard(address)
	})

	// QR code placeholder
	qrLabel := widget.NewLabel("QR Code")
	qrLabel.Alignment = fyne.TextAlignCenter

	// Instructions
	instructionsText := widget.NewRichTextFromMarkdown(`
# Receiving Bitcoin

This is your Silent Payment address. You can share this address with anyone who wants to send you Bitcoin.

**Important Notes:**
- This address can be reused safely
- All payments to this address will be private
- Make sure to keep your seed phrase safe
`)

	// Main content
	content := container.NewVBox(
		instructionsText,
		widget.NewSeparator(),
		addressLabel,
		addressEntry,
		copyBtn,
		widget.NewSeparator(),
		qrLabel,
		widget.NewLabel("QR Code will be implemented"),
	)

	return content
}

func (g *MainGUI) copyToClipboard(text string) {
	// Copy to clipboard
	g.window.Clipboard().SetContent(text)

	// Show confirmation
	dialog.ShowInformation("Copied", "Address copied to clipboard!", g.window)
}
