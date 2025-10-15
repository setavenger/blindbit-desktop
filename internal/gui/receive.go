package gui

import (
	"bytes"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/skip2/go-qrcode"
)

func (g *MainGUI) createReceiveTab() fyne.CanvasObject {
	// Get the Silent Payment address
	address := g.manager.GetSilentPaymentAddress()

	// Main title
	titleLabel := widget.NewLabel("Receiving Bitcoin")
	titleLabel.TextStyle.Bold = true

	// Introduction text
	introText := widget.NewLabel("This is your Silent Payment address. You can share this address with anyone who wants to send you Bitcoin.")
	introText.Wrapping = fyne.TextWrapWord

	// Address section
	addressTitle := widget.NewLabel("Your Silent Payment Address:")
	addressTitle.TextStyle.Bold = true

	// Address display - use a label instead of entry for better readability
	addressLabel := widget.NewLabel(address)
	addressLabel.TextStyle.Monospace = true
	addressLabel.Wrapping = fyne.TextWrapWord // Allow wrapping for long addresses
	addressLabel.Alignment = fyne.TextAlignLeading

	// Notification label for copy feedback
	notificationLabel := widget.NewLabel("")
	notificationLabel.Alignment = fyne.TextAlignCenter
	notificationLabel.TextStyle.Bold = true
	notificationLabel.Hide() // Initially hidden

	// Copy button - full width
	copyBtn := widget.NewButton("Copy to Clipboard", func() {
		g.copyToClipboard(address, notificationLabel)
	})

	// QR Code section
	qrTitle := widget.NewLabel("QR Code")
	qrTitle.TextStyle.Bold = true

	// Generate QR code
	qrImage := g.generateQRCode(address)

	addressSection := container.NewVBox(
		addressTitle,
		addressLabel,
		copyBtn,
		notificationLabel,
	)

	qrContainer := container.NewVBox(
		qrTitle,
		qrImage,
	)

	// Main content with proper spacing
	content := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
		introText,
		widget.NewSeparator(),
		addressSection,
		widget.NewSeparator(),
		qrContainer,
	)

	return content
}

func (g *MainGUI) copyToClipboard(text string, notificationLabel *widget.Label) {
	// Copy to clipboard
	g.window.Clipboard().SetContent(text)

	// Show temporary notification in the UI
	notificationLabel.SetText("✓ Copied to clipboard!")
	notificationLabel.Show()

	// Hide after 2 seconds
	go func() {
		time.Sleep(2 * time.Second)
		notificationLabel.SetText("")
		notificationLabel.Hide()
	}()
}

func (g *MainGUI) generateQRCode(address string) fyne.CanvasObject {
	// Generate QR code
	qr, err := qrcode.New(address, qrcode.Medium)
	if err != nil {
		errorLabel := widget.NewLabel("Failed to generate QR code")
		errorLabel.Alignment = fyne.TextAlignCenter
		return errorLabel
	}

	// Convert to PNG bytes
	var buf bytes.Buffer
	err = qr.Write(256, &buf)
	if err != nil {
		errorLabel := widget.NewLabel("Failed to encode QR code")
		errorLabel.Alignment = fyne.TextAlignCenter
		return errorLabel
	}

	// Create Fyne image resource
	imageResource := fyne.NewStaticResource("qr.png", buf.Bytes())
	imageCanvas := canvas.NewImageFromResource(imageResource)
	imageCanvas.FillMode = canvas.ImageFillOriginal
	imageCanvas.SetMinSize(fyne.NewSize(256, 256))

	return imageCanvas
}
