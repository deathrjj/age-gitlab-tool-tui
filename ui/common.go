package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/deathrjj/age-gitlab-tool-tui/models"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// UpdateUserList refreshes the list with filtered users.
// It prefixes usernames with "- " if unselected or "✓ " if selected.
func UpdateUserList(list *tview.List, users []models.User, selectedUsers models.UserSelectionMap) {
	list.Clear()
	isDemoMode := os.Getenv("AGE_TOOL_DEMO_MODE") != ""
	
	for _, user := range users {
		prefix := "- "
		color := "white"
		if selectedUsers[user.ID] {
			prefix = "✓ "
			color = "green"
		}
		
		username := user.Username
		if isDemoMode && len(username) > 2 {
			// In demo mode, censor all characters after the first two
			username = username[:2] + strings.Repeat("*", len(username)-2)
		}
		
		list.AddItem(fmt.Sprintf("[%s]%s", color, prefix+username), "", 0, nil)
	}
}

// UpdateBottomBar updates the bottom bar text based on current focus.
func UpdateBottomBar(app *tview.Application, bottomBar *tview.TextView, searchInput *tview.InputField, 
	userList *tview.List, dataInput *tview.TextArea, encryptButton *tview.Button) {
	
	focused := app.GetFocus()
	var text string
	
	if focused == userList || focused == searchInput {
		text = "↑/↓: Move Highlight | ⏎ : Toggle Selection"
		if searchInput != nil && userList != nil && dataInput != nil {
			// Check if there's at least one selected user
			currentSelectedCount := 0
			for i := 0; i < userList.GetItemCount(); i++ {
				mainText, _ := userList.GetItemText(i)
				if strings.Contains(mainText, "✓") {
					currentSelectedCount++
				}
			}
			
			if currentSelectedCount > 0 {
				text += " | ⇥ : Switch to Data"
			}
		}
	} else if focused == dataInput {
		text = "⇥ : Switch to Encrypt Button"
	} else if focused == encryptButton {
		if dataInput != nil && dataInput.GetText() != "" {
			text = "⏎ : Encrypt | ⇥ : Switch to Recipients"
		} else {
			text = "⇥ : Switch to Recipients"
		}
	}
	
	bottomBar.SetText(text)
}

// ContainsCaseInsensitive returns true if s contains substr (case-insensitive).
func ContainsCaseInsensitive(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// CreateErrorModal creates a modal to display error messages
func CreateErrorModal(app *tview.Application, message string, returnFocus tview.Primitive) *tview.Modal {
	return tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.SetRoot(returnFocus, true)
		})
}

// SetupKeyboardNavigation sets up common keyboard shortcuts for navigation between components
func SetupKeyboardNavigation(app *tview.Application, components ...tview.Primitive) {
	for i, component := range components {
		index := i // Capture loop variable
		
		// Each component can have its own input capture method
		// We need to handle each type specifically
		switch c := component.(type) {
		case *tview.InputField:
			originalHandler := c.GetInputCapture()
			c.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if event.Key() == tcell.KeyTab {
					nextIndex := (index + 1) % len(components)
					app.SetFocus(components[nextIndex])
					return nil
				}
				if originalHandler != nil {
					return originalHandler(event)
				}
				return event
			})
		case *tview.TextArea:
			originalHandler := c.GetInputCapture()
			c.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if event.Key() == tcell.KeyTab {
					nextIndex := (index + 1) % len(components)
					app.SetFocus(components[nextIndex])
					return nil
				}
				if originalHandler != nil {
					return originalHandler(event)
				}
				return event
			})
		case *tview.List:
			originalHandler := c.GetInputCapture()
			c.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if event.Key() == tcell.KeyTab {
					nextIndex := (index + 1) % len(components)
					app.SetFocus(components[nextIndex])
					return nil
				}
				if originalHandler != nil {
					return originalHandler(event)
				}
				return event
			})
		case *tview.Button:
			originalHandler := c.GetInputCapture()
			c.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if event.Key() == tcell.KeyTab {
					nextIndex := (index + 1) % len(components)
					app.SetFocus(components[nextIndex])
					return nil
				}
				if originalHandler != nil {
					return originalHandler(event)
				}
				return event
			})
		}
	}
} 
