package service

import (
	"fitness-tracker/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MetricChange is one exercise's stats in the latest workout and how they moved since
// the one before it.
type MetricChange struct {
	ExerciseName string  `json:"exercise_name"`
	Variation    string  `json:"variation"`
	MaxWeight    float64 `json:"max_weight"`
	TotalReps    int     `json:"reps"`
	TotalVolume  float64 `json:"volume"`
	WeightChange float64 `json:"weight_change"`
	RepsChange   int     `json:"reps_change"`
	VolumeChange float64 `json:"volume_change"`
}

type exerciseTotals struct {
	maxWeight   float64
	totalReps   int
	totalVolume float64
}

func totalsOf(sets []models.WorkoutSet) exerciseTotals {
	var t exerciseTotals
	for _, s := range sets {
		t.totalReps += s.Reps
		t.totalVolume += float64(s.Reps) * s.Weight
		if s.Weight > t.maxWeight {
			t.maxWeight = s.Weight
		}
	}
	return t
}

// CompareWorkouts compares the latest workout with the previous one. Exercises are
// matched by name and variation and returned in the order they were done in the latest
// workout; exercises not present in the previous workout show their own totals as the change.
func CompareWorkouts(latest, previous models.FullWorkout, nameOf func(primitive.ObjectID) string) []MetricChange {
	byKey := make(map[string]MetricChange)
	var order []string

	for _, ex := range latest.Exercises {
		name := nameOf(ex.ExerciseID)
		key := name + "|" + ex.Variation
		t := totalsOf(ex.Sets)
		if _, seen := byKey[key]; !seen {
			order = append(order, key)
		}
		byKey[key] = MetricChange{
			ExerciseName: name,
			Variation:    ex.Variation,
			MaxWeight:    t.maxWeight,
			TotalReps:    t.totalReps,
			TotalVolume:  t.totalVolume,
			WeightChange: t.maxWeight,
			RepsChange:   t.totalReps,
			VolumeChange: t.totalVolume,
		}
	}

	for _, ex := range previous.Exercises {
		key := nameOf(ex.ExerciseID) + "|" + ex.Variation
		current, ok := byKey[key]
		if !ok {
			continue
		}
		t := totalsOf(ex.Sets)
		current.RepsChange = current.TotalReps - t.totalReps
		current.WeightChange = current.MaxWeight - t.maxWeight
		current.VolumeChange = current.TotalVolume - t.totalVolume
		byKey[key] = current
	}

	changes := make([]MetricChange, 0, len(order))
	for _, key := range order {
		changes = append(changes, byKey[key])
	}
	return changes
}
