package classroom

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type classObservationRepo struct {
	collection *mongo.Collection
}

func NewClassObservationRepo(db *mongo.Database) ClassObservationRepository {
	return &classObservationRepo{collection: db.Collection("class_observations")}
}

func (r *classObservationRepo) Create(ctx context.Context, obs *ClassObservation) error {
	_, err := r.collection.InsertOne(ctx, obs)
	return err
}

func (r *classObservationRepo) FindByClassAndPeriod(ctx context.Context, classID string, periodo string) ([]ClassObservation, error) {
	filter := bson.M{"class_id": classID}
	if periodo != "" {
		filter["periodo_referencia"] = periodo
	}
	cur, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var result []ClassObservation
	for cur.Next(ctx) {
		var obs ClassObservation
		if err := cur.Decode(&obs); err != nil {
			return nil, err
		}
		result = append(result, obs)
	}
	return result, nil
}
