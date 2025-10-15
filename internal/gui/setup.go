package gui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/setavenger/blindbit-desktop/internal/configs"
	"github.com/setavenger/blindbit-desktop/internal/controller"
	"github.com/setavenger/blindbit-desktop/internal/storage"
	"github.com/setavenger/blindbit-lib/logging"
	"github.com/setavenger/blindbit-lib/types"
	"github.com/setavenger/blindbit-lib/wallet"
)

type SetupWizard struct {
	app      fyne.App
	window   fyne.Window
	dataDir  string
	onFinish func(*controller.Manager)
}

func NewSetupWizard(app fyne.App, window fyne.Window, dataDir string, onFinish func(*controller.Manager)) *SetupWizard {
	return &SetupWizard{
		app:      app,
		window:   window,
		dataDir:  dataDir,
		onFinish: onFinish,
	}
}

func (s *SetupWizard) Show() {
	s.showWelcomeDialog()
}

func (s *SetupWizard) showWelcomeDialog() {
	welcomeText := widget.NewRichTextFromMarkdown(`
# Welcome to BlindBit Desktop

This wizard will help you set up your Bitcoin Silent Payment wallet.

You can either:
- **Create a new wallet** with a generated seed phrase
- **Import an existing wallet** using your seed phrase

**Important**: Make sure to write down your seed phrase and keep it safe!
`)

	createBtn := widget.NewButton("Create New Wallet", func() {
		s.showWalletTypeDialog()
	})

	importBtn := widget.NewButton("Import Existing Wallet", func() {
		s.showImportDialog()
	})

	content := container.NewVBox(
		welcomeText,
		widget.NewSeparator(),
		createBtn,
		importBtn,
	)

	// Set the window content directly instead of using a dialog
	s.window.SetContent(content)
	s.window.Resize(fyne.NewSize(500, 400))
}

func (s *SetupWizard) showWalletTypeDialog() {
	// Generate a new mnemonic
	mnemonic, err := wallet.GenerateMnemonic()
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to generate mnemonic: %v", err), s.window)
		return
	}

	// Display the generated mnemonic for the user to save
	mnemonicText := widget.NewRichTextFromMarkdown(fmt.Sprintf(`
# Your New Wallet Seed Phrase

**IMPORTANT**: Write down these 24 words in the exact order shown below. 
This is your only way to recover your wallet if you lose access to this device.

**Store them safely and never share them with anyone!**

## Your Seed Phrase:

%s

**Please confirm that you have written down these words before continuing.**
`, mnemonic))

	mnemonicText.Wrapping = fyne.TextWrapWord

	confirmBtn := widget.NewButton("I Have Written Down My Seed Phrase", func() {
		s.showMnemonicConfirmation(mnemonic)
	})

	backBtn := widget.NewButton("Back", func() {
		s.showWelcomeDialog()
	})

	content := container.NewVBox(
		mnemonicText,
		widget.NewSeparator(),
		container.NewHBox(backBtn, confirmBtn),
	)

	// Set the window content directly
	s.window.SetContent(content)
	s.window.Resize(fyne.NewSize(600, 500))
}

func (s *SetupWizard) showMnemonicConfirmation(mnemonic string) {
	// Ask user to confirm they've written down the mnemonic
	confirmText := widget.NewRichTextFromMarkdown(`
# Confirm Your Seed Phrase

To ensure you have written down your seed phrase correctly, please enter it below.

**This is your last chance to verify you have saved your seed phrase!**
`)

	mnemonicEntry := widget.NewMultiLineEntry()
	mnemonicEntry.SetPlaceHolder("Enter your 24-word seed phrase here to confirm...")
	mnemonicEntry.Resize(fyne.NewSize(450, 120))

	confirmBtn := widget.NewButton("Confirm Seed Phrase", func() {
		enteredMnemonic := strings.TrimSpace(mnemonicEntry.Text)
		if enteredMnemonic == mnemonic {
			// Mnemonic matches, proceed to configuration
			s.createWalletFromMnemonic(mnemonic)
		} else {
			err := errors.New("seed phrase does not match. Please try again")
			dialog.ShowError(err, s.window)
		}
	})

	backBtn := widget.NewButton("Back", func() {
		s.showWalletTypeDialog()
	})

	content := container.NewVBox(
		confirmText,
		widget.NewSeparator(),
		mnemonicEntry,
		widget.NewSeparator(),
		container.NewHBox(backBtn, confirmBtn),
	)

	// Set the window content directly
	s.window.SetContent(content)
	s.window.Resize(fyne.NewSize(600, 500))
}

func (s *SetupWizard) showImportDialog() {
	mnemonicEntry := widget.NewMultiLineEntry()
	mnemonicEntry.SetPlaceHolder("Enter your 24-word seed phrase here...")
	mnemonicEntry.Resize(fyne.NewSize(450, 120))

	importBtn := widget.NewButton("Import Wallet", func() {
		mnemonic := strings.TrimSpace(mnemonicEntry.Text)
		if s.validateMnemonic(mnemonic) {
			s.createWalletFromMnemonic(mnemonic)
		} else {
			dialog.ShowError(fmt.Errorf("invalid mnemonic"), s.window)
		}
	})

	backBtn := widget.NewButton("Back", func() {
		s.showWelcomeDialog()
	})

	content := container.NewVBox(
		widget.NewLabel("Enter your seed phrase:"),
		mnemonicEntry,
		widget.NewSeparator(),
		container.NewHBox(backBtn, importBtn),
	)

	// Set the window content directly
	s.window.SetContent(content)
	s.window.Resize(fyne.NewSize(600, 500))
}

func (s *SetupWizard) createWalletFromMnemonic(mnemonic string) {
	// Create wallet from mnemonic (default to signet for now, will be configurable)
	walletInstance, err := wallet.NewFromMnemonic(mnemonic, types.NetworkSignet)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to create wallet from mnemonic: %v", err), s.window)
		return
	}

	// Create manager with the wallet
	manager := controller.NewManager()
	manager.Wallet = walletInstance
	manager.DataDir = s.dataDir

	// Show configuration dialog
	s.showConfigurationDialog(manager)
}

func (s *SetupWizard) showConfigurationDialog(manager *controller.Manager) {
	// Network selection
	networkLabel := widget.NewLabel("Network:")
	networkRadio := widget.NewRadioGroup([]string{"mainnet", "testnet", "signet"}, func(value string) {
		switch value {
		case "mainnet":
			manager.Wallet.Network = types.NetworkMainnet
		case "testnet":
			manager.Wallet.Network = types.NetworkTestnet
		case "signet":
			manager.Wallet.Network = types.NetworkSignet
		}
	})
	networkRadio.SetSelected("signet") // Default to signet

	// Birth height
	birthHeightEntry := widget.NewEntry()
	birthHeightEntry.SetPlaceHolder("Leave empty for current height")
	birthHeightLabel := widget.NewLabel("Birth Height (optional):")

	// Oracle address
	oracleEntry := widget.NewEntry()
	oracleEntry.SetText(configs.DefaultOracleAddress)
	oracleLabel := widget.NewLabel("Oracle Address:")

	// Dust limit
	dustLimitEntry := widget.NewEntry()
	dustLimitEntry.SetText(fmt.Sprintf("%d", configs.DefaultMinimumAmount))
	dustLimitLabel := widget.NewLabel("Dust Limit (satoshis):")

	// Min change amount
	minChangeEntry := widget.NewEntry()
	minChangeEntry.SetText(fmt.Sprintf("%d", configs.DefaultMinimumAmount))
	minChangeLabel := widget.NewLabel("Min Change Amount (satoshis):")

	saveBtn := widget.NewButton("Save & Continue", func() {
		// Parse birth height
		if birthHeightEntry.Text != "" {
			if height, err := strconv.Atoi(birthHeightEntry.Text); err == nil {
				manager.BirthHeight = height
			}
		}

		// Parse dust limit
		if dustLimit, err := strconv.Atoi(dustLimitEntry.Text); err == nil {
			manager.DustLimit = dustLimit
		}

		// Parse min change amount
		if minChange, err := strconv.ParseUint(minChangeEntry.Text, 10, 64); err == nil {
			manager.MinChangeAmount = minChange
		}

		// Set oracle address
		manager.OracleAddress = oracleEntry.Text

		// Save the manager
		if err := storage.SavePlain(s.dataDir, manager); err != nil {
			logging.L.Err(err).
				Str("datadir", s.dataDir).
				Msg("failed to save wallet")
			dialog.ShowError(fmt.Errorf("failed to save wallet: %v", err), s.window)
			return
		}

		// Call the finish callback - the main GUI will replace the window content
		s.onFinish(manager)
	})

	backBtn := widget.NewButton("Back", func() {
		s.showWelcomeDialog()
	})

	content := container.NewVBox(
		widget.NewLabel("Configure your wallet:"),
		widget.NewSeparator(),
		networkLabel,
		networkRadio,
		widget.NewSeparator(),
		birthHeightLabel,
		birthHeightEntry,
		oracleLabel,
		oracleEntry,
		dustLimitLabel,
		dustLimitEntry,
		minChangeLabel,
		minChangeEntry,
		widget.NewSeparator(),
		container.NewHBox(backBtn, saveBtn),
	)

	// Set the window content directly
	s.window.SetContent(content)
	s.window.Resize(fyne.NewSize(600, 500))
}

func (s *SetupWizard) validateMnemonic(mnemonic string) bool {
	words := strings.Fields(mnemonic)
	// Basic validation - should be 12 or 24 words
	return len(words) == 12 || len(words) == 24
}
