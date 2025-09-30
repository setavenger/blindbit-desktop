package gui

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"sort"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/setavenger/blindbit-desktop/internal/manager"
	"github.com/setavenger/blindbit-lib/logging"
)

// MainGUI represents the main GUI application
type MainGUI struct {
	window          fyne.Window
	walletManager   *manager.Manager
	content         *fyne.Container
	addressLabel    *widget.Label
	balanceLabel    *widget.Label
	scanStatusLabel *widget.Label
	scanHeightLabel *widget.Label
	chainTipLabel   *widget.Label
	oracleInfoLabel *widget.Label
	utxoList        *widget.Table
	utxoData        []UTXODisplay
	updateTicker    *time.Ticker
	cachedAddress   string       // Cache the address to avoid constant regeneration
	cachedChainTip  uint64       // Cache chain tip to avoid blocking network calls
	trayManager     *TrayManager // System tray manager
}

// UTXODisplay represents a UTXO for display in the GUI
type UTXODisplay struct {
	TxID      string
	Amount    string
	State     string
	Timestamp string
	Vout      uint32 // Added Vout for enhanced display
	Label     string // Add label information for display
}

func (g *MainGUI) UtxoCount(states ...string) int {
	if len(states) == 0 {
		// todo: unspent should be global defined constant
		states = []string{"unspent"}
	}
	var count int
	for _, v := range g.utxoData {
		if len(states) > 0 {
			if slices.Contains(states, v.State) {
				count++
			}
		} else {
			count++
		}
	}
	return count
}

// NewMainGUI creates a new main GUI instance
func NewMainGUI(
	app fyne.App, window fyne.Window, walletManager *manager.Manager,
) *MainGUI {
	gui := &MainGUI{
		window:        window,
		walletManager: walletManager,
		utxoData:      []UTXODisplay{},
	}
	gui.createContent()

	// Initialize system tray manager
	gui.trayManager = NewTrayManager(app, window)

	// Initialize cached chain tip in background with delay to avoid blocking startup
	go func() {
		// Wait a bit for the UI to fully initialize before making network calls
		time.Sleep(1 * time.Second)
		// Only try to get chain tip if scanner is ready
		if gui.walletManager.IsScannerReady() {
			if chainTip, err := gui.walletManager.GetChainTip(); err == nil {
				gui.cachedChainTip = chainTip
			}
		}
	}()

	// per default start scanning
	if err := gui.walletManager.StartScanning(); err != nil {
		logging.L.Fatal().Err(err).Msg("failed to start scanner")
	}

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
	// Note: Tray manager cleanup is handled automatically by the systray package
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
	g.chainTipLabel = widget.NewLabel("Chain Tip: Loading...")
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
		g.chainTipLabel,
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
	if !g.walletManager.IsScannerReady() {
		g.scanStatusLabel.SetText("Scan Status: Initializing...")
		g.scanStatusLabel.TextStyle = fyne.TextStyle{}
	} else if g.walletManager.IsScanning() {
		g.scanStatusLabel.SetText("Scan Status: Scanning")
		g.scanStatusLabel.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		g.scanStatusLabel.SetText("Scan Status: Not Scanning")
		g.scanStatusLabel.TextStyle = fyne.TextStyle{}
	}

	// Update scan height immediately (this is fast)
	scanHeight := g.walletManager.GetScanHeight()
	g.scanHeightLabel.SetText(fmt.Sprintf("Scan Height: %d", scanHeight))

	// Update chain tip and sync status using cached value to avoid blocking
	if g.cachedChainTip > 0 {
		scanHeight := g.walletManager.GetScanHeight()
		syncPercentage := float64(scanHeight) / float64(g.cachedChainTip) * 100.0
		if syncPercentage > 100.0 {
			syncPercentage = 100.0
		}

		g.chainTipLabel.SetText(
			fmt.Sprintf("Chain Tip: %d (%.1f%% synced)", g.cachedChainTip, syncPercentage),
		)

		// Color code the sync status
		if syncPercentage >= 100.0 {
			g.chainTipLabel.TextStyle = fyne.TextStyle{Bold: true}
		} else if syncPercentage >= 90.0 {
			g.chainTipLabel.TextStyle = fyne.TextStyle{Bold: true}
		} else {
			g.chainTipLabel.TextStyle = fyne.TextStyle{}
		}
	} else {
		g.chainTipLabel.SetText("Chain Tip: Loading...")
		g.chainTipLabel.TextStyle = fyne.TextStyle{}
	}

	oracleUrl := g.walletManager.GetOracleURL()
	// Update oracle info
	g.oracleInfoLabel.SetText(fmt.Sprintf("Oracle: %s", oracleUrl))

	// Refresh the labels
	g.addressLabel.Refresh()
	g.balanceLabel.Refresh()
	g.scanStatusLabel.Refresh()
	g.scanHeightLabel.Refresh()
	g.chainTipLabel.Refresh()
	g.oracleInfoLabel.Refresh()
}

// updateScanHeightOnly quickly updates just the scan height without network calls
func (g *MainGUI) updateScanHeightOnly() {
	scanHeight := g.walletManager.GetScanHeight()
	g.scanHeightLabel.SetText(fmt.Sprintf("Scan Height: %d", scanHeight))
	g.scanHeightLabel.Refresh()
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
		// Format label information
		labelText := ""
		if utxo.Label != nil {
			labelText = fmt.Sprintf("M=%d", utxo.Label.M)
		}

		// Format timestamp
		timestamp := time.Unix(int64(utxo.Timestamp), 0).Format("2006-01-02 15:04:05")

		g.utxoData = append(g.utxoData, UTXODisplay{
			TxID:      hex.EncodeToString(utxo.Txid[:]),
			Amount:    fmt.Sprintf("%d sats", utxo.Amount),
			State:     utxo.State.String(),
			Timestamp: timestamp,
			Vout:      utxo.Vout,
			Label:     labelText,
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

	// Start background chain tip updater (less frequent to avoid blocking)
	go g.startChainTipUpdater()

	go func() {
		utxoRefreshCounter := 0
		for range g.updateTicker.C {
			// Only update if we have a main screen
			if g.content != nil && g.scanStatusLabel != nil {
				g.updateWalletInfo()

				// Refresh UTXOs less frequently to reduce blocking (every 10 seconds instead of 2)
				utxoRefreshCounter++
				if utxoRefreshCounter >= 5 { // 5 * 2 seconds = 10 seconds
					go g.refreshUTXOs() // Run in background goroutine
					utxoRefreshCounter = 0
				}
			}
		}
	}()
}

// startChainTipUpdater runs a background goroutine to update chain tip cache
func (g *MainGUI) startChainTipUpdater() {
	ticker := time.NewTicker(30 * time.Second) // Update chain tip every 30 seconds
	defer ticker.Stop()

	for range ticker.C {
		// Update chain tip in background without blocking UI
		go func() {
			// Only try to get chain tip if scanner is ready
			if g.walletManager.IsScannerReady() {
				if chainTip, err := g.walletManager.GetChainTip(); err == nil {
					g.cachedChainTip = chainTip
				}
			}
		}()
	}
}

// stopPeriodicUpdates stops periodic updates
func (g *MainGUI) stopPeriodicUpdates() {
	if g.updateTicker != nil {
		g.updateTicker.Stop()
	}
}

// IsWindowVisible returns whether the main window is currently visible
func (g *MainGUI) IsWindowVisible() bool {
	if g.trayManager != nil {
		return g.trayManager.IsVisible()
	}
	return true // Default to visible if no tray manager
}

// UpdateTrayIcon updates the system tray icon
func (g *MainGUI) UpdateTrayIcon(iconBytes []byte) {
	if g.trayManager != nil {
		g.trayManager.UpdateTrayIcon(iconBytes)
	}
}

// UpdateTrayTooltip updates the system tray tooltip
func (g *MainGUI) UpdateTrayTooltip(tooltip string) {
	if g.trayManager != nil {
		g.trayManager.UpdateTrayTooltip(tooltip)
	}
}

// UpdateTrayTitle updates the system tray title
func (g *MainGUI) UpdateTrayTitle(title string) {
	if g.trayManager != nil {
		g.trayManager.UpdateTrayTitle(title)
	}
}

// restartApplication attempts to re-execute the current binary with the same arguments, then exits the current process.
func (g *MainGUI) restartApplication() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	// Preserve original args (excluding the current process name; os.Args[0] is the path)
	args := os.Args[1:]

	cmd := exec.Command(exePath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Start the new process and exit the current one
	if err := cmd.Start(); err != nil {
		return err
	}
	// Close the current app window and exit
	if g.window != nil {
		g.window.Close()
	}
	os.Exit(0)
	return nil
}

// ShowTransactionDetails shows transaction details in a new window
func (g *MainGUI) ShowTransactionDetails(result *manager.TransactionResult) {
	// Create a new window for transaction details
	txWindow := fyne.CurrentApp().NewWindow("Transaction Details")
	txWindow.Resize(fyne.NewSize(800, 600))
	txWindow.SetFixedSize(false) // Make it resizable

	// Create transaction details tab
	txTab := NewTransactionDetailsTab(txWindow, g.walletManager, result)
	content := txTab.CreateTransactionDetailsView()

	txWindow.SetContent(content)
	txWindow.Show()
}
