package gitlab

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/deathrjj/age-gitlab-tool-tui/models"
)

// Client handles GitLab API interactions
type Client struct {
	BaseURL string
	Token   string
	client  *http.Client
}

// NewClient creates a new GitLab API client
func NewClient() (*Client, error) {
	baseURL := os.Getenv("GITLAB_URL")
	token := os.Getenv("GITLAB_TOKEN")
	
	if baseURL == "" || token == "" {
		return nil, fmt.Errorf("GITLAB_URL or GITLAB_TOKEN not set")
	}
	
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// FetchUsers retrieves GitLab users page by page
func (c *Client) FetchUsers() ([]models.User, error) {
	var users []models.User
	perPage := 100
	
	for page := 1; ; page++ {
		url := fmt.Sprintf("%s/api/v4/users?active=true&humans=true&exclude_external=true&page=%d&per_page=%d", 
			c.BaseURL, page, perPage)
		
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		
		req.Header.Add("PRIVATE-TOKEN", c.Token)
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		
		var pageUsers []models.User
		if err := json.Unmarshal(body, &pageUsers); err != nil {
			return nil, err
		}
		
		users = append(users, pageUsers...)
		if len(pageUsers) < perPage {
			break
		}
	}
	
	// Sort users by username
	sort.Slice(users, func(i, j int) bool {
		return users[i].Username < users[j].Username
	})
	
	return users, nil
}

// FetchUserKeys retrieves the SSH keys for a given user ID
func (c *Client) FetchUserKeys(userID int) ([]string, error) {
	url := fmt.Sprintf("%s/api/v4/users/%d/keys", c.BaseURL, userID)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Add("PRIVATE-TOKEN", c.Token)
	resp, err := c.client.Do(req)
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
