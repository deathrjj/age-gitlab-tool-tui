package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"filippo.io/age/armor"
	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// User represents a GitLab user.
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

var (
	allUsers      []User
	filteredUsers []User
	// Map of user ID to selection status.
	selectedUsers = make(map[int]bool)
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// checkClipboardForAgeFile checks if the clipboard contains an age encrypted file
func checkClipboardForAgeFile() (string, bool) {
	text, err := clipboard.ReadAll()
	if err != nil {
		return "", false
	}

	// Check for age encrypted file markers
	if strings.Contains(text, "-----BEGIN AGE ENCRYPTED FILE-----") &&
		strings.Contains(text, "-----END AGE ENCRYPTED FILE-----") {
		return text, true
	}
	return "", false
}

// decryptAgeFile decrypts an age encrypted file using a private key file
func decryptAgeFile(encryptedText, privateKeyPath, passphrase string) (string, error) {
	// Read the private key file
	keyData, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key file: %w", err)
	}

	// To handle passphrase-protected SSH keys, we need to use golang.org/x/crypto/ssh
	// For simplicity, we'll use a workaround - we'll check if ParseIdentity fails with
	// a passphrase error, and if it does, we'll return a specific error to trigger the passphrase prompt
	
	var sshIdentity age.Identity
	if passphrase == "" {
		// Try without passphrase
		sshIdentity, err = agessh.ParseIdentity(keyData)
		if err != nil && strings.Contains(err.Error(), "passphrase") {
			return "", fmt.Errorf("ssh key is passphrase protected, please provide passphrase")
		}
	} else {
		// Use age.ParseIdentity as a temporary solution
		// In a production app, this would be replaced with proper SSH key decryption
		// using the passphrase and then parsing the decrypted key
		sshKeyFile, err := os.CreateTemp("", "ssh-key-*")
		if err != nil {
			return "", fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(sshKeyFile.Name())
		defer sshKeyFile.Close()
		
		// Write the key data to the temp file
		_, err = sshKeyFile.Write(keyData)
		if err != nil {
			return "", fmt.Errorf("failed to write temp key file: %w", err)
		}
		sshKeyFile.Close()
		
		// Run the command to decrypt the key
		decryptCmd := exec.Command("bash", "-c", 
			fmt.Sprintf("ssh-keygen -p -P '%s' -N '' -f '%s'", passphrase, sshKeyFile.Name()))
		if err := decryptCmd.Run(); err != nil {
			return "", fmt.Errorf("failed to decrypt SSH key with passphrase: %w", err)
		}
		
		// Read the decrypted key
		decryptedKeyData, err := ioutil.ReadFile(sshKeyFile.Name())
		if err != nil {
			return "", fmt.Errorf("failed to read decrypted key: %w", err)
		}
		
		// Parse the decrypted key
		sshIdentity, err = agessh.ParseIdentity(decryptedKeyData)
		if err != nil {
			return "", fmt.Errorf("failed to parse decrypted SSH key: %w", err)
		}
	}
	
	if err != nil {
		return "", fmt.Errorf("failed to parse SSH key: %w", err)
	}

	// Decrypt the message using the SSH identity
	r, err := age.Decrypt(armor.NewReader(strings.NewReader(encryptedText)), sshIdentity)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	// Read the decrypted content
	decrypted, err := ioutil.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read decrypted content: %w", err)
	}

	return string(decrypted), nil
}

// promptForDecryption shows a prompt asking if the user wants to decrypt the age file
func promptForDecryption(app *tview.Application, encryptedText string) {
	modal := tview.NewModal().
		SetText("Age file detected in clipboard. Would you like to decrypt it?").
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Yes" {
				// Check if private key path is set
				privateKeyPath := os.Getenv("AGE_PRIVATE_KEY_PATH")
				if privateKeyPath == "" {
					// Prompt for private key path
					promptForPrivateKeyPath(app, encryptedText)
					return
				}

				// Try to decrypt without passphrase first
				decrypted, err := decryptAgeFile(encryptedText, privateKeyPath, "")
				if err != nil {
					if strings.Contains(err.Error(), "please provide passphrase") {
						// Key is passphrase protected, prompt for passphrase
						promptForPassphrase(app, encryptedText, privateKeyPath)
						return
					}
					
					// Other error
					errorModal := tview.NewModal().
						SetText(fmt.Sprintf("Error decrypting: %v", err)).
						AddButtons([]string{"OK"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							app.Stop()
						})
					app.SetRoot(errorModal, true)
					return
				}

				// Show decrypted message and exit
				app.Stop()
				fmt.Println("Decrypted message:")
				fmt.Println(decrypted)
			} else {
				// Continue with normal app flow
				startEncryptionUI(app)
			}
		})

	app.SetRoot(modal, true)
}

// promptForPrivateKeyPath shows a form to enter the AGE_PRIVATE_KEY_PATH
func promptForPrivateKeyPath(app *tview.Application, encryptedText string) {
	form := tview.NewForm()
	
	var keyPath string
	
	form.AddInputField("Path to SSH private key:", "", 50, nil, func(text string) {
		keyPath = text
	})
	
	form.AddButton("Continue", func() {
		if keyPath == "" {
			errorModal := tview.NewModal().
				SetText("Please enter a valid file path").
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					app.SetRoot(form, true)
				})
			app.SetRoot(errorModal, true)
			return
		}
		
		// Check if file exists
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			errorModal := tview.NewModal().
				SetText("File does not exist. Please enter a valid path.").
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					app.SetRoot(form, true)
				})
			app.SetRoot(errorModal, true)
			return
		}
		
		// Set environment variable
		os.Setenv("AGE_PRIVATE_KEY_PATH", keyPath)
		
		// Continue with decryption
		privateKeyPath := keyPath
		decrypted, err := decryptAgeFile(encryptedText, privateKeyPath, "")
		if err != nil {
			if strings.Contains(err.Error(), "please provide passphrase") {
				// Key is passphrase protected, prompt for passphrase
				promptForPassphrase(app, encryptedText, privateKeyPath)
				return
			}
			
			// Other error
			errorModal := tview.NewModal().
				SetText(fmt.Sprintf("Error decrypting: %v", err)).
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					app.Stop()
				})
			app.SetRoot(errorModal, true)
			return
		}

		// Show decrypted message and exit
		app.Stop()
		fmt.Println("Decrypted message:")
		fmt.Println(decrypted)
	})
	
	form.AddButton("Cancel", func() {
		startEncryptionUI(app)
	})
	
	form.SetBorder(true).SetTitle("SSH Private Key Path").SetTitleAlign(tview.AlignCenter)
	app.SetRoot(form, true)
	app.SetFocus(form)
}

// promptForPassphrase shows a prompt for entering the SSH key passphrase
func promptForPassphrase(app *tview.Application, encryptedText, privateKeyPath string) {
	form := tview.NewForm()
	
	var passphrase string
	
	form.AddPasswordField("Passphrase:", "", 50, '*', func(text string) {
		passphrase = text
	})
	
	form.AddButton("Decrypt", func() {
		// Try to decrypt with provided passphrase
		decrypted, err := decryptAgeFile(encryptedText, privateKeyPath, passphrase)
		if err != nil {
			errorModal := tview.NewModal().
				SetText(fmt.Sprintf("Error decrypting: %v", err)).
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					// Go back to passphrase prompt
					app.SetRoot(form, true)
				})
			app.SetRoot(errorModal, true)
			return
		}
		
		// Show decrypted message and exit
		app.Stop()
		fmt.Println("Decrypted message:")
		fmt.Println(decrypted)
	})
	
	form.AddButton("Cancel", func() {
		startEncryptionUI(app)
	})
	
	form.SetBorder(true).SetTitle("SSH Key Passphrase").SetTitleAlign(tview.AlignCenter)
	app.SetRoot(form, true)
	app.SetFocus(form)
}

// startEncryptionUI initializes the encryption UI
func startEncryptionUI(app *tview.Application) {
	loadingText := tview.NewTextView().
		SetText("Initializing...").
		SetTextAlign(tview.AlignCenter)
	app.SetRoot(loadingText, true)

	// Check if GitLab URL is set first
	baseURL := os.Getenv("GITLAB_URL")
	if baseURL == "" {
		// First prompt for GitLab URL
		promptForGitLabURL(app)
		return
	}
	
	// Then check if GitLab token is set
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		// Then prompt for GitLab token
		promptForGitLabToken(app)
		return
	}
	
	// If we have all variables, continue with loading users
	loadUsers(app)
}

// promptForGitLabURL shows a form to enter GitLab URL
func promptForGitLabURL(app *tview.Application) {
	form := tview.NewForm()
	
	var gitlabURL string
	
	form.AddInputField("GitLab URL:", "", 50, nil, func(text string) {
		gitlabURL = text
	})
	
	form.AddButton("Continue", func() {
		if gitlabURL == "" {
			errorModal := tview.NewModal().
				SetText("GitLab URL is required").
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					app.SetRoot(form, true)
				})
			app.SetRoot(errorModal, true)
			return
		}
		
		// Set environment variable
		os.Setenv("GITLAB_URL", gitlabURL)
		
		// Now check for token
		token := os.Getenv("GITLAB_TOKEN")
		if token == "" {
			promptForGitLabToken(app)
		} else {
			loadUsers(app)
		}
	})
	
	form.AddButton("Cancel", func() {
		app.Stop()
	})
	
	form.SetBorder(true).SetTitle("GitLab URL").SetTitleAlign(tview.AlignCenter)
	app.SetRoot(form, true)
	app.SetFocus(form)
}

// promptForGitLabToken shows a form to enter GitLab token
func promptForGitLabToken(app *tview.Application) {
	form := tview.NewForm()
	
	var gitlabToken string
	
	form.AddPasswordField("GitLab Token:", "", 50, '*', func(text string) {
		gitlabToken = text
	})
	
	form.AddButton("Continue", func() {
		if gitlabToken == "" {
			errorModal := tview.NewModal().
				SetText("GitLab token is required").
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					app.SetRoot(form, true)
				})
			app.SetRoot(errorModal, true)
			return
		}
		
		// Set environment variable
		os.Setenv("GITLAB_TOKEN", gitlabToken)
		
		// Continue with loading users
		loadUsers(app)
	})
	
	form.AddButton("Cancel", func() {
		app.Stop()
	})
	
	form.SetBorder(true).SetTitle("GitLab Token").SetTitleAlign(tview.AlignCenter)
	app.SetRoot(form, true)
	app.SetFocus(form)
}

// loadUsers fetches users from GitLab and initializes the encryption UI
func loadUsers(app *tview.Application) {
	loadingText := tview.NewTextView().
		SetText("Loading users...").
		SetTextAlign(tview.AlignCenter)
	app.SetRoot(loadingText, true)

	// Declare variables in outer scope.
	var searchInput *tview.InputField
	var dataInput *tview.TextArea
	var layout tview.Primitive
	var bottomBar *tview.TextView
	var encryptButton *tview.Button

	go func() {
		users, err := fetchUsers()
		if err != nil {
			app.QueueUpdateDraw(func() {
				modal := tview.NewModal().
					SetText(fmt.Sprintf("Error fetching users: %v", err)).
					AddButtons([]string{"Quit"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) { app.Stop() })
				app.SetRoot(modal, false)
			})
			return
		}
		allUsers = users
		filteredUsers = users

		// Create user list.
		userList := tview.NewList()
		updateUserList(userList, filteredUsers)
		userList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
			if index < 0 || index >= len(filteredUsers) {
				return
			}
			u := filteredUsers[index]
			if selectedUsers[u.ID] {
				delete(selectedUsers, u.ID)
			} else {
				selectedUsers[u.ID] = true
			}
			updateUserList(userList, filteredUsers)
			userList.SetCurrentItem(index)
			updateBottomBar(app, bottomBar, searchInput, userList, dataInput, encryptButton)
		})
		userList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyTab {
				if len(selectedUsers) > 0 {
					app.SetFocus(dataInput)
					updateBottomBar(app, bottomBar, searchInput, userList, dataInput, encryptButton)
				}
				return nil
			}
			switch event.Key() {
			case tcell.KeyUp, tcell.KeyDown, tcell.KeyEnter:
				return event
			case tcell.KeyRune:
				app.SetFocus(searchInput)
				current := searchInput.GetText()
				searchInput.SetText(current + string(event.Rune()))
				updateBottomBar(app, bottomBar, searchInput, userList, dataInput, encryptButton)
				return nil
			default:
				app.SetFocus(searchInput)
				updateBottomBar(app, bottomBar, searchInput, userList, dataInput, encryptButton)
				return event
			}
		})

		// Create search input.
		searchInput = tview.NewInputField()
		searchInput.SetChangedFunc(func(text string) {
			filteredUsers = nil
			for _, user := range allUsers {
				if text == "" || containsCaseInsensitive(user.Username, text) {
					filteredUsers = append(filteredUsers, user)
				}
			}
			updateUserList(userList, filteredUsers)
			updateBottomBar(app, bottomBar, searchInput, userList, dataInput, encryptButton)
		})
		searchInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyUp, tcell.KeyDown, tcell.KeyEnter:
				app.SetFocus(userList)
				updateBottomBar(app, bottomBar, searchInput, userList, dataInput, encryptButton)
				return nil
			}
			return event
		})

		// Left panel: search input and user list.
		usersPanel := tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(searchInput, 3, 0, true).
			AddItem(userList, 0, 1, false)
		usersPanel.SetBorder(true).SetTitle("Recipients")

		// Create Data panel as a text area.
		dataInput = tview.NewTextArea().
			SetWrap(true).
			SetWordWrap(true)
			
		// Add encrypt button
		encryptButton = tview.NewButton("Encrypt").
			SetSelectedFunc(func() {
				if dataInput.GetText() != "" {
					go func() {
						encrypted, err := encryptData(dataInput.GetText(), selectedUsers)
						if err != nil {
							app.QueueUpdateDraw(func() {
								modal := tview.NewModal().
									SetText(fmt.Sprintf("Encryption failed: %v", err)).
									AddButtons([]string{"OK"}).
									SetDoneFunc(func(buttonIndex int, buttonLabel string) {
										app.SetRoot(layout, true).SetFocus(dataInput)
										updateBottomBar(app, bottomBar, searchInput, userList, dataInput, encryptButton)
									})
								app.SetRoot(modal, false)
							})
							return
						}
						app.Stop()
						fmt.Println(encrypted)
					}()
				}
			})

		dataInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyTab {
				app.SetFocus(encryptButton)
				updateBottomBar(app, bottomBar, searchInput, userList, dataInput, encryptButton)
				return nil
			}
			updateBottomBar(app, bottomBar, searchInput, userList, dataInput, encryptButton)
			return event
		})

		encryptButton.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyTab {
				app.SetFocus(userList)
				updateBottomBar(app, bottomBar, searchInput, userList, dataInput, encryptButton)
				return nil
			}
			return event
		})

		dataPanel := tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(dataInput, 0, 1, false).
			AddItem(encryptButton, 1, 0, false)
		dataPanel.SetBorder(true).SetTitle("Data")

		// Create Bottom bar.
		bottomBar = tview.NewTextView().
			SetDynamicColors(true).
			SetTextAlign(tview.AlignCenter)
		bottomBar.SetText("↑/↓: move highlight | Enter: toggle selection")

		// Main layout: two columns on top, bottom bar as last row.
		mainFlex := tview.NewFlex().
			AddItem(usersPanel, 0, 1, true).
			AddItem(dataPanel, 0, 1, true)
		layout = tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(mainFlex, 0, 1, true).
			AddItem(bottomBar, 1, 0, false)

		app.QueueUpdateDraw(func() {
			bottomBar.SetText("↑/↓: move highlight | Enter: toggle selection")
			app.SetRoot(layout, true).SetFocus(userList)
			updateBottomBar(app, bottomBar, searchInput, userList, dataInput, encryptButton)
		})
	}()
}

// updateBottomBar updates the bottom bar text based on current focus.
func updateBottomBar(app *tview.Application, bottomBar *tview.TextView, searchInput *tview.InputField, userList *tview.List, dataInput *tview.TextArea, encryptButton *tview.Button) {
	focused := app.GetFocus()
	var text string
	if focused == userList || focused == searchInput {
		text = "↑/↓: Move Highlight | ⏎ : Toggle Selection"
		if len(selectedUsers) > 0 {
			text += " | ⇥ : Switch to Data"
		}
	} else if focused == dataInput {
		text = "⇥ : Switch to Encrypt Button"
	} else if focused == encryptButton {
		if dataInput.GetText() != "" {
			text = "⏎ : Encrypt | ⇥ : Switch to Recipients"
		} else {
			text = "⇥ : Switch to Recipients"
		}
	}
	bottomBar.SetText(text)
}

// fetchUsers retrieves GitLab users page by page.
func fetchUsers() ([]User, error) {
	baseURL := os.Getenv("GITLAB_URL")
	token := os.Getenv("GITLAB_TOKEN")
	if baseURL == "" || token == "" {
		return nil, fmt.Errorf("GITLAB_URL or GITLAB_TOKEN not set")
	}
	var users []User
	perPage := 100
	for page := 1; ; page++ {
		url := fmt.Sprintf("%s/api/v4/users?active=true&humans=true&exclude_external=true&page=%d&per_page=%d", baseURL, page, perPage)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Add("PRIVATE-TOKEN", token)
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		var pageUsers []User
		if err := json.Unmarshal(body, &pageUsers); err != nil {
			return nil, err
		}
		users = append(users, pageUsers...)
		if len(pageUsers) < perPage {
			break
		}
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].Username < users[j].Username
	})
	return users, nil
}

// fetchUserKeys retrieves the keys for a given user ID without any splitting.
func fetchUserKeys(userID int) ([]string, error) {
	baseURL := os.Getenv("GITLAB_URL")
	token := os.Getenv("GITLAB_TOKEN")
	url := fmt.Sprintf("%s/api/v4/users/%d/keys", baseURL, userID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("PRIVATE-TOKEN", token)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	var keysResp []struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(body, &keysResp); err != nil {
		return nil, err
	}
	var keys []string
	for _, k := range keysResp {
		keys = append(keys, k.Key)
	}
	return keys, nil
}

// encryptData encrypts plaintext using age with each selected user's key as a recipient.
func encryptData(plaintext string, selected map[int]bool) (string, error) {
	var recipients []age.Recipient
	for uid := range selected {
		keys, err := fetchUserKeys(uid)
		if err != nil {
			return "", err
		}
		for _, keyStr := range keys {
			// Use agessh.ParseRecipient to parse an SSH key as an age recipient.
			rec, err := agessh.ParseRecipient(keyStr)
			if err != nil {
				return "", fmt.Errorf("failed to parse recipient for user %d: %w", uid, err)
			}
			recipients = append(recipients, rec)
		}
	}
	var buf bytes.Buffer
	armorWriter := armor.NewWriter(&buf)
	w, err := age.Encrypt(armorWriter, recipients...)
	if err != nil {
		return "", err
	}
	if _, err := w.Write([]byte(plaintext)); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	if err := armorWriter.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// containsCaseInsensitive returns true if s contains substr (case-insensitive).
func containsCaseInsensitive(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// updateUserList refreshes the list with filtered users.
// It prefixes usernames with "- " if unselected or "✓ " if selected.
func updateUserList(list *tview.List, users []User) {
	list.Clear()
	for _, user := range users {
		prefix := "- "
		color := "white"
		if selectedUsers[user.ID] {
			prefix = "✓ "
			color = "green"
		}
		list.AddItem(fmt.Sprintf("[%s]%s", color, prefix+user.Username), "", 0, nil)
	}
}

func main() {
	app := tview.NewApplication()

	// Check clipboard for age encrypted file
	if encryptedText, found := checkClipboardForAgeFile(); found {
		promptForDecryption(app, encryptedText)
	} else {
		startEncryptionUI(app)
	}

	if err := app.Run(); err != nil {
		panic(err)
	}
}
