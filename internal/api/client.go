package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Client talks to the existing backend API as-is.
type Client struct {
	BaseURL string
	HTTP    *http.Client
	// Mock serves generated analytics data (MOCK_ANALYTICS=1).
	Mock bool

	mu     sync.Mutex
	userID string

	cache exerciseCache
}

// UserID returns the current user's id, fetched once from /me.
func (c *Client) UserID() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.userID != "" {
		return c.userID, nil
	}
	u, err := c.Me()
	if err != nil {
		return "", err
	}
	c.userID = u.ID
	return c.userID, nil
}

func NewClient() *Client {
	base := os.Getenv("BACKEND_URL")
	if base == "" {
		base = "http://opi5:8090"
	}
	return &Client{
		BaseURL: strings.TrimRight(base, "/"),
		HTTP:    &http.Client{Timeout: 10 * time.Second},
		Mock:    os.Getenv("MOCK_ANALYTICS") == "1",
	}
}

// User is the response of GET /me and the body of PATCH /user/update.
type User struct {
	ID             string  `json:"id,omitempty"`
	Gender         string  `json:"gender"`
	Username       string  `json:"username"`
	Height         float64 `json:"height"`
	Weight         float64 `json:"weight"`
	DateOfBirth    string  `json:"date_of_birth"`
	UnitPreference string  `json:"unit_preference"`
}

func (c *Client) Me() (*User, error) {
	resp, err := c.HTTP.Get(c.BaseURL + "/me")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET /me: status %d", resp.StatusCode)
	}
	var u User
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (c *Client) UpdateUser(userID string, u User) error {
	u.ID = ""
	body, err := json.Marshal(u)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPatch,
		c.BaseURL+"/user/update?user_id="+url.QueryEscape(userID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("PATCH /user/update: status %d", resp.StatusCode)
	}
	return nil
}
