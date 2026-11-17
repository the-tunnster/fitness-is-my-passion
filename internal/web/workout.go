package web

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"fitness-tracker/internal/api"
	"fitness-tracker/internal/live"
	"fitness-tracker/internal/ui"
)

type workoutMsg struct{ Success, Error, Warning string }

const savedLocally = " Your changes are saved locally and will sync on your next action."

// applyExerciseForm copies the posted variation, equipment and sets onto the exercise.
func applyExerciseForm(ex *api.SessionExercise, r *http.Request) {
	if v := r.FormValue("variation"); v != "" {
		ex.Variation = v
	}
	if v := r.FormValue("equipment"); v != "" {
		ex.Equipment = v
	}
	reps, weights := r.Form["reps"], r.Form["weight"]
	if len(reps) == 0 || len(reps) != len(weights) {
		return
	}
	sets := make([]api.WorkoutSet, 0, len(reps))
	for i := range reps {
		n, _ := strconv.Atoi(reps[i])
		w, _ := strconv.ParseFloat(weights[i], 64)
		sets = append(sets, api.WorkoutSet{Reps: max(n, 0), Weight: max(w, 0)})
	}
	ex.Sets = sets
}

func registerWorkout(mux *http.ServeMux, tpl ui.Templates, client api.Backend, sessions *live.Manager) {
	// render shows the recorder from the live session (local file; the backend is only
	// consulted when no file exists).
	render := func(w http.ResponseWriter, r *http.Request, m workoutMsg) {
		data := ui.PageData{Title: "Workout Recorder", Success: m.Success, Error: m.Error, Warning: m.Warning}
		userID, err := client.UserID()
		var st *live.State
		if err == nil {
			st, err = sessions.Current(userID)
		}
		if err != nil {
			log.Println(err)
			data.Error = backendDown
			tpl.Render(w, r, "workout.html", data)
			return
		}

		if st == nil {
			routines, _ := client.RoutineList(userID)
			if len(routines) == 0 {
				data.Info = "You don't seem to have any routines set up. Please create one to access it here."
			} else {
				data.Workout = &ui.WorkoutView{Routines: routines}
			}
			tpl.Render(w, r, "workout.html", data)
			return
		}
		if st.Dirty && data.Error == "" && data.Warning == "" {
			data.Warning = "Some changes haven't synced to the backend yet. They will sync on your next action."
		}

		idx := st.ExerciseIndex
		if idx < 0 || idx >= len(st.Exercises) {
			idx = 0
		}
		view := &ui.WorkoutView{Active: true, Index: idx}
		for i, ex := range st.Exercises {
			view.Options = append(view.Options, ui.WorkoutOption{Index: i, Label: ex.DisplayName(), Selected: i == idx})
		}
		if all, err := client.ExerciseList(); err == nil {
			for _, e := range all {
				view.AllExercises = append(view.AllExercises, e.Name)
			}
		}
		if idx < len(st.Exercises) {
			ex := &st.Exercises[idx]
			view.Exercise = ex
			view.Variations, view.Equipment = []string{"None"}, []string{"None"}
			if meta, err := client.ExerciseData(ex.ExerciseID); err == nil {
				view.Variations, view.Equipment = meta.Variations, meta.Equipment
			}
		}
		data.Workout = view
		tpl.Render(w, r, "workout.html", data)
	}

	// update applies a change to the live session and reports a failure message for
	// the user. It returns false when the response has already been sent.
	update := func(w http.ResponseWriter, r *http.Request, fn func(*api.Session), failure string) (workoutMsg, bool) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return workoutMsg{}, false
		}
		userID, err := client.UserID()
		if err != nil {
			log.Println(err)
			render(w, r, workoutMsg{Error: backendDown})
			return workoutMsg{}, false
		}
		if _, err := sessions.Update(userID, fn); err != nil {
			log.Println(err)
			if errors.Is(err, live.ErrNoSession) {
				render(w, r, workoutMsg{Error: "No active workout session."})
				return workoutMsg{}, false
			}
			return workoutMsg{Error: failure + savedLocally}, true
		}
		return workoutMsg{}, true
	}

	mux.HandleFunc("GET /workout", func(w http.ResponseWriter, r *http.Request) {
		render(w, r, workoutMsg{})
	})

	mux.HandleFunc("POST /workout/start", func(w http.ResponseWriter, r *http.Request) {
		routineID := r.FormValue("routine_id")
		if routineID == "" {
			render(w, r, workoutMsg{Warning: "Please select a routine to start."})
			return
		}
		userID, err := client.UserID()
		if err == nil {
			_, err = sessions.Start(userID, routineID)
		}
		if err != nil {
			log.Println(err)
			render(w, r, workoutMsg{Error: "Failed to start workout session. Please try again."})
			return
		}
		render(w, r, workoutMsg{})
	})

	mux.HandleFunc("POST /workout/select", func(w http.ResponseWriter, r *http.Request) {
		idx, convErr := strconv.Atoi(r.FormValue("index"))
		msg, ok := update(w, r, func(s *api.Session) {
			if convErr == nil && idx >= 0 && idx < len(s.Exercises) {
				s.ExerciseIndex = idx
			}
		}, "Failed to update session information.")
		if ok {
			render(w, r, msg)
		}
	})

	mux.HandleFunc("POST /workout/add", func(w http.ResponseWriter, r *http.Request) {
		all, _ := client.ExerciseList()
		var chosen *api.Exercise
		for i := range all {
			if all[i].Name == r.FormValue("exercise") {
				chosen = &all[i]
			}
		}
		msg, ok := update(w, r, func(s *api.Session) {
			if chosen == nil {
				return
			}
			s.Exercises = append(s.Exercises, api.SessionExercise{
				ExerciseID: chosen.ID, Name: chosen.Name, Variation: "None", Equipment: "None",
				Sets: []api.WorkoutSet{{Reps: 8}, {Reps: 8}, {Reps: 8}},
			})
			s.ExerciseIndex = len(s.Exercises) - 1
		}, "Failed to update session information.")
		if ok {
			render(w, r, msg)
		}
	})

	mux.HandleFunc("POST /workout/set", func(w http.ResponseWriter, r *http.Request) {
		action, failure, success := r.FormValue("action"), "Failed to update session information.", ""
		switch action {
		case "add", "drop":
		case "update":
			failure, success = "Failed to update exercise information.", "Exercise updated."
		default:
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		posted, convErr := strconv.Atoi(r.FormValue("index"))

		msg, ok := update(w, r, func(s *api.Session) {
			idx := s.ExerciseIndex
			if convErr == nil && posted >= 0 && posted < len(s.Exercises) {
				idx = posted
			}
			if idx < 0 || idx >= len(s.Exercises) {
				return
			}
			s.ExerciseIndex = idx
			ex := &s.Exercises[idx]
			applyExerciseForm(ex, r)
			switch action {
			case "add":
				reps := 8
				if len(ex.Sets) > 0 {
					reps = ex.Sets[0].Reps
				}
				ex.Sets = append(ex.Sets, api.WorkoutSet{Reps: reps})
			case "drop":
				if len(ex.Sets) > 0 {
					ex.Sets = ex.Sets[:len(ex.Sets)-1]
				}
			}
		}, failure)
		if !ok {
			return
		}
		if msg.Error == "" {
			msg.Success = success
		}
		render(w, r, msg)
	})

	mux.HandleFunc("POST /workout/finish", func(w http.ResponseWriter, r *http.Request) {
		userID, err := client.UserID()
		if err != nil {
			log.Println(err)
			render(w, r, workoutMsg{Error: backendDown})
			return
		}
		res, err := sessions.Finish(userID)
		switch {
		case errors.Is(err, live.ErrNoSession):
			render(w, r, workoutMsg{Error: "No active workout session."})
			return
		case errors.Is(err, live.ErrNotSynced):
			log.Println(err)
			render(w, r, workoutMsg{Error: "Couldn't sync your session to the backend. Your changes are saved locally; please try again."})
			return
		case errors.Is(err, live.ErrCreateWorkout):
			log.Println(err)
			render(w, r, workoutMsg{Error: "Couldn't save workout."})
			return
		case err != nil:
			log.Println(err)
			render(w, r, workoutMsg{Error: "Couldn't clear session data."})
			return
		}
		if res.HistoryErr != nil {
			log.Println(res.HistoryErr)
		}
		// Like Streamlit, continue on to the post-workout analytics page.
		w.Header().Set("HX-Location", `{"path":"/workout/analytics","target":"#content"}`)
	})

	mux.HandleFunc("POST /workout/cancel", func(w http.ResponseWriter, r *http.Request) {
		userID, err := client.UserID()
		if err == nil {
			err = sessions.Cancel(userID)
		}
		switch {
		case errors.Is(err, live.ErrNoSession):
			render(w, r, workoutMsg{Error: "No active workout session."})
		case err != nil:
			log.Println(err)
			render(w, r, workoutMsg{Error: "Couldn't clear session data."})
		default:
			render(w, r, workoutMsg{Success: "Workout cancelled. All progress discarded!"})
		}
	})
}
