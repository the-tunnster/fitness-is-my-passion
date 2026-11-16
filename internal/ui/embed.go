package ui

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"fitness/internal/api"
)

//go:embed templates static
var assets embed.FS

// Page describes one entry in the navigation sidebar.
type Page struct {
	Path  string
	Title string
	Icon  string
}

// NavGroup mirrors the sidebar sections of the Streamlit app.
type NavGroup struct {
	Name  string
	Pages []Page
}

var Navigation = []NavGroup{
	{"", []Page{{"/", "Home", "fitness_center"}}},
	{"Exercises", []Page{
		{"/exercises/manage", "Manage", "construction"},
		{"/exercises/create", "Create", "library_add"},
	}},
	{"Routines", []Page{
		{"/routines/manage", "Manage", "construction"},
		{"/routines/create", "Create", "library_add"},
	}},
	{"Strength Training", []Page{
		{"/workout", "Record a Session", "exercise"},
		{"/workout/analytics", "Workout Analytics", "candlestick_chart"},
		{"/history", "Historical Analytics", "trending_up"},
	}},
	{"User Account", []Page{{"/profile", "Management", "account_box"}}},
}

// PageData is passed to every template.
type PageData struct {
	Title    string
	Path     string
	Nav      []NavGroup
	Username string
	User     *api.User
	Error    string
	Success  string

	Exercises  []api.Exercise
	Exercise   *api.Exercise
	Categories []string

	Routines      []api.Routine
	RoutineID     string
	RoutineName   string
	Rows          []RoutineRow
	ExerciseNames []string
	Action        string
	Info          string
	Warning       string
	Notice        string
	Mock          bool

	Selected   string
	Metrics    []MetricGroup
	ChartJSON  string

	Workout *WorkoutView
}

// Metric is one stat card; Dir is "up", "down" or empty when there is no change.
type Metric struct {
	Label string
	Value string
	Delta string
	Dir   string
}

// MetricGroup is the set of cards shown for one exercise.
type MetricGroup struct {
	Title   string
	Metrics []Metric
}

// WorkoutOption is one entry of the exercise selector on the recorder page.
type WorkoutOption struct {
	Index    int
	Label    string
	Selected bool
}

// WorkoutView is the recorder page state; Active is false while choosing a routine.
type WorkoutView struct {
	Active   bool
	Routines []api.Routine

	Options       []WorkoutOption
	Index         int
	Exercise      *api.SessionExercise
	Variations    []string
	Equipment     []string
	AllExercises  []string
}

// RoutineRow is one editable exercise line in the routine editor.
type RoutineRow struct {
	Name string
	Sets int
	Reps int
}

var funcs = template.FuncMap{
	"inc": func(i int) int { return i + 1 },
	"rowData": func(names []string, row RoutineRow) map[string]any {
		return map[string]any{"Names": names, "Row": row}
	},
	"newRow": func(names []string) RoutineRow {
		return RoutineRow{Name: names[0], Sets: 3, Reps: 10}
	},
	"join": func(items []string) string { return strings.Join(items, "\n") },
	// Textarea height grows with its content, with a minimum of three lines.
	"rows": func(items []string) int { return max(len(items)+1, 3) },
	// The backend returns RFC 3339 timestamps; date inputs need YYYY-MM-DD.
	"dateOnly": func(s string) string {
		if i := strings.IndexAny(s, "T "); i >= 0 {
			return s[:i]
		}
		return s
	},
}

// Templates holds one parsed template set per page, each combined with base.html.
type Templates map[string]*template.Template

// ParseTemplates parses every page template together with base.html on startup.
func ParseTemplates(pages ...string) (Templates, error) {
	set := Templates{}
	for _, p := range pages {
		t, err := template.New(p).Funcs(funcs).ParseFS(assets, "templates/base.html", "templates/partial_*.html", "templates/"+p)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", p, err)
		}
		set[p] = t
	}
	return set, nil
}

// Render writes the full page for normal requests and only the content
// partial for HTMX requests.
func (t Templates) Render(w http.ResponseWriter, r *http.Request, page string, data PageData) {
	tpl, ok := t[page]
	if !ok {
		http.Error(w, "unknown template", http.StatusInternalServerError)
		return
	}
	data.Nav = Navigation
	data.Path = r.URL.Path
	name := "base"
	if r.Header.Get("HX-Request") == "true" {
		name = "content"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Add("Vary", "HX-Request")
	if err := tpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// StaticHandler serves the embedded static directory under /static/.
func StaticHandler() http.Handler {
	sub, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
}
