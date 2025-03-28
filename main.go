package main

import (
	"github.com/deathrjj/age-gitlab-tool-tui/encryption"
	"github.com/deathrjj/age-gitlab-tool-tui/ui"
	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()

	// Check clipboard for age encrypted file
	if encryptedText, found := encryption.CheckClipboardForAgeFile(); found {
		decryptionUI := ui.NewDecryptionUI(app, encryptedText)
		decryptionUI.PromptForDecryption()
	} else {
		encryptionUI := ui.NewEncryptionUI(app)
		encryptionUI.StartEncryptionUI()
	}

	if err := app.Run(); err != nil {
		panic(err)
	}
}
