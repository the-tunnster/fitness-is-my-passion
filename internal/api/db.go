package api

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"fitness-tracker/internal/database"
	"fitness-tracker/internal/models"
	"fitness-tracker/internal/service"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

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

// DB is the Backend that reads and writes the database directly.
type DB struct {
	mu     sync.Mutex
	userID string

	cache exerciseCache
}

var _ Backend = (*DB)(nil)

func NewDB() *DB { return &DB{} }

func parseID(what, hex string) (primitive.ObjectID, error) {
	id, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return id, fmt.Errorf("invalid %s %q", what, hex)
	}
	return id, nil
}

// notFound turns "no matching document" into ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrNotFound
	}
	return err
}

func (d *DB) UserID() (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.userID != "" {
		return d.userID, nil
	}
	u, err := d.Me()
	if err != nil {
		return "", err
	}
	d.userID = u.ID
	return d.userID, nil
}

func (d *DB) Me() (*User, error) {
	u, err := database.GetUser()
	if err != nil {
		return nil, err
	}
	return &User{
		ID:             u.ID.Hex(),
		Gender:         u.Gender,
		Username:       u.Username,
		Height:         u.Height,
		Weight:         u.Weight,
		DateOfBirth:    u.DateOfBirth,
		UnitPreference: u.UnitPreference,
	}, nil
}

func (d *DB) UpdateUser(userID string, u User) error {
	id, err := parseID("user id", userID)
	if err != nil {
		return err
	}
	return database.UpdateUser(id, bson.M{
		"username":       u.Username,
		"gender":         u.Gender,
		"dateOfBirth":    u.DateOfBirth,
		"height":         u.Height,
		"weight":         u.Weight,
		"unitPreference": u.UnitPreference,
		"updatedAt":      time.Now(),
	})
}

func exerciseFrom(e models.Exercise) Exercise {
	return Exercise{ID: e.ID.Hex(), Name: e.Name, Category: e.Category, Variations: e.Variations, Equipment: e.Equipment}
}

// ExerciseList returns a copy of the (briefly cached) exercise list.
func (d *DB) ExerciseList() ([]Exercise, error) {
	d.cache.mu.Lock()
	if d.cache.list != nil && time.Since(d.cache.listAt) < exerciseCacheTTL {
		out := append([]Exercise(nil), d.cache.list...)
		d.cache.mu.Unlock()
		return out, nil
	}
	d.cache.mu.Unlock()

	rows, err := database.GetExerciseList()
	if err != nil {
		return nil, err
	}
	var list []Exercise
	for _, e := range rows {
		list = append(list, exerciseFrom(e))
	}

	d.cache.mu.Lock()
	d.cache.list, d.cache.listAt = list, time.Now()
	d.cache.mu.Unlock()
	return append([]Exercise(nil), list...), nil
}

func (d *DB) ExerciseData(exerciseID string) (*Exercise, error) {
	d.cache.mu.Lock()
	if hit, ok := d.cache.data[exerciseID]; ok && time.Since(hit.at) < exerciseCacheTTL {
		e := hit.exercise
		d.cache.mu.Unlock()
		return &e, nil
	}
	d.cache.mu.Unlock()

	id, err := parseID("exercise id", exerciseID)
	if err != nil {
		return nil, err
	}
	m, err := database.GetExerciseData(id)
	if err != nil {
		return nil, notFound(err)
	}
	e := exerciseFrom(m)

	d.cache.mu.Lock()
	if d.cache.data == nil {
		d.cache.data = map[string]cachedExercise{}
	}
	d.cache.data[exerciseID] = cachedExercise{exercise: e, at: time.Now()}
	d.cache.mu.Unlock()
	return &e, nil
}

func (d *DB) CreateExercise(e Exercise) (string, error) {
	id, err := database.CreateExercise(models.Exercise{
		Name: e.Name, Category: e.Category, Variations: e.Variations, Equipment: e.Equipment,
	})
	if err != nil {
		return "", err
	}
	d.cache.invalidate()
	return id.Hex(), nil
}

func (d *DB) UpdateExercise(e Exercise) error {
	id, err := parseID("exercise id", e.ID)
	if err != nil {
		return err
	}
	err = database.UpdateExercise(id, bson.M{
		"name":       e.Name,
		"category":   e.Category,
		"variations": e.Variations,
		"equipment":  e.Equipment,
		"updatedAt":  time.Now(),
	})
	if err != nil {
		return notFound(err)
	}
	d.cache.invalidate()
	return nil
}

func (d *DB) RoutineList(userID string) ([]Routine, error) {
	uid, err := parseID("user id", userID)
	if err != nil {
		return nil, err
	}
	list, err := database.GetUserRoutines(uid)
	if err != nil {
		return nil, err
	}
	var out []Routine
	for _, r := range list {
		out = append(out, Routine{ID: r.ID.Hex(), UserID: r.UserID.Hex(), Name: r.Name})
	}
	return out, nil
}

func (d *DB) RoutineData(userID, routineID string) (*FullRoutine, error) {
	uid, err := parseID("user id", userID)
	if err != nil {
		return nil, err
	}
	rid, err := parseID("routine id", routineID)
	if err != nil {
		return nil, err
	}
	r, err := database.GetRoutineData(uid, rid)
	if err != nil {
		return nil, notFound(err)
	}
	out := &FullRoutine{ID: r.ID.Hex(), UserID: r.UserID.Hex(), Name: r.Name}
	for _, e := range r.Exercises {
		out.Exercises = append(out.Exercises, RoutineExercise{
			ExerciseID: e.ExerciseID.Hex(), Name: e.Name, TargetSets: e.TargetSets, TargetReps: e.TargetReps,
		})
	}
	return out, nil
}

func routineExercisesToModels(in []RoutineExercise) ([]models.RoutineExercise, error) {
	var out []models.RoutineExercise
	for _, e := range in {
		id, err := parseID("exercise id", e.ExerciseID)
		if err != nil {
			return nil, err
		}
		out = append(out, models.RoutineExercise{
			ExerciseID: id, Name: e.Name, TargetSets: e.TargetSets, TargetReps: e.TargetReps,
		})
	}
	return out, nil
}

func (d *DB) CreateRoutine(userID, name string, exercises []RoutineExercise) error {
	uid, err := parseID("user id", userID)
	if err != nil {
		return err
	}
	exs, err := routineExercisesToModels(exercises)
	if err != nil {
		return err
	}
	_, err = database.CreateRoutine(models.FullRoutine{UserID: uid, Name: name, Exercises: exs})
	return err
}

func (d *DB) UpdateRoutine(userID, routineID string, exercises []RoutineExercise) error {
	uid, err := parseID("user id", userID)
	if err != nil {
		return err
	}
	rid, err := parseID("routine id", routineID)
	if err != nil {
		return err
	}
	exs, err := routineExercisesToModels(exercises)
	if err != nil {
		return err
	}
	return notFound(database.UpdateRoutine(rid, uid, bson.M{"exercises": exs}))
}

func (d *DB) DeleteRoutine(userID, routineID string) error {
	uid, err := parseID("user id", userID)
	if err != nil {
		return err
	}
	rid, err := parseID("routine id", routineID)
	if err != nil {
		return err
	}
	return database.DeleteRoutine(uid, rid)
}

func (d *DB) SessionData(userID string) (*Session, error) {
	uid, err := parseID("user id", userID)
	if err != nil {
		return nil, err
	}
	s, err := database.GetUserSessionData(uid)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if s.ID.IsZero() {
		return nil, nil
	}

	out := &Session{ID: s.ID.Hex(), UserID: s.UserID.Hex(), ExerciseIndex: s.ExerciseIndex}
	if !s.RoutineID.IsZero() {
		out.RoutineID = s.RoutineID.Hex()
	}
	last := s.LastUpdate.Time().Format(time.RFC3339Nano)
	out.LastUpdate = &last
	for _, e := range s.Exercises {
		name := e.Name
		if name == "" {
			if ex, err := database.GetExerciseData(e.ExerciseID); err == nil {
				name = ex.Name
			}
		}
		se := SessionExercise{ExerciseID: e.ExerciseID.Hex(), Equipment: e.Equipment, Variation: e.Variation, Name: name}
		for _, set := range e.Sets {
			se.Sets = append(se.Sets, WorkoutSet{Reps: set.Reps, Weight: set.Weight})
		}
		out.Exercises = append(out.Exercises, se)
	}
	return out, nil
}

func (d *DB) CreateSession(userID, routineID string) error {
	session, err := service.GenerateWorkoutSession(userID, routineID)
	if err != nil {
		return err
	}
	_, err = database.UpsertSession(*session)
	return err
}

func (d *DB) UpdateSession(s *Session) error {
	id, err := parseID("session id", s.ID)
	if err != nil {
		return err
	}
	if s.ExerciseIndex < 0 {
		return fmt.Errorf("invalid exercise index %d", s.ExerciseIndex)
	}
	var exercises []models.WorkoutExercise
	for _, e := range s.Exercises {
		eid, err := parseID("exercise id", e.ExerciseID)
		if err != nil {
			return err
		}
		sets := make([]models.WorkoutSet, 0, len(e.Sets))
		for _, set := range e.Sets {
			sets = append(sets, models.WorkoutSet{Reps: set.Reps, Weight: set.Weight})
		}
		exercises = append(exercises, models.WorkoutExercise{
			ExerciseID: eid, Equipment: e.Equipment, Variation: e.Variation, Sets: sets, Name: e.Name,
		})
	}
	return notFound(database.UpdateSession(id, bson.M{
		"exercises":     exercises,
		"exerciseIndex": s.ExerciseIndex,
		"lastUpdated":   primitive.NewDateTimeFromTime(time.Now()),
	}))
}

func (d *DB) DeleteSession(sessionID string) error {
	id, err := parseID("session id", sessionID)
	if err != nil {
		return err
	}
	return database.DeleteSession(id)
}

func (d *DB) CreateWorkout(sessionID string) (string, error) {
	id, err := parseID("session id", sessionID)
	if err != nil {
		return "", err
	}
	session, err := database.GetSessionData(id)
	if err != nil {
		return "", err
	}
	workoutID, err := database.CreateWorkout(models.FullWorkout{
		UserID:      session.UserID,
		RoutineID:   session.RoutineID,
		WorkoutDate: primitive.NewDateTimeFromTime(time.Now()),
		Exercises:   session.Exercises,
	})
	if err != nil {
		return "", err
	}
	return workoutID.Hex(), nil
}

func (d *DB) UpdateHistory(userID, workoutID string) error {
	return service.UpdateExerciseHistory(userID, workoutID)
}

func (d *DB) WorkoutComparison(userID, routineID string) ([]ComparisonEntry, error) {
	uid, err := parseID("user id", userID)
	if err != nil {
		return nil, err
	}
	rid, err := parseID("routine id", routineID)
	if err != nil {
		return nil, err
	}
	workouts, err := database.GetLastTwoWorkouts(uid, rid)
	if err != nil {
		return nil, err
	}
	if len(workouts) < 2 {
		return nil, nil
	}
	changes := service.CompareWorkouts(workouts[0], workouts[1], func(id primitive.ObjectID) string {
		name, _ := database.GetExerciseName(id)
		return name
	})
	entries := make([]ComparisonEntry, 0, len(changes))
	for _, c := range changes {
		entries = append(entries, ComparisonEntry{
			ExerciseName: c.ExerciseName,
			Variation:    c.Variation,
			MaxWeight:    c.MaxWeight,
			WeightChange: c.WeightChange,
			Reps:         float64(c.TotalReps),
			RepsChange:   float64(c.RepsChange),
			Volume:       c.TotalVolume,
			VolumeChange: c.VolumeChange,
		})
	}
	return entries, nil
}

func (d *DB) ExerciseHistory(userID, exerciseID string) ([]HistoryPoint, error) {
	uid, err := parseID("user id", userID)
	if err != nil {
		return nil, err
	}
	eid, err := parseID("exercise id", exerciseID)
	if err != nil {
		return nil, err
	}
	history, err := database.GetExerciseHistoryData(eid, uid)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var points []HistoryPoint
	for _, p := range service.SummarizeHistory(history) {
		volume := p.Volume
		points = append(points, HistoryPoint{Date: p.Date, Weight: p.Weight, Volume: &volume})
	}
	return points, nil
}
