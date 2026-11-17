// Package api is the data source the web UI works against: plain data types and the
// Backend interface that reads and writes them. DB is the implementation backed by the
// database.
package api

import "errors"

// ErrNotFound means the record being changed no longer exists, e.g. an expired session.
var ErrNotFound = errors.New("not found")

// IsNotFound reports whether err means the record doesn't exist.
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

// Backend is everything the UI and the live workout session need from the data source.
type Backend interface {
	// UserID returns the current user's id.
	UserID() (string, error)
	Me() (*User, error)
	UpdateUser(userID string, u User) error

	ExerciseList() ([]Exercise, error)
	ExerciseData(exerciseID string) (*Exercise, error)
	// CreateExercise returns the new exercise id.
	CreateExercise(e Exercise) (string, error)
	UpdateExercise(e Exercise) error

	RoutineList(userID string) ([]Routine, error)
	RoutineData(userID, routineID string) (*FullRoutine, error)
	CreateRoutine(userID, name string, exercises []RoutineExercise) error
	UpdateRoutine(userID, routineID string, exercises []RoutineExercise) error
	DeleteRoutine(userID, routineID string) error

	// SessionData returns nil (and no error) when the user has no active session.
	SessionData(userID string) (*Session, error)
	CreateSession(userID, routineID string) error
	// UpdateSession returns ErrNotFound when the session no longer exists.
	UpdateSession(s *Session) error
	DeleteSession(sessionID string) error
	// CreateWorkout commits a session and returns the new workout id.
	CreateWorkout(sessionID string) (string, error)
	UpdateHistory(userID, workoutID string) error

	// WorkoutComparison returns nil (and no error) when there aren't enough workouts to compare.
	WorkoutComparison(userID, routineID string) ([]ComparisonEntry, error)
	ExerciseHistory(userID, exerciseID string) ([]HistoryPoint, error)
}
