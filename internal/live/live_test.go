package live

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"fitness-tracker/internal/api"
)

// fakeBackend implements just the endpoints the live session uses.
type fakeBackend struct {
	session      *api.Session
	failPatch    atomic.Bool
	failDelete   atomic.Bool
	sessionGets  atomic.Int32
	workoutPosts atomic.Int32
}

func newManager(t *testing.T) (*Manager, *fakeBackend, string) {
	t.Helper()
	fb := &fakeBackend{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /me", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"id": "u1", "username": "t"})
	})
	mux.HandleFunc("POST /session/create", func(w http.ResponseWriter, r *http.Request) {
		fb.session = &api.Session{ID: "s1", UserID: "u1", RoutineID: r.URL.Query().Get("routine_id"),
			Exercises: []api.SessionExercise{{ExerciseID: "e1", Name: "Squat", Sets: []api.WorkoutSet{{Reps: 8}}}}}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode("s1")
	})
	mux.HandleFunc("GET /session/data", func(w http.ResponseWriter, r *http.Request) {
		fb.sessionGets.Add(1)
		if fb.session == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(fb.session)
	})
	mux.HandleFunc("PATCH /session/update", func(w http.ResponseWriter, r *http.Request) {
		if fb.failPatch.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewDecoder(r.Body).Decode(&fb.session.Exercises)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /workouts/create", func(w http.ResponseWriter, r *http.Request) {
		fb.workoutPosts.Add(1)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode("w1")
	})
	mux.HandleFunc("DELETE /session/delete", func(w http.ResponseWriter, r *http.Request) {
		if fb.failDelete.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		fb.session = nil
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("PATCH /history/update", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	path := filepath.Join(t.TempDir(), "live.json")
	return New(&api.Client{BaseURL: srv.URL, HTTP: srv.Client()}, path), fb, path
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func addSet(s *api.Session) {
	s.Exercises[0].Sets = append(s.Exercises[0].Sets, api.WorkoutSet{Reps: 5, Weight: 20})
}

func TestStartStoresLocallyAndReadsStayLocal(t *testing.T) {
	m, fb, path := newManager(t)
	if _, err := m.Start("u1", "r1"); err != nil {
		t.Fatal(err)
	}
	if !exists(path) {
		t.Fatal("session file not created")
	}
	reads := fb.sessionGets.Load()
	for i := 0; i < 3; i++ {
		if st, err := m.Current("u1"); err != nil || st == nil {
			t.Fatalf("Current: %v %v", st, err)
		}
	}
	if got := fb.sessionGets.Load(); got != reads {
		t.Fatalf("Current hit the backend: %d reads after start, %d now", reads, got)
	}
}

func TestNoFileAdoptsBackendSessionOnce(t *testing.T) {
	m, fb, path := newManager(t)
	fb.session = &api.Session{ID: "s9", Exercises: []api.SessionExercise{{Name: "Row"}}}

	st, err := m.Current("u1")
	if err != nil || st == nil || st.ID != "s9" {
		t.Fatalf("expected adopted session, got %+v, %v", st, err)
	}
	if !exists(path) {
		t.Fatal("adopted session not written to file")
	}
	if _, err := m.Current("u1"); err != nil || fb.sessionGets.Load() != 1 {
		t.Fatalf("second Current should be local; backend reads = %d", fb.sessionGets.Load())
	}
}

func TestNoFileAndNoBackendSession(t *testing.T) {
	m, _, path := newManager(t)
	st, err := m.Current("u1")
	if err != nil || st != nil {
		t.Fatalf("expected no session, got %+v, %v", st, err)
	}
	if exists(path) {
		t.Fatal("file should not exist")
	}
}

func TestFailedPushKeepsChangeAndBlocksFinish(t *testing.T) {
	m, fb, path := newManager(t)
	m.Start("u1", "r1")

	fb.failPatch.Store(true)
	st, err := m.Update("u1", addSet)
	if err == nil {
		t.Fatal("expected push error")
	}
	if !st.Dirty || len(st.Exercises[0].Sets) != 2 {
		t.Fatalf("local change lost: %+v", st)
	}
	if cur, _ := m.Current("u1"); len(cur.Exercises[0].Sets) != 2 || !cur.Dirty {
		t.Fatal("change not persisted in file")
	}

	if _, err := m.Finish("u1"); !errors.Is(err, ErrNotSynced) {
		t.Fatalf("Finish should refuse while unsynced, got %v", err)
	}
	if fb.workoutPosts.Load() != 0 {
		t.Fatal("workout must not be created while unsynced")
	}

	fb.failPatch.Store(false)
	if _, err := m.Finish("u1"); err != nil {
		t.Fatal(err)
	}
	if fb.workoutPosts.Load() != 1 || fb.session != nil || exists(path) {
		t.Fatalf("finish incomplete: workouts=%d session=%v file=%v", fb.workoutPosts.Load(), fb.session, exists(path))
	}
}

func TestPushSendsLatestStateAfterRecovery(t *testing.T) {
	m, fb, _ := newManager(t)
	m.Start("u1", "r1")
	fb.failPatch.Store(true)
	m.Update("u1", addSet)
	fb.failPatch.Store(false)
	st, err := m.Update("u1", func(s *api.Session) { s.ExerciseIndex = 0 })
	if err != nil || st.Dirty {
		t.Fatalf("expected clean sync, got dirty=%v err=%v", st.Dirty, err)
	}
	if len(fb.session.Exercises[0].Sets) != 2 {
		t.Fatal("earlier unsynced change was not included in the next push")
	}
}

func TestFinishRetryDoesNotDuplicateWorkout(t *testing.T) {
	m, fb, path := newManager(t)
	m.Start("u1", "r1")

	fb.failDelete.Store(true)
	if _, err := m.Finish("u1"); !errors.Is(err, ErrClearSession) {
		t.Fatalf("expected ErrClearSession, got %v", err)
	}
	if st := m.load(); st == nil || st.WorkoutID != "w1" {
		t.Fatal("workout id not remembered after partial failure")
	}

	fb.failDelete.Store(false)
	if _, err := m.Finish("u1"); err != nil {
		t.Fatal(err)
	}
	if fb.workoutPosts.Load() != 1 {
		t.Fatalf("workout created %d times", fb.workoutPosts.Load())
	}
	if exists(path) {
		t.Fatal("file should be removed after finishing")
	}
}

func TestCancelWhenBackendSessionAlreadyGone(t *testing.T) {
	m, fb, path := newManager(t)
	m.Start("u1", "r1")
	fb.session = nil
	fb.failDelete.Store(true)

	if err := m.Cancel("u1"); err != nil {
		t.Fatalf("cancel should succeed when the backend no longer has the session: %v", err)
	}
	if exists(path) {
		t.Fatal("file should be removed")
	}
}
