package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"fitness-tracker/internal/api"
	"fitness-tracker/internal/config"
	"fitness-tracker/internal/database"
	"fitness-tracker/internal/middleware"
	"fitness-tracker/internal/routes"
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

// apiBaseURL is where the UI reaches the JSON API served by this same process.
// If BIND_ADDR restricts the listener to one interface, loopback wouldn't work, so use it.
func apiBaseURL() string {
	host := config.AppConfig.BindAddr
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	return "http://" + host + ":" + config.AppConfig.Port + "/api"
}

func main() {
	log.Println("Loading app config...")
	config.LoadConfig()

	log.Println("Initialising database connection...")
	database.InitMongo()

	log.Println("Registering routes and multiplexer...")
	mux := http.NewServeMux()
	routes.RegisterRoutes(mux)

	if err := web.Register(mux, api.NewClient(apiBaseURL()), sessionFile()); err != nil {
		log.Fatal(err)
	}

	addr := config.AppConfig.BindAddr + ":" + config.AppConfig.Port
	log.Printf("Server running on %s", addr)
	log.Fatal(http.ListenAndServe(addr, middleware.BasicAuth(config.AppConfig.AuthUser, config.AppConfig.AuthPassword, mux)))
}
