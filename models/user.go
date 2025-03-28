package models

// User represents a GitLab user.
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

// UserSelectionMap stores which users are selected (map of user ID to selection status)
type UserSelectionMap map[int]bool 
