package database

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func InitIndexes(ctx context.Context, db *mongo.Database) error {
	if err := initRoutineIndexes(ctx, db); err != nil {
		return err
	}
	if err := initExerciseIndexes(ctx, db); err != nil {
		return err
	}
	if err := initSessionIndexes(ctx, db); err != nil {
		return err
	}
	if err := initWorkoutIndexes(ctx, db); err != nil {
		return err
	}
	if err := initExerciseHistoryIndexes(ctx, db); err != nil {
		return err
	}
	return nil
}

func initRoutineIndexes(ctx context.Context, db *mongo.Database) error {
	routines := db.Collection("routines")
	_, err := routines.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "userID", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "userID", Value: 1}, {Key: "name", Value: 1}},
		},
	})
	return err
}

func initExerciseIndexes(ctx context.Context, db *mongo.Database) error {
	exercises := db.Collection("exercises")

	// Create unique index on name (idempotent if it already exists uniquely)
	_, err := exercises.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

func initSessionIndexes(ctx context.Context, db *mongo.Database) error {
	sessions := db.Collection("sessions")
	_, err := sessions.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userID", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		return err
	}

	// Sessions used to expire after 48 hours through a TTL index on lastUpdated; remove it
	// from databases that still have it.
	if _, err := sessions.Indexes().DropOne(ctx, "lastUpdated_1"); err != nil {
		var cmdErr mongo.CommandError
		if errors.As(err, &cmdErr) && (cmdErr.Code == 27 || cmdErr.Code == 26) { // IndexNotFound, NamespaceNotFound
			return nil
		}
		return err
	}
	return nil
}

func initWorkoutIndexes(ctx context.Context, db *mongo.Database) error {
	workouts := db.Collection("workouts")
	_, err := workouts.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "userID", Value: 1}, {Key: "workoutDate", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "routineID", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "userID", Value: 1}, {Key: "routineID", Value: 1}, {Key: "workoutDate", Value: -1}},
		},
	})
	return err
}

func initExerciseHistoryIndexes(ctx context.Context, db *mongo.Database) error {
	exerciseHistory := db.Collection("exerciseHistory")
	_, err := exerciseHistory.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "userID", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "exerciseID", Value: 1}},
		},
	})
	return err
}
