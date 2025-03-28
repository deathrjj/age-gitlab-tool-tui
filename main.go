package main

import (
	"fmt"
	"os"

	"github.com/deathrjj/age-gitlab-tool-tui/encryption"
	"github.com/deathrjj/age-gitlab-tool-tui/ui"
	"github.com/rivo/tview"
)

func main() {
	// Check if demo mode is enabled
	if os.Getenv("AGE_TOOL_DEMO_MODE") != "" {
		fmt.Println("Running in demo mode - usernames will be partially censored")
	}
	
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
