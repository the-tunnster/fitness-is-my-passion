package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"fitness-tracker/internal/api"
	"fitness-tracker/internal/web"
)

// noDatabase is a Backend whose calls the pages make all fail, standing in for the missing database.
type noDatabase struct{ api.Backend }

var errNoDatabase = errors.New("no database")

func (noDatabase) UserID() (string, error)                   { return "", errNoDatabase }
func (noDatabase) Me() (*api.User, error)                    { return nil, errNoDatabase }
func (noDatabase) ExerciseList() ([]api.Exercise, error)     { return nil, errNoDatabase }
func (noDatabase) RoutineList(string) ([]api.Routine, error) { return nil, errNoDatabase }

// newMux wires the UI the way main does, without needing a database.
func newMux(t *testing.T) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	if err := web.Register(mux, noDatabase{}, filepath.Join(t.TempDir(), "live.json")); err != nil {
		t.Fatal(err)
	}
	return mux
}

func do(mux *http.ServeMux, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func TestUIPagesAndStaticAssetsAreServed(t *testing.T) {
	mux := newMux(t)

	if rec := do(mux, http.MethodGet, "/routines/create"); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), "Routine Creation") {
		t.Fatalf("UI page not served: %d", rec.Code)
	}
	if rec := do(mux, http.MethodGet, "/static/style.css"); rec.Code != http.StatusOK {
		t.Fatalf("static assets not served: %d", rec.Code)
	}
	if rec := do(mux, http.MethodGet, "/api/me"); rec.Code != http.StatusNotFound {
		t.Fatalf("the JSON API is gone, /api/me should be 404, got %d", rec.Code)
	}
}
