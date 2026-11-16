package api

import (
	"hash/fnv"
	"math"
	"math/rand"
	"time"
)

// Generated data for previewing the analytics pages before real workouts exist.
// Enabled with MOCK_ANALYTICS=1; it is deterministic per routine/exercise id.

func seeded(id string) *rand.Rand {
	h := fnv.New64a()
	h.Write([]byte(id))
	return rand.New(rand.NewSource(int64(h.Sum64())))
}

func round(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}

func (c *Client) mockComparison(routineID string) []ComparisonEntry {
	rng := seeded(routineID)
	names := []string{"Squats", "Bench Press", "Deadlifts"}
	if userID, err := c.UserID(); err == nil {
		if rt, err := c.RoutineData(userID, routineID); err == nil && len(rt.Exercises) > 0 {
			names = names[:0]
			for _, ex := range rt.Exercises {
				names = append(names, ex.Name)
			}
		}
	}
	variations := []string{"Standard", "Wide Grip", "Close Grip", "Seated"}
	entries := make([]ComparisonEntry, 0, len(names))
	for i, name := range names {
		weight := float64(20 + rng.Intn(18)*5)
		reps := float64(24 + rng.Intn(20))
		entry := ComparisonEntry{
			ExerciseName: name,
			Variation:    variations[rng.Intn(len(variations))],
			MaxWeight:    weight,
			Reps:         reps,
			Volume:       round(weight*reps*0.8, 1),
		}
		// Mix of gains, losses and no change so every delta style shows up.
		switch i % 3 {
		case 0:
			entry.WeightChange, entry.RepsChange, entry.VolumeChange = 2.5, 3, round(entry.Volume*0.06, 2)
		case 1:
			entry.WeightChange, entry.RepsChange, entry.VolumeChange = 0, -2, -round(entry.Volume*0.03, 2)
		default:
			entry.WeightChange, entry.RepsChange, entry.VolumeChange = -5, 0, round(entry.Volume*0.01, 2)
		}
		entries = append(entries, entry)
	}
	return entries
}

func mockHistory(exerciseID string) []HistoryPoint {
	rng := seeded(exerciseID)
	const sessions = 18
	start := time.Now().AddDate(0, 0, -7*sessions)
	weight := float64(30 + rng.Intn(10)*5)
	points := make([]HistoryPoint, 0, sessions)
	for i := 0; i < sessions; i++ {
		// Progressive overload with the odd stalled or deload session.
		weight += float64(rng.Intn(3)) * 1.25
		if rng.Intn(10) == 0 {
			weight -= 5
		}
		date := start.AddDate(0, 0, 7*i+rng.Intn(3)).Format("2006-01-02")
		vol := round(weight*float64(24+rng.Intn(5)), 0)
		p := HistoryPoint{Date: date, Weight: round(weight, 2)}
		// One entry without volume, as older records can lack it.
		if i != 3 {
			p.Volume = &vol
		}
		points = append(points, p)
	}
	return points
}
