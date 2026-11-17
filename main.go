package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"fitness-tracker/internal/api"
	"fitness-tracker/internal/config"
	"fitness-tracker/internal/database"
	"fitness-tracker/internal/ui"
	"fitness-tracker/internal/web"
)

// sessionFile is where the live workout session is kept, so an in-progress workout
// survives a restart.
func sessionFile() string {
	if p := os.Getenv("SESSION_FILE"); p != "" {
		return p
	}
	return filepath.Join("data", "live-session.json")
}

func main() {
	log.Println("Loading app config...")
	config.LoadConfig()

	log.Println("Initialising database connection...")
	database.InitMongo()

	log.Println("Registering routes and multiplexer...")
	mux := http.NewServeMux()

	if err := web.Register(mux, api.NewDB(), sessionFile()); err != nil {
		log.Fatal(err)
	}

	addr := ":" + config.AppConfig.Port
	log.Printf("Server running on %s", addr)
	log.Fatal(http.ListenAndServe(addr, ui.Compress(mux)))
}
