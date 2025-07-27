package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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

	return container.NewVBox(
		widget.NewLabel("Receive Bitcoin"),
		addressLabel,
		copyButton,
	)
}
