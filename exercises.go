package main

import (
	"log"
	"net/http"
	"strings"

	"fitness/internal/api"
	"fitness/internal/ui"
)

// splitLines mirrors the Streamlit parsing: one trimmed, non-empty item per line.
func splitLines(s string) []string {
	items := []string{}
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			items = append(items, line)
		}
	}
	return items
}

func exerciseFromForm(r *http.Request) api.Exercise {
	return api.Exercise{
		ID:         r.FormValue("id"),
		Name:       strings.TrimSpace(r.FormValue("name")),
		Category:   r.FormValue("category"),
		Variations: splitLines(r.FormValue("variations")),
		Equipment:  splitLines(r.FormValue("equipment")),
	}
}

func registerExercises(mux *http.ServeMux, tpl ui.Templates, client *api.Client) {
	manage := func(w http.ResponseWriter, r *http.Request, selected, success, errMsg string) {
		data := ui.PageData{Title: "Exercise Editor", Success: success, Error: errMsg, Categories: api.ExerciseCategories}
		list, err := client.ExerciseList()
		if err != nil {
			log.Println(err)
			list = nil
		}
		data.Exercises = list
		for i := range list {
			if list[i].Name == selected {
				data.Exercise = &list[i]
			}
		}
		tpl.Render(w, r, "exercise_manage.html", data)
	}

	mux.HandleFunc("GET /exercises/manage", func(w http.ResponseWriter, r *http.Request) {
		manage(w, r, r.URL.Query().Get("exercise"), "", "")
	})

	mux.HandleFunc("POST /exercises/manage", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		e := exerciseFromForm(r)
		if err := client.UpdateExercise(e); err != nil {
			log.Println(err)
			manage(w, r, e.Name, "", "Failed to update exercise. Please try again.")
			return
		}
		manage(w, r, e.Name, "Exercise '"+e.Name+"' updated successfully!", "")
	})

	create := func(w http.ResponseWriter, r *http.Request, draft *api.Exercise, errMsg string) {
		tpl.Render(w, r, "exercise_create.html", ui.PageData{
			Title: "Exercise Creator", Error: errMsg, Exercise: draft, Categories: api.ExerciseCategories,
		})
	}

	mux.HandleFunc("GET /exercises/create", func(w http.ResponseWriter, r *http.Request) {
		create(w, r, nil, "")
	})

	mux.HandleFunc("POST /exercises/create", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		e := exerciseFromForm(r)
		if e.Name == "" {
			create(w, r, &e, "Exercise name is required!")
			return
		}
		if list, err := client.ExerciseList(); err == nil {
			for _, existing := range list {
				if existing.Name == e.Name {
					create(w, r, &e, "Exercise name already exists! Please choose a different name.")
					return
				}
			}
		}
		id, err := client.CreateExercise(e)
		if err != nil {
			log.Println(err)
			create(w, r, &e, "Failed to create exercise. Please try again.")
			return
		}
		// Like Streamlit, land on the editor after creating.
		w.Header().Set("HX-Push-Url", "/exercises/manage")
		manage(w, r, "", "Exercise '"+e.Name+"' created successfully! ID: "+id, "")
	})
}
