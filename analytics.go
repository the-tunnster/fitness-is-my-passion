package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"fitness/internal/api"
	"fitness/internal/ui"
)

// metric formats a stat the way Streamlit's st.metric does: the delta is shown to two
// decimals, coloured by sign, and hidden when there is no change.
func metric(label string, value, change float64) ui.Metric {
	m := ui.Metric{Label: label, Value: fmt.Sprint(value)}
	switch {
	case change > 0:
		m.Delta, m.Dir = fmt.Sprintf("%.2f", change), "up"
	case change < 0:
		m.Delta, m.Dir = fmt.Sprintf("%.2f", -change), "down"
	}
	return m
}

func registerAnalytics(mux *http.ServeMux, tpl ui.Templates, client *api.Client) {
	mux.HandleFunc("GET /workout/analytics", func(w http.ResponseWriter, r *http.Request) {
		data := ui.PageData{Title: "Post Workout Analytics", Mock: client.Mock, Selected: r.URL.Query().Get("routine")}
		userID, err := client.UserID()
		if err != nil {
			log.Println(err)
			data.Error = backendDown
			tpl.Render(w, r, "post_workout.html", data)
			return
		}
		routines, _ := client.RoutineList(userID)
		if len(routines) == 0 {
			data.Info = "You don't seem to have any routines set up. Please create one to access it here."
			tpl.Render(w, r, "post_workout.html", data)
			return
		}
		data.Routines = routines

		if data.Selected != "" {
			entries, err := client.WorkoutComparison(userID, data.Selected)
			if err != nil {
				log.Println(err)
			}
			if len(entries) == 0 {
				data.Notice = "There aren't enough workouts saved for a comparison."
			}
			for _, e := range entries {
				data.Metrics = append(data.Metrics, ui.MetricGroup{
					Title: e.ExerciseName + ", " + e.Variation,
					Metrics: []ui.Metric{
						metric("Max Weight", e.MaxWeight, e.WeightChange),
						metric("Total Reps", e.Reps, e.RepsChange),
						metric("Volume Moved", e.Volume, e.VolumeChange),
					},
				})
			}
		}
		tpl.Render(w, r, "post_workout.html", data)
	})

	mux.HandleFunc("GET /history", func(w http.ResponseWriter, r *http.Request) {
		data := ui.PageData{Title: "Historical Workout Analytics", Mock: client.Mock, Selected: r.URL.Query().Get("exercise")}
		userID, err := client.UserID()
		if err != nil {
			log.Println(err)
			data.Error = backendDown
			tpl.Render(w, r, "history.html", data)
			return
		}
		exercises, _ := client.ExerciseList()
		if len(exercises) == 0 {
			data.Info = "No exercises found in your database. Please add exercises first."
			tpl.Render(w, r, "history.html", data)
			return
		}
		data.Exercises = exercises

		var name string
		for _, e := range exercises {
			if e.ID == data.Selected {
				name = e.Name
			}
		}
		if name != "" {
			points, err := client.ExerciseHistory(userID, data.Selected)
			if err != nil {
				log.Println(err)
			}
			switch {
			case len(points) == 0:
				data.Notice = fmt.Sprintf("No workout history found for '%s'.", name)
			case len(points) == 1:
				data.Notice = fmt.Sprintf("Not enough valid data found for '%s'.", name)
			default:
				b, _ := json.Marshal(points)
				data.ChartJSON = string(b)
			}
		}
		tpl.Render(w, r, "history.html", data)
	})
}
