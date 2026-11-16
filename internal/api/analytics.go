package api

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// ComparisonEntry is one exercise of GET /workouts/comparison.
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

// HistoryPoint is one entry of GET /history/data. Volume may be missing.
type HistoryPoint struct {
	Date   string   `json:"date"`
	Weight float64  `json:"weight"`
	Volume *float64 `json:"volume,omitempty"`
}

// WorkoutComparison returns nil (and no error) when there aren't enough workouts to compare.
func (c *Client) WorkoutComparison(userID, routineID string) ([]ComparisonEntry, error) {
	if c.Mock {
		return c.mockComparison(routineID), nil
	}
	q := url.Values{"user_id": {userID}, "routine_id": {routineID}}
	resp, err := c.send(http.MethodGet, "/workouts/comparison", q, nil, http.StatusOK, http.StatusNotFound)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	var entries []ComparisonEntry
	return entries, json.NewDecoder(resp.Body).Decode(&entries)
}

func (c *Client) ExerciseHistory(userID, exerciseID string) ([]HistoryPoint, error) {
	if c.Mock {
		return mockHistory(exerciseID), nil
	}
	q := url.Values{"user_id": {userID}, "exercise_id": {exerciseID}}
	resp, err := c.send(http.MethodGet, "/history/data", q, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var points []HistoryPoint
	return points, json.NewDecoder(resp.Body).Decode(&points)
}
