package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

type Routine struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
	Name   string `json:"name"`
}

type RoutineExercise struct {
	ExerciseID  string `json:"exercise_id"`
	Name        string `json:"name"`
	TargetSets  int    `json:"target_sets"`
	TargetReps  int    `json:"target_reps"`
}

type FullRoutine struct {
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	Name      string            `json:"name"`
	Exercises []RoutineExercise `json:"exercises"`
}

// send performs a request and fails unless the status is one of ok.
func (c *Client) send(method, path string, query url.Values, body any, ok ...int) (*http.Response, error) {
	return c.sendH(method, path, query, body, nil, ok...)
}

func (c *Client) sendH(method, path string, query url.Values, body any, headers map[string]string, ok ...int) (*http.Response, error) {
	var rdr *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	u := c.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequest(method, u, rdr)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	for _, s := range ok {
		if resp.StatusCode == s {
			return resp, nil
		}
	}
	resp.Body.Close()
	return nil, &StatusError{Method: method, Path: path, Code: resp.StatusCode}
}

// StatusError is returned when the API answers with an unexpected status.
type StatusError struct {
	Method, Path string
	Code         int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("%s %s: status %d", e.Method, e.Path, e.Code)
}

// IsNotFound reports whether err is an API 404, e.g. a session that no longer exists.
func IsNotFound(err error) bool {
	var se *StatusError
	return errors.As(err, &se) && se.Code == http.StatusNotFound
}

func (c *Client) RoutineList(userID string) ([]Routine, error) {
	resp, err := c.send(http.MethodGet, "/routines/list", url.Values{"user_id": {userID}}, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var list []Routine
	return list, json.NewDecoder(resp.Body).Decode(&list)
}

func (c *Client) RoutineData(userID, routineID string) (*FullRoutine, error) {
	q := url.Values{"user_id": {userID}, "routine_id": {routineID}}
	resp, err := c.send(http.MethodGet, "/routines/data", q, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var r FullRoutine
	return &r, json.NewDecoder(resp.Body).Decode(&r)
}

func (c *Client) CreateRoutine(userID, name string, exercises []RoutineExercise) error {
	body := map[string]any{"name": name, "exercises": exercises}
	resp, err := c.send(http.MethodPost, "/routines/create", url.Values{"user_id": {userID}}, body, http.StatusCreated)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func (c *Client) UpdateRoutine(userID, routineID string, exercises []RoutineExercise) error {
	q := url.Values{"user_id": {userID}, "routine_id": {routineID}}
	resp, err := c.send(http.MethodPatch, "/routines/update", q, exercises, http.StatusOK, http.StatusNoContent)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func (c *Client) DeleteRoutine(userID, routineID string) error {
	q := url.Values{"user_id": {userID}, "routine_id": {routineID}}
	resp, err := c.send(http.MethodDelete, "/routines/delete", q, nil, http.StatusNoContent)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}
