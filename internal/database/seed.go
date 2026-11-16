package database

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"time"

	"fitness-tracker/internal/config"
	"fitness-tracker/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// defaultExercises is the built-in exercise list, so the binary needs no file next to it.
//
//go:embed exercises.json
var defaultExercises []byte

// exercisesJSON returns the exercise data to seed from: the file at path when
// EXERCISES_JSON_PATH is set, otherwise the embedded list.
func exercisesJSON(path string) ([]byte, error) {
	if path == "" {
		return defaultExercises, nil
	}
	return os.ReadFile(path)
}

// EnsureExercisesSeeded checks the exercises collection and, if empty,
// populates it from exercises.json. Safe to run on startup.
func EnsureExercisesSeeded(ctx context.Context, db *mongo.Database) error {
	exercises := db.Collection("exercises")

	// Check if collection has any documents
	count, err := exercises.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	path := config.AppConfig.ExercisesJSONPath
	if path == "" {
		log.Println("Seeding exercises from the built-in list...")
	} else {
		log.Printf("Seeding exercises from %s...", path)
	}
	inserted, err := SeedExercisesFromJSON(ctx, db, path)
	if err != nil {
		return err
	}
	if inserted > 0 {
		log.Printf("Inserted %d exercises", inserted)
	}

	// Ensure Warm-Up and Cool-Down exist even if not present in the JSON
	if inserted == 0 {
		log.Println("JSON file not found/empty; ensuring Warm-Up and Cool-Down placeholders...")
	}
	return ensureWarmupCooldownDefaults(ctx, exercises)
}

// SeedExercisesFromJSON reads exercises from a JSON file (or the embedded list when path
// is empty) and inserts them into the exercises collection. Returns how many were inserted.
func SeedExercisesFromJSON(ctx context.Context, db *mongo.Database, path string) (int, error) {
	data, err := exercisesJSON(path)
	if err != nil {
		// If file doesn't exist, treat as non-fatal and return false
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}

	var items []models.Exercise
	if err := json.Unmarshal(data, &items); err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}

	// De-duplicate by name within the JSON payload
	seen := make(map[string]struct{})
	var docs []interface{}
	for _, it := range items {
		if it.Name == "" {
			continue
		}
		if _, exists := seen[it.Name]; exists {
			continue
		}
		seen[it.Name] = struct{}{}
		docs = append(docs, it)
	}
	if len(docs) == 0 {
		return 0, nil
	}

	insertCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	res, err := db.Collection("exercises").InsertMany(insertCtx, docs)
	if err != nil {
		return 0, err
	}
	return len(res.InsertedIDs), nil
}

// ensureWarmupCooldownDefaults inserts placeholders for Warm-Up and Cool-Down
// if they don't already exist.
func ensureWarmupCooldownDefaults(ctx context.Context, exercises *mongo.Collection) error {
	// Warm-Up
	if inserted, err := upsertExerciseByName(ctx, exercises, models.Exercise{
		Name:       "Warm-Up",
		Category:   "General",
		Variations: []string{"None"},
		Equipment:  []string{"None"},
	}); err != nil {
		return err
	} else if inserted {
		log.Println("Upserted Warm-Up exercise")
	}
	// Cool-Down
	if inserted, err := upsertExerciseByName(ctx, exercises, models.Exercise{
		Name:       "Cool-Down",
		Category:   "General",
		Variations: []string{"None"},
		Equipment:  []string{"None"},
	}); err != nil {
		return err
	} else if inserted {
		log.Println("Upserted Cool-Down exercise")
	}
	return nil
}

func upsertExerciseByName(ctx context.Context, exercises *mongo.Collection, ex models.Exercise) (bool, error) {
	res, err := exercises.UpdateOne(ctx,
		bson.M{"name": ex.Name},
		bson.M{"$setOnInsert": ex},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return false, err
	}
	return res.UpsertedID != nil, nil
}
