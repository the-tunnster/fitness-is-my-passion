package service

import (
	"reflect"
	"testing"
	"time"

	"fitness-tracker/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func day(d int, sets ...models.WorkoutSet) models.ExerciseSets {
	return models.ExerciseSets{
		Date:        primitive.NewDateTimeFromTime(time.Date(2026, 1, d, 12, 0, 0, 0, time.UTC)),
		WorkoutSets: sets,
	}
}

func TestSummarizeHistoryGroupsByDayAndSkipsEmptySets(t *testing.T) {
	h := models.ExerciseHistory{Sets: []models.ExerciseSets{
		day(5, models.WorkoutSet{Reps: 5, Weight: 100}, models.WorkoutSet{Reps: 3, Weight: 110}),
		day(2, models.WorkoutSet{Reps: 10, Weight: 50}, models.WorkoutSet{Reps: 0, Weight: 200}),
		day(9, models.WorkoutSet{Reps: 8, Weight: 0}),
	}}
	want := []DailyPoint{
		{Date: "2026-01-02", Weight: 50, Volume: 500},
		{Date: "2026-01-05", Weight: 110, Volume: 830},
	}
	if got := SummarizeHistory(h); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if got := SummarizeHistory(models.ExerciseHistory{}); got == nil || len(got) != 0 {
		t.Fatalf("empty history should give an empty, non-nil slice, got %#v", got)
	}
}
