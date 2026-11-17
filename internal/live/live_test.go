package live

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"fitness-tracker/internal/api"
)

// fakeBackend is an in-memory stand-in for the data source; it implements just the
// calls the live session makes.
type fakeBackend struct {
	api.Backend

	session         *api.Session
	failPatch       atomic.Bool
	failDelete      atomic.Bool
	sessionGets     atomic.Int32
	sessionsCreated atomic.Int32
	workoutPosts    atomic.Int32
	// workoutSets is how many sets the backend's copy of the first exercise had when the
	// workout was built from it.
	workoutSets atomic.Int32
}

func clone[T any](v T) T {
	b, _ := json.Marshal(v)
	var out T
	json.Unmarshal(b, &out)
	return out
}

func (fb *fakeBackend) UserID() (string, error) { return "u1", nil }

func (fb *fakeBackend) CreateSession(userID, routineID string) error {
	id := fmt.Sprintf("s%d", fb.sessionsCreated.Add(1))
	fb.session = &api.Session{ID: id, UserID: "u1", RoutineID: routineID,
		Exercises: []api.SessionExercise{{ExerciseID: "e1", Name: "Squat", Sets: []api.WorkoutSet{{Reps: 8}}}}}
	return nil
}

func (fb *fakeBackend) SessionData(userID string) (*api.Session, error) {
	fb.sessionGets.Add(1)
	if fb.session == nil {
		return nil, nil
	}
	s := clone(*fb.session)
	return &s, nil
}

func (fb *fakeBackend) UpdateSession(s *api.Session) error {
	if fb.failPatch.Load() {
		return errors.New("update failed")
	}
	// Like the real backend: updating a session that doesn't exist (expired) is not found.
	if fb.session == nil || fb.session.ID != s.ID {
		return api.ErrNotFound
	}
	fb.session.Exercises = clone(s.Exercises)
	return nil
}

func (fb *fakeBackend) CreateWorkout(sessionID string) (string, error) {
	// The backend builds the workout from its own copy of the session.
	if fb.session == nil || fb.session.ID != sessionID {
		return "", errors.New("no such session")
	}
	fb.workoutPosts.Add(1)
	fb.workoutSets.Store(int32(len(fb.session.Exercises[0].Sets)))
	return "w1", nil
}

func (fb *fakeBackend) DeleteSession(sessionID string) error {
	if fb.failDelete.Load() {
		return errors.New("delete failed")
	}
	fb.session = nil
	return nil
}

func (fb *fakeBackend) UpdateHistory(userID, workoutID string) error { return nil }

func newManager(t *testing.T) (*Manager, *fakeBackend, string) {
	t.Helper()
	fb := &fakeBackend{}
	path := filepath.Join(t.TempDir(), "live.json")
	return New(fb, path), fb, path
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

func TestExpiredBackendSessionIsRecreatedKeepingLocalWork(t *testing.T) {
	m, fb, _ := newManager(t)
	st, _ := m.Start("u1", "r1")
	oldID := st.ID

	fb.session = nil // the backend's TTL removed it while the workout was idle

	st, err := m.Update("u1", addSet)
	if err != nil {
		t.Fatalf("an expired session should be recreated transparently: %v", err)
	}
	if st.ID == oldID || st.Dirty {
		t.Fatalf("expected a new, synced session; id %q -> %q, dirty=%v", oldID, st.ID, st.Dirty)
	}
	if got := len(fb.session.Exercises[0].Sets); got != 2 {
		t.Fatalf("local work not pushed into the recreated session: %d sets", got)
	}
}

func TestFinishCommitsLocalWorkEvenIfBackendSessionExpired(t *testing.T) {
	m, fb, path := newManager(t)
	m.Start("u1", "r1")
	m.Update("u1", addSet) // local + backend now have 2 sets
	fb.session = nil       // expires without us noticing: nothing is dirty

	if _, err := m.Finish("u1"); err != nil {
		t.Fatal(err)
	}
	if fb.workoutPosts.Load() != 1 || fb.workoutSets.Load() != 2 {
		t.Fatalf("workout should be built from the local work: workouts=%d sets=%d", fb.workoutPosts.Load(), fb.workoutSets.Load())
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
