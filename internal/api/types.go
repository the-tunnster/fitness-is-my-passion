package api

// User is the current user's profile.
type User struct {
	ID             string  `json:"id,omitempty"`
	Gender         string  `json:"gender"`
	Username       string  `json:"username"`
	Height         float64 `json:"height"`
	Weight         float64 `json:"weight"`
	DateOfBirth    string  `json:"date_of_birth"`
	UnitPreference string  `json:"unit_preference"`
}

type Exercise struct {
	ID         string   `json:"id,omitempty"`
	Name       string   `json:"name"`
	Category   string   `json:"category"`
	Variations []string `json:"variations"`
	Equipment  []string `json:"equipment"`
}

var ExerciseCategories = []string{"Strength", "Power", "Flexibility", "Stability", "Endurance", "Functional"}

type Routine struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
	Name   string `json:"name"`
}

type RoutineExercise struct {
	ExerciseID string `json:"exercise_id"`
	Name       string `json:"name"`
	TargetSets int    `json:"target_sets"`
	TargetReps int    `json:"target_reps"`
}

type FullRoutine struct {
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	Name      string            `json:"name"`
	Exercises []RoutineExercise `json:"exercises"`
}

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

// ComparisonEntry is one exercise of the latest-versus-previous workout comparison.
type ComparisonEntry struct {
	ExerciseName string  `json:"exercise_name"`
	Variation    string  `json:"variation"`
	MaxWeight    float64 `json:"max_weight"`
	WeightChange float64 `json:"weight_change"`
	Reps         float64 `json:"reps"`
	RepsChange   float64 `json:"reps_change"`
	Volume       float64 `json:"volume"`
	VolumeChange float64 `json:"volume_change"`
}

// HistoryPoint is one day of an exercise's history. Volume may be missing.
type HistoryPoint struct {
	Date   string   `json:"date"`
	Weight float64  `json:"weight"`
	Volume *float64 `json:"volume,omitempty"`
}
