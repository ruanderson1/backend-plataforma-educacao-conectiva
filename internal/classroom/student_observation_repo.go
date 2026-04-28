package classroom

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type studentObservationRepo struct {
	collection *mongo.Collection
}

func NewStudentObservationRepo(db *mongo.Database) StudentObservationRepository {
	return &studentObservationRepo{collection: db.Collection("student_observations")}
}

func (r *studentObservationRepo) Create(ctx context.Context, obs *StudentObservation) error {
	_, err := r.collection.InsertOne(ctx, obs)
	return err
}

func (r *studentObservationRepo) FindByStudentAndPeriod(ctx context.Context, studentID string, periodo string) ([]StudentObservation, error) {
	filter := bson.M{"student_id": studentID}
	if periodo != "" {
		filter["periodo_referencia"] = periodo
	}
	cur, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var result []StudentObservation
	for cur.Next(ctx) {
		var obs StudentObservation
		if err := cur.Decode(&obs); err != nil {
			return nil, err
		}
		result = append(result, obs)
	}
	return result, nil
}
