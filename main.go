package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"filippo.io/age/armor"
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
			text = "⏎ : Encrypt | ⇥ : Switch to Users"
		} else {
			text = "⇥ : Switch to Users"
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
		searchInput = tview.NewInputField().SetLabel("Search: ")
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
		usersPanel.SetBorder(true).SetTitle("Users")

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

	if err := app.Run(); err != nil {
		panic(err)
	}
}
