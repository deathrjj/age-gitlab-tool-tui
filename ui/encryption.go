package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/deathrjj/age-gitlab-tool-tui/encryption"
	"github.com/deathrjj/age-gitlab-tool-tui/gitlab"
	"github.com/deathrjj/age-gitlab-tool-tui/models"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// EncryptionUI handles the encryption UI flow
type EncryptionUI struct {
	App           *tview.Application
	AllUsers      []models.User
	FilteredUsers []models.User
	SelectedUsers models.UserSelectionMap
	GitlabClient  *gitlab.Client
}

// NewEncryptionUI creates a new encryption UI instance
func NewEncryptionUI(app *tview.Application) *EncryptionUI {
	return &EncryptionUI{
		App:           app,
		SelectedUsers: make(models.UserSelectionMap),
	}
}

// StartEncryptionUI initializes the encryption UI
func (ui *EncryptionUI) StartEncryptionUI() {
	loadingText := tview.NewTextView().
		SetText("Initializing...").
		SetTextAlign(tview.AlignCenter)
	ui.App.SetRoot(loadingText, true)

	// Check if GitLab URL is set first
	baseURL := os.Getenv("GITLAB_URL")
	if baseURL == "" {
		// First prompt for GitLab URL
		ui.PromptForGitLabURL()
		return
	}
	
	// Then check if GitLab token is set
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		// Then prompt for GitLab token
		ui.PromptForGitLabToken()
		return
	}
	
	// If we have all variables, continue with loading users
	ui.LoadUsers()
}

// PromptForGitLabURL shows a form to enter GitLab URL
func (ui *EncryptionUI) PromptForGitLabURL() {
	form := tview.NewForm()
	
	var gitlabURL string
	
	form.AddInputField("GitLab URL:", "", 50, nil, func(text string) {
		gitlabURL = text
	})
	
	form.AddButton("Continue", func() {
		if gitlabURL == "" {
			errorModal := CreateErrorModal(ui.App, "GitLab URL is required", form)
			ui.App.SetRoot(errorModal, true)
			return
		}
		
		// Set environment variable
		os.Setenv("GITLAB_URL", gitlabURL)
		
		// Now check for token
		token := os.Getenv("GITLAB_TOKEN")
		if token == "" {
			ui.PromptForGitLabToken()
		} else {
			ui.LoadUsers()
		}
	})
	
	form.AddButton("Cancel", func() {
		ui.App.Stop()
	})
	
	form.SetBorder(true).SetTitle("GitLab URL").SetTitleAlign(tview.AlignCenter)
	ui.App.SetRoot(form, true)
	ui.App.SetFocus(form)
}

// PromptForGitLabToken shows a form to enter GitLab token
func (ui *EncryptionUI) PromptForGitLabToken() {
	form := tview.NewForm()
	
	var gitlabToken string
	
	form.AddPasswordField("GitLab Token:", "", 50, '*', func(text string) {
		gitlabToken = text
	})
	
	form.AddButton("Continue", func() {
		if gitlabToken == "" {
			errorModal := CreateErrorModal(ui.App, "GitLab token is required", form)
			ui.App.SetRoot(errorModal, true)
			return
		}
		
		// Set environment variable
		os.Setenv("GITLAB_TOKEN", gitlabToken)
		
		// Continue with loading users
		ui.LoadUsers()
	})
	
	form.AddButton("Cancel", func() {
		ui.App.Stop()
	})
	
	form.SetBorder(true).SetTitle("GitLab Token").SetTitleAlign(tview.AlignCenter)
	ui.App.SetRoot(form, true)
	ui.App.SetFocus(form)
}

// LoadUsers fetches users from GitLab and initializes the encryption UI
func (ui *EncryptionUI) LoadUsers() {
	loadingText := tview.NewTextView().
		SetText("Loading users...").
		SetTextAlign(tview.AlignCenter)
	ui.App.SetRoot(loadingText, true)

	// Initialize GitLab client
	var err error
	ui.GitlabClient, err = gitlab.NewClient()
	if err != nil {
		ui.App.QueueUpdateDraw(func() {
			modal := tview.NewModal().
				SetText(fmt.Sprintf("Error initializing GitLab client: %v", err)).
				AddButtons([]string{"Quit"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) { ui.App.Stop() })
			ui.App.SetRoot(modal, false)
		})
		return
	}

	// Declare UI components
	var searchInput *tview.InputField
	var dataInput *tview.TextArea
	var layout tview.Primitive
	var bottomBar *tview.TextView
	var encryptButton *tview.Button

	go func() {
		users, err := ui.GitlabClient.FetchUsers()
		if err != nil {
			ui.App.QueueUpdateDraw(func() {
				modal := tview.NewModal().
					SetText(fmt.Sprintf("Error fetching users: %v", err)).
					AddButtons([]string{"Quit"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) { ui.App.Stop() })
				ui.App.SetRoot(modal, false)
			})
			return
		}
		ui.AllUsers = users
		ui.FilteredUsers = users

		// Create user list.
		userList := tview.NewList()
		UpdateUserList(userList, ui.FilteredUsers, ui.SelectedUsers)
		userList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
			if index < 0 || index >= len(ui.FilteredUsers) {
				return
			}
			u := ui.FilteredUsers[index]
			if ui.SelectedUsers[u.ID] {
				delete(ui.SelectedUsers, u.ID)
			} else {
				ui.SelectedUsers[u.ID] = true
			}
			UpdateUserList(userList, ui.FilteredUsers, ui.SelectedUsers)
			userList.SetCurrentItem(index)
			UpdateBottomBar(ui.App, bottomBar, searchInput, userList, dataInput, encryptButton)
		})
		userList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyTab {
				if len(ui.SelectedUsers) > 0 {
					ui.App.SetFocus(dataInput)
					UpdateBottomBar(ui.App, bottomBar, searchInput, userList, dataInput, encryptButton)
				}
				return nil
			}
			switch event.Key() {
			case tcell.KeyUp, tcell.KeyDown, tcell.KeyEnter:
				return event
			case tcell.KeyRune:
				ui.App.SetFocus(searchInput)
				current := searchInput.GetText()
				searchInput.SetText(current + string(event.Rune()))
				UpdateBottomBar(ui.App, bottomBar, searchInput, userList, dataInput, encryptButton)
				return nil
			default:
				ui.App.SetFocus(searchInput)
				UpdateBottomBar(ui.App, bottomBar, searchInput, userList, dataInput, encryptButton)
				return event
			}
		})

		// Create search input.
		searchInput = tview.NewInputField()
		searchInput.SetChangedFunc(func(text string) {
			ui.FilteredUsers = nil
			searchText := strings.ToLower(text)
			
			for _, user := range ui.AllUsers {
				// Always search using original usernames, not censored ones
				if text == "" || ContainsCaseInsensitive(user.Username, searchText) {
					ui.FilteredUsers = append(ui.FilteredUsers, user)
				}
			}
			
			UpdateUserList(userList, ui.FilteredUsers, ui.SelectedUsers)
			UpdateBottomBar(ui.App, bottomBar, searchInput, userList, dataInput, encryptButton)
		})
		searchInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyUp, tcell.KeyDown, tcell.KeyEnter:
				ui.App.SetFocus(userList)
				UpdateBottomBar(ui.App, bottomBar, searchInput, userList, dataInput, encryptButton)
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
						encrypted, err := encryption.EncryptData(dataInput.GetText(), ui.SelectedUsers, ui.GitlabClient)
						if err != nil {
							ui.App.QueueUpdateDraw(func() {
								modal := tview.NewModal().
									SetText(fmt.Sprintf("Encryption failed: %v", err)).
									AddButtons([]string{"OK"}).
									SetDoneFunc(func(buttonIndex int, buttonLabel string) {
										ui.App.SetRoot(layout, true).SetFocus(dataInput)
										UpdateBottomBar(ui.App, bottomBar, searchInput, userList, dataInput, encryptButton)
									})
								ui.App.SetRoot(modal, false)
							})
							return
						}
						ui.App.Stop()
						fmt.Println(encrypted)
					}()
				}
			})

		dataInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyTab {
				ui.App.SetFocus(encryptButton)
				UpdateBottomBar(ui.App, bottomBar, searchInput, userList, dataInput, encryptButton)
				return nil
			}
			UpdateBottomBar(ui.App, bottomBar, searchInput, userList, dataInput, encryptButton)
			return event
		})

		encryptButton.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyTab {
				ui.App.SetFocus(userList)
				UpdateBottomBar(ui.App, bottomBar, searchInput, userList, dataInput, encryptButton)
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

		ui.App.QueueUpdateDraw(func() {
			bottomBar.SetText("↑/↓: move highlight | Enter: toggle selection")
			ui.App.SetRoot(layout, true).SetFocus(userList)
			UpdateBottomBar(ui.App, bottomBar, searchInput, userList, dataInput, encryptButton)
		})
	}()
} 
