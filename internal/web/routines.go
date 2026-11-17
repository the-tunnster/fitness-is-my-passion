package web

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"fitness-tracker/internal/api"
	"fitness-tracker/internal/ui"
)

const backendDown = "Couldn't reach the backend."

// rowsFromForm rebuilds the editor rows from the posted form, in order.
func rowsFromForm(r *http.Request) []ui.RoutineRow {
	names, sets, reps := r.Form["ex_name"], r.Form["ex_sets"], r.Form["ex_reps"]
	rows := make([]ui.RoutineRow, 0, len(names))
	for i, name := range names {
		row := ui.RoutineRow{Name: name, Sets: 1, Reps: 1}
		if i < len(sets) {
			if n, err := strconv.Atoi(sets[i]); err == nil && n >= 1 {
				row.Sets = n
			}
		}
		if i < len(reps) {
			if n, err := strconv.Atoi(reps[i]); err == nil && n >= 1 {
				row.Reps = n
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// toRoutineExercises resolves exercise names to ids using the exercise list.
func toRoutineExercises(rows []ui.RoutineRow, exercises []api.Exercise) []api.RoutineExercise {
	ids := map[string]string{}
	for _, e := range exercises {
		ids[e.Name] = e.ID
	}
	out := make([]api.RoutineExercise, 0, len(rows))
	for _, row := range rows {
		out = append(out, api.RoutineExercise{ExerciseID: ids[row.Name], Name: row.Name, TargetSets: row.Sets, TargetReps: row.Reps})
	}
	return out
}

func exerciseNames(exercises []api.Exercise) []string {
	names := make([]string, 0, len(exercises))
	for _, e := range exercises {
		names = append(names, e.Name)
	}
	if len(names) == 0 {
		names = []string{"None"}
	}
	return names
}

func registerRoutines(mux *http.ServeMux, tpl ui.Templates, client api.Backend) {
	type manageView struct {
		ID      string
		Rows    []ui.RoutineRow // nil means load from the backend
		Success string
		Error   string
	}

	manage := func(w http.ResponseWriter, r *http.Request, v manageView) {
		data := ui.PageData{Title: "Routine Management", Action: "/routines/manage", Success: v.Success, Error: v.Error}
		userID, err := client.UserID()
		if err != nil {
			log.Println(err)
			data.Error = backendDown
			tpl.Render(w, r, "routine_manage.html", data)
			return
		}
		routines, _ := client.RoutineList(userID)
		if len(routines) == 0 {
			data.Info = "No routines found."
			tpl.Render(w, r, "routine_manage.html", data)
			return
		}
		data.Routines = routines
		for _, rt := range routines {
			if rt.ID == v.ID {
				data.RoutineID = v.ID
			}
		}
		if data.RoutineID != "" {
			exercises, _ := client.ExerciseList()
			data.ExerciseNames = exerciseNames(exercises)
			data.Rows = v.Rows
			if data.Rows == nil {
				full, err := client.RoutineData(userID, data.RoutineID)
				if err != nil {
					log.Println(err)
					data.Error = "Could not load routine."
				} else {
					for _, ex := range full.Exercises {
						data.Rows = append(data.Rows, ui.RoutineRow{Name: ex.Name, Sets: ex.TargetSets, Reps: ex.TargetReps})
					}
				}
			}
		}
		tpl.Render(w, r, "routine_manage.html", data)
	}

	mux.HandleFunc("GET /routines/manage", func(w http.ResponseWriter, r *http.Request) {
		manage(w, r, manageView{ID: r.URL.Query().Get("routine")})
	})

	mux.HandleFunc("POST /routines/manage", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id, rows := r.FormValue("routine_id"), rowsFromForm(r)
		exercises, _ := client.ExerciseList()
		userID, err := client.UserID()
		if err == nil {
			err = client.UpdateRoutine(userID, id, toRoutineExercises(rows, exercises))
		}
		if err != nil {
			log.Println(err)
			manage(w, r, manageView{ID: id, Rows: rows, Error: "Save failed."})
			return
		}
		manage(w, r, manageView{ID: id, Success: "Saved!"})
	})

	mux.HandleFunc("POST /routines/delete", func(w http.ResponseWriter, r *http.Request) {
		id := r.FormValue("routine_id")
		userID, err := client.UserID()
		if err == nil {
			err = client.DeleteRoutine(userID, id)
		}
		if err != nil {
			log.Println(err)
			manage(w, r, manageView{ID: id, Error: "That didn't work."})
			return
		}
		manage(w, r, manageView{Success: "Deleted Routine"})
	})

	create := func(w http.ResponseWriter, r *http.Request, name string, rows []ui.RoutineRow, errMsg string) {
		exercises, _ := client.ExerciseList()
		tpl.Render(w, r, "routine_create.html", ui.PageData{
			Title: "Routine Creation", Action: "/routines/create", Error: errMsg,
			RoutineName: name, Rows: rows, ExerciseNames: exerciseNames(exercises),
		})
	}

	mux.HandleFunc("GET /routines/create", func(w http.ResponseWriter, r *http.Request) {
		create(w, r, "", nil, "")
	})

	mux.HandleFunc("POST /routines/create", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		name, rows := r.FormValue("name"), rowsFromForm(r)
		exercises, _ := client.ExerciseList()
		switch {
		case strings.TrimSpace(name) == "":
			create(w, r, name, rows, "Routine name is required.")
			return
		case len(rows) == 0:
			create(w, r, name, rows, "Add at least one exercise.")
			return
		}
		userID, err := client.UserID()
		if err == nil {
			err = client.CreateRoutine(userID, name, toRoutineExercises(rows, exercises))
		}
		if err != nil {
			log.Println(err)
			create(w, r, name, rows, "Creation failed.")
			return
		}
		// Like Streamlit, land on the manager after creating.
		w.Header().Set("HX-Push-Url", "/routines/manage")
		manage(w, r, manageView{Success: "Routine Created!"})
	})
}
