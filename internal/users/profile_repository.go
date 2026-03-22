package users

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProfileRepository struct {
	collection *mongo.Collection
}

func NewProfileRepository(db *mongo.Database) *ProfileRepository {
	return &ProfileRepository{collection: db.Collection("profiles")}
}

func (r *ProfileRepository) Upsert(ctx context.Context, profile Profile) error {
	profile.UpdatedAt = time.Now()
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"public_id": profile.PublicID},
		bson.M{"$set": profile},
		options.Update().SetUpsert(true),
	)
	return err
}

func (r *ProfileRepository) GetByPublicID(ctx context.Context, publicID string) (Profile, error) {
	var profile Profile
	err := r.collection.FindOne(ctx, bson.M{"public_id": publicID}).Decode(&profile)
	return profile, err
}

func boolPtr(b bool) *bool { return &b }
