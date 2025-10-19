package gui

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/setavenger/blindbit-desktop/internal/storage"
	"github.com/setavenger/blindbit-lib/logging"
)

func (g *MainGUI) createScanningTab() fyne.CanvasObject {
	// Main title
	titleLabel := widget.NewLabel("Scanning Overview")
	titleLabel.TextStyle.Bold = true

	// Scanning status section
	scanStatusTitle := widget.NewLabel("Scanning Status")
	scanStatusTitle.TextStyle.Bold = true

	// Current scan height
	currentScanLabel := widget.NewLabel("Current Scan Height: N/A")

	// Chain tip height
	chainTipLabel := widget.NewLabel("Chain Tip: N/A")

	// Scanning status
	scanStatusLabel := widget.NewLabel("Status: Not scanning")
	scanStatusLabel.TextStyle.Bold = true

	// Rescan options
	rescanTitle := widget.NewLabel("Rescan Options")
	rescanTitle.TextStyle.Bold = true

	// Rescan from height input
	rescanHeightEntry := widget.NewEntry()
	rescanHeightEntry.SetPlaceHolder("Enter height to rescan from (leave empty for birth height)")
	rescanHeightLabel := widget.NewLabel("Rescan from height:")

	// Control buttons
	rescanBtn := widget.NewButton("Rescan", func() {
		heightStr := rescanHeightEntry.Text
		var height int
		var err error

		if heightStr != "" {
			height, err = strconv.Atoi(heightStr)
			if err != nil {
				dialog.ShowError(fmt.Errorf("invalid height: %v", err), g.window)
				return
			}
		} else {
			height = g.manager.BirthHeight
		}

		g.startRescanning(height)
	})

	refreshBtn := widget.NewButton("Refresh Status", func() {
		g.refreshScanStatus(currentScanLabel, chainTipLabel, scanStatusLabel)
	})

	// Progress bar
	progressBar := widget.NewProgressBar()
	progressBar.Hide()

	// Update initial values
	g.refreshScanStatus(currentScanLabel, chainTipLabel, scanStatusLabel)

	// Start periodic refresh of chain tip
	go g.startPeriodicRefresh(chainTipLabel, currentScanLabel, scanStatusLabel)

	// Start real-time progress updates from scanner
	go g.startRealTimeProgressUpdates(currentScanLabel)

	// Start stream end detection for final updates
	go g.startStreamEndDetection(currentScanLabel)

	// Layout sections
	scanStatusSection := container.NewVBox(
		scanStatusTitle,
		currentScanLabel,
		chainTipLabel,
		scanStatusLabel,
	)

	rescanSection := container.NewVBox(
		rescanTitle,
		rescanHeightLabel,
		rescanHeightEntry,
		container.NewHBox(rescanBtn, refreshBtn),
	)

	// Main content
	content := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
		scanStatusSection,
		widget.NewSeparator(),
		rescanSection,
		progressBar,
	)

	return content
}

func (g *MainGUI) startRescanning(fromHeight int) {
	// Scanner should already be initialized in main.go
	if g.manager.Scanner == nil {
		dialog.ShowError(fmt.Errorf("scanner not initialized"), g.window)
		return
	}

	g.performScan(uint32(fromHeight), "Rescanning", fmt.Sprintf("Rescanning started from height %d to current tip", fromHeight))
}

// performScan is the unified scanning function that handles rescanning operations
func (g *MainGUI) performScan(startHeight uint32, operationName, dialogMessage string) {
	// Start scanning from specified height to current tip
	go func() {
		// Get current height
		currentHeight, err := g.manager.GetCurrentHeight()
		if err != nil {
			logging.L.Err(err).Msg("failed to get current height")
			return
		}

		logging.L.Info().
			Uint32("start_height", startHeight).
			Uint32("current_height", currentHeight).
			Str("operation", operationName).
			Msg("starting rescan operation")

		// Start rescanning - channel handling is done by the manager
		// err = g.manager.Scanner.Scan(context.Background(), startHeight, currentHeight)
		err = g.manager.Scanner.Scan(context.Background(), startHeight, currentHeight)
		if err != nil {
			logging.L.Err(err).Msg("rescanning failed")
		} else {
			// Update wallet's LastScanHeight to the final height
			g.manager.Wallet.LastScanHeight = uint64(currentHeight)

			// Send final update to GUI to ensure it shows the completed scan height
			if g.manager.GUIScanProgressChan != nil {
				select {
				case g.manager.GUIScanProgressChan <- currentHeight:
					logging.L.Debug().Uint32("final_height", currentHeight).Msg("sent final scan update to GUI")
				default:
					logging.L.Debug().Msg("GUI progress channel full, skipping final update")
				}
			}

			// Signal that scanning stream has ended
			g.manager.SignalStreamEnd()

			// Save wallet after rescan completion
			if err := storage.SavePlain(g.manager.DataDir, g.manager); err != nil {
				logging.L.Err(err).Msg("failed to save wallet after rescan")
			} else {
				logging.L.Info().Msg("wallet saved after rescan completion")
			}
		}

		logging.L.Info().
			Uint32("start_height", startHeight).
			Uint32("current_height", currentHeight).
			Str("operation", operationName).
			Msg("rescanning finished")
	}()

	dialog.ShowInformation(operationName, dialogMessage, g.window)
}

func (g *MainGUI) refreshScanStatus(currentScanLabel, chainTipLabel, scanStatusLabel *widget.Label) {
	// Update current scan height from wallet - always show the value
	currentScanLabel.SetText(fmt.Sprintf("Current Scan Height: %d", g.manager.Wallet.LastScanHeight))

	// Update chain tip from oracle
	if currentHeight, err := g.manager.GetCurrentHeight(); err == nil {
		chainTipLabel.SetText(fmt.Sprintf("Chain Tip: %d", currentHeight))
	} else {
		chainTipLabel.SetText("Chain Tip: Unable to fetch")
		logging.L.Err(err).Msg("failed to get current height from oracle")
	}

	// Update scanning status
	if g.manager.OracleClient != nil {
		if g.manager.Scanner != nil {
			scanStatusLabel.SetText("Status: Scanner ready")
		} else {
			scanStatusLabel.SetText("Status: Oracle connected, scanner not initialized")
		}
	} else {
		scanStatusLabel.SetText("Status: Oracle not connected")
	}
}

// startPeriodicRefresh starts a goroutine that periodically refreshes chain tip and scan status
func (g *MainGUI) startPeriodicRefresh(chainTipLabel, currentScanLabel, scanStatusLabel *widget.Label) {
	ticker := time.NewTicker(10 * time.Second) // Refresh every 10 seconds for better responsiveness
	defer ticker.Stop()

	for range ticker.C {
		// Update chain tip
		if currentHeight, err := g.manager.GetCurrentHeight(); err == nil {
			chainTipLabel.SetText(fmt.Sprintf("Chain Tip: %d", currentHeight))
		} else {
			chainTipLabel.SetText("Chain Tip: Unable to fetch")
			logging.L.Err(err).Msg("periodic refresh failed to get current height")
		}

		// Update current scan height - always show the value
		currentScanLabel.SetText(fmt.Sprintf("Current Scan Height: %d", g.manager.Wallet.LastScanHeight))

		// Update scanning status
		if g.manager.OracleClient != nil {
			if g.manager.Scanner != nil {
				scanStatusLabel.SetText("Status: Scanner ready")
			} else {
				scanStatusLabel.SetText("Status: Oracle connected, scanner not initialized")
			}
		} else {
			scanStatusLabel.SetText("Status: Oracle not connected")
		}

	}
}

// startRealTimeProgressUpdates listens to the scanner's progress channel and updates the GUI in real-time
func (g *MainGUI) startRealTimeProgressUpdates(currentScanLabel *widget.Label) {
	if g.manager.GUIScanProgressChan == nil {
		logging.L.Warn().Msg("GUI progress channel not initialized")
		logging.L.Info().Msg("Initilising GUI progress channel")
		g.manager.GUIScanProgressChan = make(chan uint32)
		return
	}
	if g.manager.StreamEndChan == nil {
		logging.L.Warn().Msg("stream end channel not initialized")
		logging.L.Info().Msg("Initialising stream end channel")
		g.manager.StreamEndChan = make(chan bool)
		return
	}

	logging.L.Info().Msg("starting real-time progress updates for scanning view")

	for {
		select {
		case height := <-g.manager.GUIScanProgressChan:
			currentScanLabel.SetText(fmt.Sprintf(
				"Current Scan Height: %d",
				g.manager.Wallet.LastScanHeight,
			))
			logging.L.Debug().
				Uint32("height", height).
				Msg("GUI updated with real-time scan progress")
		case <-g.manager.StreamEndChan:
			currentScanLabel.SetText(fmt.Sprintf(
				"Current Scan Height: %d",
				g.manager.Wallet.LastScanHeight,
			))
			logging.L.Info().Msg("stream ended, GUI updated with final scan height")
			return
		case <-time.After(10 * time.Second):
			currentScanLabel.SetText(fmt.Sprintf(
				"Current Scan Height: %d",
				g.manager.Wallet.LastScanHeight,
			))
			logging.L.Debug().Msg("GUI updated with real-time scan progress")
		}
	}
}

// startStreamEndDetection listens for stream end signals and ensures final updates are shown
func (g *MainGUI) startStreamEndDetection(currentScanLabel *widget.Label) {
	// todo: remove this function
	// redundant with startrealtime processing doing everything

	// if g.manager.StreamEndChan == nil {
	// 	logging.L.Warn().Msg("stream end channel not initialized")
	// 	logging.L.Info().Msg("Initialising stream end channel")
	// 	g.manager.StreamEndChan = make(chan bool)
	// 	return
	// }
	//
	// logging.L.Info().Msg("starting stream end detection for scanning view")
	//
	// for range g.manager.StreamEndChan {
	// 	// Force update the GUI with the current scan height when stream ends
	// 	currentScanLabel.SetText(
	// 		fmt.Sprintf("Current Scan Height: %d", g.manager.Wallet.LastScanHeight),
	// 	)
	// 	logging.L.Info().
	// 		Uint64("final_height", g.manager.Wallet.LastScanHeight).
	// 		Msg("stream ended, GUI updated with final scan height")
	// }
	//
	// logging.L.Info().Msg("stream end detection stopped")
}
