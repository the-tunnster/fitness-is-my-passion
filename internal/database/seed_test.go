package database

import (
	"encoding/json"
	"testing"

	"fitness-tracker/internal/models"
)

func TestEmbeddedExerciseListIsValid(t *testing.T) {
	data, err := exercisesJSON("")
	if err != nil {
		t.Fatal(err)
	}
	var items []models.Exercise
	if err := json.Unmarshal(data, &items); err != nil {
		t.Fatalf("embedded exercises.json is not valid: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("embedded exercise list is empty")
	}

	seen := map[string]bool{}
	for _, e := range items {
		if e.Name == "" {
			t.Fatal("exercise without a name")
		}
		if seen[e.Name] {
			t.Fatalf("duplicate exercise name %q (names have a unique index)", e.Name)
		}
		seen[e.Name] = true
	}
	// The history service skips these two, so they must exist under these exact names.
	for _, name := range []string{"Warm-Up", "Cool-Down"} {
		if !seen[name] {
			t.Errorf("%s missing from the exercise list", name)
		}
	}
}

func TestExplicitPathStillOverridesEmbeddedList(t *testing.T) {
	if _, err := exercisesJSON("definitely-not-a-real-file.json"); err == nil {
		t.Fatal("an explicit missing path should error rather than silently use the embedded list")
	}
}
