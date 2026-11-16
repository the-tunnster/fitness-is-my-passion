package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestRespondUpdateErrorDistinguishesMissingFromFailed(t *testing.T) {
	rec := httptest.NewRecorder()
	respondUpdateError(rec, mongo.ErrNoDocuments, "session")
	if rec.Code != http.StatusNotFound {
		t.Errorf("nothing matched: got %d, want 404 (never a silent success)", rec.Code)
	}

	rec = httptest.NewRecorder()
	respondUpdateError(rec, errors.New("connection reset"), "session")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("other failures: got %d, want 500", rec.Code)
	}
}

func TestExerciseUpdatesKeepsOnlyEditableFields(t *testing.T) {
	incoming := map[string]interface{}{
		"id":         "6af8503de2ddcac3e4a4b7c6", // sent by clients; must not be stored
		"_id":        "000000000000000000000000",
		"name":       "Squat",
		"category":   "Strength",
		"variations": []interface{}{"High bar"},
		"equipment":  []interface{}{"Barbell"},
		"isAdmin":    true,
	}
	want := bson.M{
		"name":       "Squat",
		"category":   "Strength",
		"variations": []interface{}{"High bar"},
		"equipment":  []interface{}{"Barbell"},
	}
	if got := exerciseUpdates(incoming); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestExerciseUpdatesWithNothingEditableIsEmpty(t *testing.T) {
	if got := exerciseUpdates(map[string]interface{}{"id": "x", "foo": 1}); len(got) != 0 {
		t.Fatalf("expected no updates, got %v", got)
	}
}
