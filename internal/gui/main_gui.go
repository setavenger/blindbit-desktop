package gui

import (
	"fmt"
	"sort"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/setavenger/blindbit-desktop/internal/wallet"
)

// MainGUI represents the main GUI application
type MainGUI struct {
	window          fyne.Window
	walletManager   *wallet.Manager
	content         *fyne.Container
	addressLabel    *widget.Label
	balanceLabel    *widget.Label
	scanStatusLabel *widget.Label
	scanHeightLabel *widget.Label
	oracleInfoLabel *widget.Label
	utxoList        *widget.Table
	utxoData        []UTXODisplay
	updateTicker    *time.Ticker
	cachedAddress   string // Cache the address to avoid constant regeneration
}

// UTXODisplay represents a UTXO for display in the GUI
type UTXODisplay struct {
	TxID      string
	Amount    string
	State     string
	Timestamp string
	Vout      uint32 // Added Vout for enhanced display
}

// NewMainGUI creates a new main GUI instance
func NewMainGUI(window fyne.Window, walletManager *wallet.Manager) *MainGUI {
	gui := &MainGUI{
		window:        window,
		walletManager: walletManager,
		utxoData:      []UTXODisplay{},
	}
	gui.createContent()

	// Start periodic updates
	gui.startPeriodicUpdates()

	return gui
}

// GetContent returns the main content container
func (g *MainGUI) GetContent() *fyne.Container {
	return g.content
}

// Cleanup cleans up resources when the GUI is destroyed
func (g *MainGUI) Cleanup() {
	g.stopPeriodicUpdates()
}

// createContent creates the main GUI content
func (g *MainGUI) createContent() {
	// Check if wallet exists
	if !g.walletManager.HasWallet() {
		g.content = g.createWelcomeScreen()
		return
	}

	// Load existing wallet
	if err := g.walletManager.LoadWallet(); err != nil {
		dialog.ShowError(fmt.Errorf("failed to load wallet: %v", err), g.window)
		g.content = g.createWelcomeScreen()
		return
	}

	g.content = g.createMainScreen()
}

// createMainScreen creates the main application screen
func (g *MainGUI) createMainScreen() *fyne.Container {
	// Header with wallet info
	g.addressLabel = widget.NewLabel("Address: Loading...")
	g.balanceLabel = widget.NewLabel("Balance: Loading...")
	g.scanStatusLabel = widget.NewLabel("Scan Status: Not Scanning")
	g.scanHeightLabel = widget.NewLabel("Scan Height: 0")
	g.oracleInfoLabel = widget.NewLabel("Oracle: Loading...")

	// Update wallet info
	g.updateWalletInfo()

	// Create tabs for different sections
	overviewTab := g.createOverviewTab()
	utxoOverviewTab := g.createUTXOOverviewTab()
	sendTab := g.createSendTab()
	receiveTab := g.createReceiveTab()
	settingsTab := g.createSettingsTab()

	tabs := container.NewAppTabs(
		container.NewTabItem("Overview", overviewTab),
		container.NewTabItem("UTXOs", utxoOverviewTab),
		container.NewTabItem("Send", sendTab),
		container.NewTabItem("Receive", receiveTab),
		container.NewTabItem("Settings", settingsTab),
	)

	// Header container with better layout
	header := container.NewVBox(
		g.addressLabel,
		g.balanceLabel,
		g.scanStatusLabel,
		g.scanHeightLabel,
		widget.NewSeparator(),
	)

	mainContainer := container.NewBorder(header, nil, nil, nil, tabs)
	return mainContainer
}

// updateWalletInfo updates the wallet information display
func (g *MainGUI) updateWalletInfo() {
	// Only get address if we don't have it cached
	if g.cachedAddress == "" {
		address, err := g.walletManager.GetAddress()
		if err != nil {
			g.cachedAddress = "Error loading address"
		} else {
			g.cachedAddress = address
		}
	}

	g.addressLabel.SetText("Address: " + g.cachedAddress)

	balance := g.walletManager.GetBalance()
	g.balanceLabel.SetText(fmt.Sprintf("Balance: %d sats", balance))

	// Update scan status
	if g.walletManager.IsScanning() {
		g.scanStatusLabel.SetText("Scan Status: Scanning")
		g.scanStatusLabel.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		g.scanStatusLabel.SetText("Scan Status: Not Scanning")
		g.scanStatusLabel.TextStyle = fyne.TextStyle{}
	}

	// Update scan height
	scanHeight := g.walletManager.GetScanHeight()
	g.scanHeightLabel.SetText(fmt.Sprintf("Scan Height: %d", scanHeight))

	oracleUrl := g.walletManager.GetOracleURL()
	// Update oracle info
	g.oracleInfoLabel.SetText(fmt.Sprintf("Oracle: %s", oracleUrl))

	// Refresh the labels
	g.addressLabel.Refresh()
	g.balanceLabel.Refresh()
	g.scanStatusLabel.Refresh()
	g.scanHeightLabel.Refresh()
	g.oracleInfoLabel.Refresh()
}

// updateOracleURLDisplay updates the Oracle URL display in the main view
func (g *MainGUI) updateOracleURLDisplay() {
	if g.oracleInfoLabel != nil {
		g.oracleInfoLabel.SetText(fmt.Sprintf("Oracle: %s", g.walletManager.GetOracleURL()))
		g.oracleInfoLabel.Refresh()
	}
}

// refreshUTXOs refreshes the UTXO list
func (g *MainGUI) refreshUTXOs() {
	utxos, err := g.walletManager.GetUTXOs()
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to get UTXOs: %v", err), g.window)
		return
	}

	g.utxoData = []UTXODisplay{}
	for _, utxo := range utxos {
		g.utxoData = append(g.utxoData, UTXODisplay{
			TxID:      utxo.TxID,
			Amount:    fmt.Sprintf("%d sats", utxo.Amount),
			State:     string(utxo.State),
			Timestamp: fmt.Sprintf("%d", utxo.Timestamp),
			Vout:      utxo.Vout, // Add Vout to UTXODisplay
		})
	}

	// Sort UTXOs by timestamp (descending - newest first), then by txid, then by vout
	g.sortUTXOs()

	// Refresh the UTXO list if it exists
	if g.utxoList != nil {
		g.utxoList.Refresh()
	}
}

// sortUTXOs sorts the UTXOs by timestamp (descending), then by txid, then by vout
func (g *MainGUI) sortUTXOs() {
	// Sort in place using sort.Slice
	sort.Slice(g.utxoData, func(i, j int) bool {
		// First sort by timestamp (descending - newest first)
		if g.utxoData[i].Timestamp != g.utxoData[j].Timestamp {
			return g.utxoData[i].Timestamp > g.utxoData[j].Timestamp
		}

		// If timestamps are equal, sort by txid
		if g.utxoData[i].TxID != g.utxoData[j].TxID {
			return g.utxoData[i].TxID < g.utxoData[j].TxID
		}

		// If txids are equal, sort by vout
		return g.utxoData[i].Vout < g.utxoData[j].Vout
	})
}

// startPeriodicUpdates starts periodic updates of wallet info
func (g *MainGUI) startPeriodicUpdates() {
	g.updateTicker = time.NewTicker(2 * time.Second) // Update every 2 seconds

	go func() {
		for range g.updateTicker.C {
			// Only update if we have a main screen
			if g.content != nil && g.scanStatusLabel != nil {
				g.updateWalletInfo()
				// Also refresh UTXOs periodically to keep the overview current
				g.refreshUTXOs()
			}
		}
	}()
}

// stopPeriodicUpdates stops periodic updates
func (g *MainGUI) stopPeriodicUpdates() {
	if g.updateTicker != nil {
		g.updateTicker.Stop()
	}
}
