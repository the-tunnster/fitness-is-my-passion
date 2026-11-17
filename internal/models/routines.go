package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RoutineExercise struct {
	ExerciseID primitive.ObjectID `bson:"exerciseID" json:"exercise_id"`
	Name       string             `bson:"name" json:"name"`
	TargetSets int                `bson:"targetSets" json:"target_sets"`
	TargetReps int                `bson:"targetReps" json:"target_reps"`
}

type FullRoutine struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"userID" json:"user_id"`
	Name      string             `bson:"name" json:"name"`
	Exercises []RoutineExercise  `bson:"exercises" json:"exercises"`
}

type Routine struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID primitive.ObjectID `bson:"userID" json:"user_id"`
	Name   string             `bson:"name" json:"name"`
}
