package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/deathrjj/age-gitlab-tool-tui/encryption"
	"github.com/rivo/tview"
)

// DecryptionUI handles the decryption UI flow
type DecryptionUI struct {
	App           *tview.Application
	EncryptedText string
}

// NewDecryptionUI creates a new decryption UI instance
func NewDecryptionUI(app *tview.Application, encryptedText string) *DecryptionUI {
	return &DecryptionUI{
		App:           app,
		EncryptedText: encryptedText,
	}
}

// PromptForDecryption shows a prompt asking if the user wants to decrypt the age file
func (ui *DecryptionUI) PromptForDecryption() {
	modal := tview.NewModal().
		SetText("Age file detected in clipboard. Would you like to decrypt it?").
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Yes" {
				// Check if private key path is set
				privateKeyPath := os.Getenv("AGE_PRIVATE_KEY_PATH")
				if privateKeyPath == "" {
					// Prompt for private key path
					ui.PromptForPrivateKeyPath()
					return
				}

				// Try to decrypt without passphrase first
				decrypted, err := encryption.DecryptAgeFile(ui.EncryptedText, privateKeyPath, "")
				if err != nil {
					if strings.Contains(err.Error(), "please provide passphrase") {
						// Key is passphrase protected, prompt for passphrase
						ui.PromptForPassphrase(privateKeyPath)
						return
					}
					
					// Other error
					errorModal := tview.NewModal().
						SetText(fmt.Sprintf("Error decrypting: %v", err)).
						AddButtons([]string{"OK"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							ui.App.Stop()
						})
					ui.App.SetRoot(errorModal, true)
					return
				}

				// Show decrypted message and exit
				ui.App.Stop()
				fmt.Println("Decrypted message:")
				fmt.Println(decrypted)
			} else {
				// Continue with normal app flow
				encryptionUI := NewEncryptionUI(ui.App)
				encryptionUI.StartEncryptionUI()
			}
		})

	ui.App.SetRoot(modal, true)
}

// PromptForPrivateKeyPath shows a form to enter the AGE_PRIVATE_KEY_PATH
func (ui *DecryptionUI) PromptForPrivateKeyPath() {
	form := tview.NewForm()
	
	var keyPath string
	
	form.AddInputField("Path to SSH private key:", "", 50, nil, func(text string) {
		keyPath = text
	})
	
	form.AddButton("Continue", func() {
		if keyPath == "" {
			errorModal := CreateErrorModal(ui.App, "Please enter a valid file path", form)
			ui.App.SetRoot(errorModal, true)
			return
		}
		
		// Check if file exists
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			errorModal := CreateErrorModal(ui.App, "File does not exist. Please enter a valid path.", form)
			ui.App.SetRoot(errorModal, true)
			return
		}
		
		// Set environment variable
		os.Setenv("AGE_PRIVATE_KEY_PATH", keyPath)
		
		// Continue with decryption
		privateKeyPath := keyPath
		decrypted, err := encryption.DecryptAgeFile(ui.EncryptedText, privateKeyPath, "")
		if err != nil {
			if strings.Contains(err.Error(), "please provide passphrase") {
				// Key is passphrase protected, prompt for passphrase
				ui.PromptForPassphrase(privateKeyPath)
				return
			}
			
			// Other error
			errorModal := tview.NewModal().
				SetText(fmt.Sprintf("Error decrypting: %v", err)).
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					ui.App.Stop()
				})
			ui.App.SetRoot(errorModal, true)
			return
		}

		// Show decrypted message and exit
		ui.App.Stop()
		fmt.Println("Decrypted message:")
		fmt.Println(decrypted)
	})
	
	form.AddButton("Cancel", func() {
		encryptionUI := NewEncryptionUI(ui.App)
		encryptionUI.StartEncryptionUI()
	})
	
	form.SetBorder(true).SetTitle("SSH Private Key Path").SetTitleAlign(tview.AlignCenter)
	ui.App.SetRoot(form, true)
	ui.App.SetFocus(form)
}

// PromptForPassphrase shows a prompt for entering the SSH key passphrase
func (ui *DecryptionUI) PromptForPassphrase(privateKeyPath string) {
	form := tview.NewForm()
	
	var passphrase string
	
	form.AddPasswordField("Passphrase:", "", 50, '*', func(text string) {
		passphrase = text
	})
	
	form.AddButton("Decrypt", func() {
		// Try to decrypt with provided passphrase
		decrypted, err := encryption.DecryptAgeFile(ui.EncryptedText, privateKeyPath, passphrase)
		if err != nil {
			errorModal := CreateErrorModal(ui.App, fmt.Sprintf("Error decrypting: %v", err), form)
			ui.App.SetRoot(errorModal, true)
			return
		}
		
		// Show decrypted message and exit
		ui.App.Stop()
		fmt.Println("Decrypted message:")
		fmt.Println(decrypted)
	})
	
	form.AddButton("Cancel", func() {
		encryptionUI := NewEncryptionUI(ui.App)
		encryptionUI.StartEncryptionUI()
	})
	
	form.SetBorder(true).SetTitle("SSH Key Passphrase").SetTitleAlign(tview.AlignCenter)
	ui.App.SetRoot(form, true)
	ui.App.SetFocus(form)
} 
