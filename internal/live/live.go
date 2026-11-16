// Package live keeps the active workout session in a local file.
//
// While a workout is running the file is the source of truth: reads never touch the
// backend, and every change is written to the file first and then pushed to the backend
// session. The backend is only read when no file exists (to pick up a session started
// elsewhere) and once right after a session is started.
package live

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"fitness-tracker/internal/api"
)

var (
	ErrNoSession       = errors.New("no active workout session")
	ErrNotSynced       = errors.New("session has changes that could not be pushed to the backend")
	ErrCreateWorkout   = errors.New("couldn't save workout")
	ErrClearSession    = errors.New("couldn't clear the backend session")
	ErrStartIncomplete = errors.New("session was created but could not be read back")
	ErrCannotRecreate  = errors.New("backend session is gone and has no routine to recreate it from")
)

// State is the local session file: the backend session plus sync bookkeeping.
type State struct {
	api.Session
	// Dirty means the local copy has changes the backend hasn't received yet.
	Dirty bool `json:"dirty"`
	// WorkoutID is set once the workout is committed, so retrying Finish after a
	// partial failure can't create a second workout.
	WorkoutID string `json:"workout_id,omitempty"`
}

type Manager struct {
	client *api.Client
	path   string
	mu     sync.Mutex
}

func New(client *api.Client, path string) *Manager {
	return &Manager{client: client, path: path}
}

// load returns nil when there is no (readable) file.
func (m *Manager) load() *State {
	b, err := os.ReadFile(m.path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("live session: %v", err)
		}
		return nil
	}
	var st State
	if err := json.Unmarshal(b, &st); err != nil || st.ID == "" {
		log.Printf("live session: ignoring unreadable file %s: %v", m.path, err)
		return nil
	}
	return &st
}

// save writes atomically so a crash can't leave a half-written file.
func (m *Manager) save(st *State) error {
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}
	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, m.path)
}

func (m *Manager) clear() error {
	if err := os.Remove(m.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// current returns the local session, adopting the backend's session if there is no file.
func (m *Manager) current(userID string) (*State, error) {
	if st := m.load(); st != nil {
		return st, nil
	}
	sess, err := m.client.SessionData(userID)
	if err != nil || sess == nil {
		return nil, err
	}
	st := &State{Session: *sess}
	if err := m.save(st); err != nil {
		return nil, err
	}
	return st, nil
}

// push sends the full exercise list, so a retry after any failure is safe. If the backend
// no longer has the session (sessions expire after a while), it is recreated from the
// routine and the local work is pushed into the new one.
func (m *Manager) push(userID string, st *State) error {
	err := m.client.UpdateSession(&st.Session)
	if api.IsNotFound(err) {
		if err = m.recreate(userID, st); err == nil {
			err = m.client.UpdateSession(&st.Session)
		}
	}
	if err != nil {
		return err
	}
	st.Dirty = false
	return m.save(st)
}

// recreate makes a fresh backend session for the same routine and adopts its id, keeping
// the locally recorded exercises.
func (m *Manager) recreate(userID string, st *State) error {
	if st.RoutineID == "" {
		return ErrCannotRecreate
	}
	if err := m.client.CreateSession(userID, st.RoutineID); err != nil {
		return err
	}
	sess, err := m.client.SessionData(userID)
	if err != nil {
		return err
	}
	if sess == nil {
		return ErrStartIncomplete
	}
	st.ID = sess.ID
	return m.save(st)
}

// Current returns the active session, or nil when there isn't one.
func (m *Manager) Current(userID string) (*State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current(userID)
}

// Start creates the backend session, reads it back once and stores it locally.
func (m *Manager) Start(userID, routineID string) (*State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.client.CreateSession(userID, routineID); err != nil {
		return nil, err
	}
	sess, err := m.client.SessionData(userID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, ErrStartIncomplete
	}
	st := &State{Session: *sess}
	return st, m.save(st)
}

// Update applies fn to the session, stores it locally and pushes it to the backend.
// If the push fails the local change is kept and the returned error says so; the
// returned state is always the up-to-date local one.
func (m *Manager) Update(userID string, fn func(*api.Session)) (*State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st, err := m.current(userID)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, ErrNoSession
	}
	fn(&st.Session)
	st.Dirty = true
	if err := m.save(st); err != nil {
		return st, err
	}
	return st, m.push(userID, st)
}

// Result reports the non-fatal outcome of finishing a workout.
type Result struct {
	WorkoutID  string
	HistoryErr error
}

// Finish pushes the session first (recreating it if the backend dropped it), commits it
// as a workout, removes the backend session and finally the local file. It can be
// retried after a failure.
func (m *Manager) Finish(userID string) (Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st, err := m.current(userID)
	if err != nil {
		return Result{}, err
	}
	if st == nil {
		return Result{}, ErrNoSession
	}

	// Always push before committing (not only when dirty): the backend builds the workout
	// from its own copy, which may have expired. Skipped once the workout exists, so a
	// retry after a partial failure can't resurrect the session.
	if st.WorkoutID == "" {
		if err := m.push(userID, st); err != nil {
			return Result{}, fmt.Errorf("%w: %v", ErrNotSynced, err)
		}
		id, err := m.client.CreateWorkout(st.ID, userID)
		if err != nil {
			return Result{}, fmt.Errorf("%w: %v", ErrCreateWorkout, err)
		}
		st.WorkoutID = id
		if err := m.save(st); err != nil {
			return Result{WorkoutID: id}, err
		}
	}
	if err := m.deleteBackend(st.ID); err != nil {
		return Result{WorkoutID: st.WorkoutID}, fmt.Errorf("%w: %v", ErrClearSession, err)
	}
	res := Result{WorkoutID: st.WorkoutID, HistoryErr: m.client.UpdateHistory(userID, st.WorkoutID)}
	return res, m.clear()
}

// Cancel discards the workout everywhere.
func (m *Manager) Cancel(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	st, err := m.current(userID)
	if err != nil {
		return err
	}
	if st == nil {
		return ErrNoSession
	}
	if err := m.deleteBackend(st.ID); err != nil {
		return fmt.Errorf("%w: %v", ErrClearSession, err)
	}
	return m.clear()
}

// deleteBackend removes the backend session. If that fails but the backend no longer
// has this session anyway (e.g. it was removed elsewhere), that counts as success.
func (m *Manager) deleteBackend(sessionID string) error {
	err := m.client.DeleteSession(sessionID)
	if err == nil {
		return nil
	}
	if userID, uerr := m.client.UserID(); uerr == nil {
		if sess, serr := m.client.SessionData(userID); serr == nil && (sess == nil || sess.ID != sessionID) {
			return nil
		}
	}
	return err
}
