// Package web serves the HTML/HTMX UI.
package web

import (
	"log"
	"net/http"
	"strconv"

	"fitness-tracker/internal/api"
	"fitness-tracker/internal/live"
	"fitness-tracker/internal/ui"
)

// Register mounts the UI (pages and static assets) on mux. client is how the UI talks to
// the JSON API; sessionFile is where the live workout session is kept.
func Register(mux *http.ServeMux, client *api.Client, sessionFile string) error {
	tpl, err := ui.ParseTemplates("home.html", "profile.html", "exercise_manage.html", "exercise_create.html", "routine_manage.html", "routine_create.html", "workout.html", "post_workout.html", "history.html")
	if err != nil {
		return err
	}

	mux.Handle("GET /static/", ui.StaticHandler())
	registerExercises(mux, tpl, client)
	registerRoutines(mux, tpl, client)
	registerWorkout(mux, tpl, client, live.New(client, sessionFile))
	registerAnalytics(mux, tpl, client)

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		data := ui.PageData{Title: "Home"}
		user, err := client.Me()
		if err != nil {
			log.Println(err)
			data.Error = backendDown
		} else {
			data.Username = user.Username
		}
		tpl.Render(w, r, "home.html", data)
	})

	showProfile := func(w http.ResponseWriter, r *http.Request, success string) {
		data := ui.PageData{Title: "Profile Management.", Success: success}
		user, err := client.Me()
		if err != nil {
			log.Println(err)
			data.Error = backendDown
		}
		data.User = user
		tpl.Render(w, r, "profile.html", data)
	}
	mux.HandleFunc("GET /profile", func(w http.ResponseWriter, r *http.Request) {
		showProfile(w, r, "")
	})
	mux.HandleFunc("POST /profile", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		height, _ := strconv.ParseFloat(r.FormValue("height"), 64)
		weight, _ := strconv.ParseFloat(r.FormValue("weight"), 64)
		userID := r.FormValue("user_id")
		updated := api.User{
			ID:             userID,
			Username:       r.FormValue("username"),
			Gender:         r.FormValue("gender"),
			DateOfBirth:    r.FormValue("date_of_birth"),
			Height:         height,
			Weight:         weight,
			UnitPreference: r.FormValue("unit_preference"),
		}
		if err := client.UpdateUser(userID, updated); err != nil {
			log.Println(err)
			tpl.Render(w, r, "profile.html", ui.PageData{Title: "Profile Management.", Error: "Couldn't update profile...", User: &updated})
			return
		}
		showProfile(w, r, "Updated user profile!")
	})
	return nil
}
