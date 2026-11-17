package service

import (
	"sort"

	"fitness-tracker/internal/config"
	"fitness-tracker/internal/database"
	"fitness-tracker/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UpdateExerciseHistory applies workout results into exercise history.
func UpdateExerciseHistory(userID, workoutID string) error {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	workoutObjID, err := primitive.ObjectIDFromHex(workoutID)
	if err != nil {
		return err
	}

	workoutData, err := database.GetWorkoutData(userObjID, workoutObjID)
	if err != nil {
		return err
	}

	var warmupID primitive.ObjectID
	var cooldownID primitive.ObjectID
	warmupValid := false
	cooldownValid := false
	if oid, err := primitive.ObjectIDFromHex(config.AppConfig.StaticExercises.WarmupID); err == nil {
		warmupID = oid
		warmupValid = true
	}
	if oid, err := primitive.ObjectIDFromHex(config.AppConfig.StaticExercises.CooldownID); err == nil {
		cooldownID = oid
		cooldownValid = true
	}

	for _, exercise := range workoutData.Exercises {
		// Skip warm-up/cool-down only if IDs are valid
		if (warmupValid && exercise.ExerciseID == warmupID) || (cooldownValid && exercise.ExerciseID == cooldownID) {
			continue
		}
		if len(exercise.Sets) == 0 {
			continue
		}
		valid := false
		for _, s := range exercise.Sets {
			if s.Reps > 0 {
				valid = true
				break
			}
		}
		if !valid {
			continue
		}

		exSets := models.ExerciseSets{
			Date:        workoutData.WorkoutDate,
			Equipment:   exercise.Equipment,
			Variation:   exercise.Variation,
			WorkoutSets: exercise.Sets,
		}

		updates := bson.M{
			"$push": bson.M{
				"exerciseSets": exSets,
			},
			"$setOnInsert": bson.M{
				"userID":     userObjID,
				"exerciseID": exercise.ExerciseID,
			},
		}

		if err := database.UpdateExerciseHistory(exercise.ExerciseID, userObjID, updates); err != nil {
			return err
		}
	}

	return nil
}

// DailyPoint is one day of an exercise's history: the heaviest weight and total volume.
type DailyPoint struct {
	Date   string  `json:"date"`
	Weight float64 `json:"weight,omitempty"`
	Volume float64 `json:"volume,omitempty"`
}

// SummarizeHistory groups an exercise's history by date (oldest first), ignoring sets
// without reps or weight. It returns a non-nil empty slice when nothing qualifies.
func SummarizeHistory(h models.ExerciseHistory) []DailyPoint {
	type day struct{ maxW, volume float64 }
	daily := make(map[string]day)
	for _, d := range h.Sets {
		date := d.Date.Time().Format("2006-01-02")
		for _, s := range d.WorkoutSets {
			if s.Reps > 0 && s.Weight > 0 {
				e := daily[date]
				if s.Weight > e.maxW {
					e.maxW = s.Weight
				}
				e.volume += s.Weight * float64(s.Reps)
				daily[date] = e
			}
		}
	}

	dates := make([]string, 0, len(daily))
	for date := range daily {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	points := make([]DailyPoint, 0, len(dates))
	for _, date := range dates {
		points = append(points, DailyPoint{Date: date, Weight: daily[date].maxW, Volume: daily[date].volume})
	}
	return points
}
