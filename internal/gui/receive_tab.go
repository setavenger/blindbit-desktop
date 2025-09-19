package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/skip2/go-qrcode"
)

// createReceiveTab creates the receive tab
func (g *MainGUI) createReceiveTab() *fyne.Container {
	address, err := g.walletManager.GetAddress()
	if err != nil {
		address = "Error loading address"
	}

	addressLabel := widget.NewLabel(address)
	addressLabel.Wrapping = fyne.TextWrapWord

	copyButton := widget.NewButtonWithIcon("Copy Address", theme.ContentCopyIcon(), func() {
		g.window.Clipboard().SetContent(address)
		dialog.ShowInformation("Copied", "Address copied to clipboard", g.window)
	})

	// Create QR code
	var qrImage *canvas.Image
	if err == nil && address != "Error loading address" {
		qrImage = g.createQRCode(address)
	}

	// Create the main container
	var content *fyne.Container
	if qrImage != nil {
		content = container.NewVBox(
			widget.NewLabel("Receive Bitcoin"),
			widget.NewLabel("Scan QR code or copy address:"),
			container.NewCenter(qrImage),
			addressLabel,
			copyButton,
		)
	} else {
		content = container.NewVBox(
			widget.NewLabel("Receive Bitcoin"),
			addressLabel,
			copyButton,
		)
	}

	return content
}

// createQRCode generates a QR code image for the given address
func (g *MainGUI) createQRCode(address string) *canvas.Image {
	// Generate QR code as PNG bytes
	pngBytes, err := qrcode.Encode(address, qrcode.Medium, 256)
	if err != nil {
		return nil
	}

	// Create Fyne image resource
	imgResource := fyne.NewStaticResource("qr-code", pngBytes)

	// Create canvas image
	canvasImg := canvas.NewImageFromResource(imgResource)
	canvasImg.FillMode = canvas.ImageFillOriginal
	canvasImg.SetMinSize(fyne.NewSize(200, 200))

	return canvasImg
}
