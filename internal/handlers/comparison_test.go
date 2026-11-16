package handlers

import (
	"reflect"
	"testing"

	"fitness-tracker/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func oid(b byte) primitive.ObjectID { return primitive.ObjectID{b} }

func exercise(id byte, variation string, sets ...models.WorkoutSet) models.WorkoutExercise {
	return models.WorkoutExercise{ExerciseID: oid(id), Variation: variation, Sets: sets}
}

var names = map[primitive.ObjectID]string{oid(1): "Squat", oid(2): "Bench", oid(3): "Row", oid(4): "Curl"}

func nameOf(id primitive.ObjectID) string { return names[id] }

func TestCompareWorkoutsKeepsLatestWorkoutOrder(t *testing.T) {
	latest := models.FullWorkout{Exercises: []models.WorkoutExercise{
		exercise(3, "Seated", models.WorkoutSet{Reps: 10, Weight: 40}),
		exercise(1, "High bar", models.WorkoutSet{Reps: 5, Weight: 100}),
		exercise(4, "None", models.WorkoutSet{Reps: 12, Weight: 15}),
		exercise(2, "Flat", models.WorkoutSet{Reps: 8, Weight: 80}),
	}}
	want := []string{"Row", "Squat", "Curl", "Bench"}

	// Go randomises map iteration, so a map-driven order fails this within a few runs.
	for i := 0; i < 200; i++ {
		var got []string
		for _, c := range compareWorkouts(latest, models.FullWorkout{}, nameOf) {
			got = append(got, c.ExerciseName)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("run %d: order %v, want %v", i, got, want)
		}
	}
}

func TestCompareWorkoutsComputesChanges(t *testing.T) {
	latest := models.FullWorkout{Exercises: []models.WorkoutExercise{
		exercise(1, "High bar", models.WorkoutSet{Reps: 5, Weight: 100}, models.WorkoutSet{Reps: 5, Weight: 105}),
		exercise(4, "None", models.WorkoutSet{Reps: 12, Weight: 15}), // new this time
	}}
	previous := models.FullWorkout{Exercises: []models.WorkoutExercise{
		exercise(1, "High bar", models.WorkoutSet{Reps: 5, Weight: 95}, models.WorkoutSet{Reps: 4, Weight: 100}),
		exercise(2, "Flat", models.WorkoutSet{Reps: 8, Weight: 80}), // dropped since
	}}

	got := compareWorkouts(latest, previous, nameOf)
	if len(got) != 2 {
		t.Fatalf("expected the two exercises of the latest workout, got %d", len(got))
	}

	squat := got[0]
	if squat.MaxWeight != 105 || squat.TotalReps != 10 || squat.TotalVolume != 1025 {
		t.Errorf("latest totals wrong: %+v", squat)
	}
	// previous: max 100, 9 reps, volume 5*95 + 4*100 = 875
	if squat.WeightChange != 5 || squat.RepsChange != 1 || squat.VolumeChange != 150 {
		t.Errorf("changes wrong: %+v", squat)
	}

	curl := got[1]
	if curl.WeightChange != curl.MaxWeight || curl.RepsChange != curl.TotalReps || curl.VolumeChange != curl.TotalVolume {
		t.Errorf("an exercise missing from the previous workout should show its own totals: %+v", curl)
	}
}
