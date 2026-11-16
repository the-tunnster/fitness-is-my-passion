package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"fitness-tracker/internal/api"
	"fitness-tracker/internal/routes"
	"fitness-tracker/internal/web"
)

// newMux wires the API and the UI the way main does, without needing a database.
func newMux(t *testing.T) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	routes.RegisterRoutes(mux)
	// Unreachable on purpose: these tests never need the API to answer.
	client := api.NewClient("http://127.0.0.1:1/api")
	if err := web.Register(mux, client, filepath.Join(t.TempDir(), "live.json")); err != nil {
		t.Fatal(err)
	}
	return mux
}

func do(mux *http.ServeMux, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

// Registering both route sets used to be impossible: the API's POST /routines/create
// collided with the UI's /routines/create form. The API now lives under /api.
func TestUIAndAPIRoutesDoNotCollide(t *testing.T) {
	mux := newMux(t)

	if rec := do(mux, http.MethodGet, "/routines/create"); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), "Routine Creation") {
		t.Fatalf("UI page not served: %d", rec.Code)
	}
	if rec := do(mux, http.MethodGet, "/static/style.css"); rec.Code != http.StatusOK {
		t.Fatalf("static assets not served: %d", rec.Code)
	}

	// The API answers under /api with its own method rules...
	rec := do(mux, http.MethodGet, "/api/routines/create")
	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "POST" {
		t.Fatalf("API route not mounted under /api: %d allow=%q", rec.Code, rec.Header().Get("Allow"))
	}
	if rec := do(mux, http.MethodPost, "/api/me"); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("/api/me should only allow GET, got %d", rec.Code)
	}
	if rec := do(mux, http.MethodGet, "/api/does-not-exist"); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown API path: %d", rec.Code)
	}

	// ...and is no longer reachable at the old root paths.
	if rec := do(mux, http.MethodGet, "/me"); rec.Code != http.StatusNotFound {
		t.Fatalf("/me should have moved to /api/me, got %d", rec.Code)
	}
}
