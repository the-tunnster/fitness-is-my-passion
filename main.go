package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"fitness/internal/api"
	"fitness/internal/live"
	"fitness/internal/ui"
)

// sessionFile is where the live workout session is kept; mount it as a volume when
// running in Docker so an in-progress workout survives a restart.
func sessionFile() string {
	if p := os.Getenv("SESSION_FILE"); p != "" {
		return p
	}
	return filepath.Join("data", "live-session.json")
}

func main() {
	tpl, err := ui.ParseTemplates("home.html", "profile.html", "exercise_manage.html", "exercise_create.html", "routine_manage.html", "routine_create.html", "workout.html", "post_workout.html", "history.html")
	if err != nil {
		log.Fatal(err)
	}
	client := api.NewClient()

	mux := http.NewServeMux()
	mux.Handle("GET /static/", ui.StaticHandler())
	registerExercises(mux, tpl, client)
	registerRoutines(mux, tpl, client)
	registerWorkout(mux, tpl, client, live.New(client, sessionFile()))
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
			data.Error = "Couldn't reach the backend."
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
		err := client.UpdateUser(userID, api.User{
			Username:       r.FormValue("username"),
			Gender:         r.FormValue("gender"),
			DateOfBirth:    r.FormValue("date_of_birth"),
			Height:         height,
			Weight:         weight,
			UnitPreference: r.FormValue("unit_preference"),
		})
		if err != nil {
			log.Println(err)
			data := ui.PageData{Title: "Profile Management.", Error: "Couldn't update profile..."}
			data.User = &api.User{
				ID: userID, Username: r.FormValue("username"), Gender: r.FormValue("gender"),
				DateOfBirth: r.FormValue("date_of_birth"), Height: height, Weight: weight,
				UnitPreference: r.FormValue("unit_preference"),
			}
			tpl.Render(w, r, "profile.html", data)
			return
		}
		showProfile(w, r, "Updated user profile!")
	})

	addr := ":" + os.Getenv("PORT")
	if addr == ":" {
		addr = ":8080"
	}
	log.Println("listening on " + addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
