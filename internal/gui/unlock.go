package gui

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/setavenger/blindbit-desktop/internal/storage"
)

// UnlockScreen is shown at startup when a wallet file already exists.
// It prompts for the wallet password (if the wallet is encrypted) and calls
// onUnlock once the credential is verified. For plaintext wallets onUnlock is
// called immediately with a nil password.
type UnlockScreen struct {
	app      fyne.App
	window   fyne.Window
	dataDir  string
	onUnlock func([]byte)
}

func NewUnlockScreen(
	app fyne.App,
	window fyne.Window,
	dataDir string,
	onUnlock func([]byte),
) *UnlockScreen {
	return &UnlockScreen{
		app:      app,
		window:   window,
		dataDir:  dataDir,
		onUnlock: onUnlock,
	}
}

func (u *UnlockScreen) Show() {
	// Detect whether the wallet on disk is encrypted.
	raw, _ := os.ReadFile(filepath.Join(u.dataDir, "wallet.dat"))
	isEncrypted := storage.IsEncrypted(raw)

	titleText := widget.NewRichTextFromMarkdown("# Welcome Back")
	titleText.Wrapping = fyne.TextWrapWord

	if !isEncrypted {
		// Plaintext wallet — no password needed.
		descLabel := widget.NewLabel(
			"Your wallet is not password-protected. Click Unlock to continue.",
		)
		descLabel.Wrapping = fyne.TextWrapWord

		unlockBtn := widget.NewButton("Unlock", func() {
			u.onUnlock(nil)
		})
		unlockBtn.Importance = widget.HighImportance

		content := container.NewVBox(
			titleText,
			widget.NewSeparator(),
			descLabel,
			widget.NewSeparator(),
			unlockBtn,
		)

		u.window.SetContent(container.NewPadded(content))
		u.window.Resize(fyne.NewSize(500, 350))
		return
	}

	// Encrypted wallet — prompt for password.
	descLabel := widget.NewLabel(
		"Enter your wallet password to unlock BlindBit Desktop.",
	)
	descLabel.Wrapping = fyne.TextWrapWord

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Password")

	errorLabel := widget.NewLabel("")
	errorLabel.Alignment = fyne.TextAlignCenter
	errorLabel.TextStyle = fyne.TextStyle{Bold: true}
	errorLabel.Hide()

	unlockBtn := widget.NewButton("Unlock", nil)
	unlockBtn.Importance = widget.HighImportance

	attempt := func() {
		pwd := []byte(passwordEntry.Text)

		errorLabel.Hide()
		unlockBtn.Disable()
		passwordEntry.Disable()

		go func() {
			_, err := storage.Load(u.dataDir, pwd)
			if err != nil {
				// Wrong password — show error and let user retry.
				errorLabel.SetText("Incorrect password, please try again")
				errorLabel.Show()
				passwordEntry.Enable()
				passwordEntry.SetText("")
				unlockBtn.Enable()
				return
			}

			// Correct password — pass it to the caller.
			u.onUnlock(pwd)
		}()
	}

	unlockBtn.OnTapped = func() { attempt() }
	passwordEntry.OnSubmitted = func(_ string) { attempt() }

	content := container.NewVBox(
		titleText,
		widget.NewSeparator(),
		descLabel,
		widget.NewSeparator(),
		widget.NewLabel("Password:"),
		passwordEntry,
		errorLabel,
		unlockBtn,
	)

	u.window.SetContent(container.NewPadded(content))
	u.window.Resize(fyne.NewSize(500, 350))
}
