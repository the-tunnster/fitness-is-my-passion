package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Exercise is the shape returned by GET /exercise/list.
type Exercise struct {
	ID         string   `json:"id,omitempty"`
	Name       string   `json:"name"`
	Category   string   `json:"category"`
	Variations []string `json:"variations"`
	Equipment  []string `json:"equipment"`
}

var ExerciseCategories = []string{"Strength", "Power", "Flexibility", "Stability", "Endurance", "Functional"}

// exerciseCacheTTL bounds how stale exercise data can get if it is edited outside this app.
const exerciseCacheTTL = 5 * time.Minute

type cachedExercise struct {
	exercise Exercise
	at       time.Time
}

// exerciseCache avoids re-reading rarely changing exercise data on every request.
type exerciseCache struct {
	mu     sync.Mutex
	list   []Exercise
	listAt time.Time
	data   map[string]cachedExercise
}

func (c *exerciseCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.list, c.listAt, c.data = nil, time.Time{}, nil
}

// ExerciseList returns a copy of the (briefly cached) exercise list.
func (c *Client) ExerciseList() ([]Exercise, error) {
	c.cache.mu.Lock()
	if c.cache.list != nil && time.Since(c.cache.listAt) < exerciseCacheTTL {
		out := append([]Exercise(nil), c.cache.list...)
		c.cache.mu.Unlock()
		return out, nil
	}
	c.cache.mu.Unlock()

	resp, err := c.HTTP.Get(c.BaseURL + "/exercise/list")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET /exercise/list: status %d", resp.StatusCode)
	}
	var list []Exercise
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}

	c.cache.mu.Lock()
	c.cache.list, c.cache.listAt = list, time.Now()
	c.cache.mu.Unlock()
	return append([]Exercise(nil), list...), nil
}

// CreateExercise returns the new exercise id.
func (c *Client) CreateExercise(e Exercise) (string, error) {
	e.ID = ""
	body, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	resp, err := c.HTTP.Post(c.BaseURL+"/exercise/create", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("POST /exercise/create: status %d", resp.StatusCode)
	}
	c.cache.invalidate()
	var id string
	if err := json.NewDecoder(resp.Body).Decode(&id); err != nil {
		return "", err
	}
	return id, nil
}

func (c *Client) UpdateExercise(e Exercise) error {
	body, err := json.Marshal(e)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPatch,
		c.BaseURL+"/exercise/update?exercise_id="+url.QueryEscape(e.ID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("PATCH /exercise/update: status %d", resp.StatusCode)
	}
	c.cache.invalidate()
	return nil
}
