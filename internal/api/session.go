package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type WorkoutSet struct {
	Reps   int     `json:"reps"`
	Weight float64 `json:"weight"`
}

type SessionExercise struct {
	Sets       []WorkoutSet `json:"sets"`
	Equipment  string       `json:"equipment"`
	Variation  string       `json:"variation"`
	ExerciseID string       `json:"exercise_id"`
	Name       string       `json:"name"`
}

// DisplayName matches the label Streamlit shows in the exercise selector.
func (e SessionExercise) DisplayName() string {
	return e.Name + " | " + e.Variation + ", " + e.Equipment
}

type Session struct {
	ID            string            `json:"id"`
	UserID        string            `json:"user_id"`
	RoutineID     string            `json:"routine_id"`
	Exercises     []SessionExercise `json:"exercises"`
	ExerciseIndex int               `json:"exercise_index"`
	LastUpdate    *string           `json:"last_update"`
}

// emptyObjectID is what the backend returns when there is no active session.
const emptyObjectID = "000000000000000000000000"

// SessionData returns nil (and no error) when the user has no active session.
func (c *Client) SessionData(userID string) (*Session, error) {
	resp, err := c.send(http.MethodGet, "/session/data", url.Values{"user_id": {userID}}, nil, http.StatusOK, http.StatusNotFound)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	var s Session
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, err
	}
	if s.ID == emptyObjectID || s.ID == "" {
		return nil, nil
	}
	return &s, nil
}

func (c *Client) CreateSession(userID, routineID string) error {
	q := url.Values{"user_id": {userID}, "routine_id": {routineID}}
	resp, err := c.sendH(http.MethodPost, "/session/create", q, nil, map[string]string{"X-User-ID": userID}, http.StatusCreated)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func (c *Client) UpdateSession(s *Session) error {
	q := url.Values{"session_id": {s.ID}, "exercise_index": {strconv.Itoa(s.ExerciseIndex)}}
	resp, err := c.send(http.MethodPatch, "/session/update", q, s.Exercises, http.StatusOK, http.StatusNoContent)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func (c *Client) DeleteSession(sessionID string) error {
	resp, err := c.send(http.MethodDelete, "/session/delete", url.Values{"session_id": {sessionID}}, nil, http.StatusNoContent)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// CreateWorkout commits a session and returns the new workout id.
func (c *Client) CreateWorkout(sessionID, userID string) (string, error) {
	q := url.Values{"session_id": {sessionID}, "user_id": {userID}}
	resp, err := c.sendH(http.MethodPost, "/workouts/create", q, nil, map[string]string{"X-User-ID": userID}, http.StatusCreated)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var id string
	if err := json.NewDecoder(resp.Body).Decode(&id); err != nil {
		return "", err
	}
	if id == "" {
		return "", errors.New("empty workout id")
	}
	return id, nil
}

func (c *Client) UpdateHistory(userID, workoutID string) error {
	q := url.Values{"user_id": {userID}, "workout_id": {workoutID}}
	resp, err := c.send(http.MethodPatch, "/history/update", q, nil, http.StatusNoContent)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func (c *Client) ExerciseData(exerciseID string) (*Exercise, error) {
	c.cache.mu.Lock()
	if hit, ok := c.cache.data[exerciseID]; ok && time.Since(hit.at) < exerciseCacheTTL {
		e := hit.exercise
		c.cache.mu.Unlock()
		return &e, nil
	}
	c.cache.mu.Unlock()

	resp, err := c.send(http.MethodGet, "/exercise/data", url.Values{"exercise_id": {exerciseID}}, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var e Exercise
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		return nil, err
	}

	c.cache.mu.Lock()
	if c.cache.data == nil {
		c.cache.data = map[string]cachedExercise{}
	}
	c.cache.data[exerciseID] = cachedExercise{exercise: e, at: time.Now()}
	c.cache.mu.Unlock()
	return &e, nil
}
