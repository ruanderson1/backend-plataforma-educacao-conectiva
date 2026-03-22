package classroom

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrNotFound            = errors.New("classroom not found")
	ErrDuplicateAccessCode = errors.New("accessCode already exists")
)

type Repository struct {
	collection *mongo.Collection
}

func NewRepository(db *mongo.Database) *Repository {
	return &Repository{collection: db.Collection("classrooms")}
}

func generateAccessCode() string {
	letters := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	code := make([]rune, 8)
	for i := range code {
		code[i] = letters[rand.Intn(len(letters))]
	}
	return fmt.Sprintf("CL-%s", string(code))
}

func (r *Repository) Create(ctx context.Context, c *ClassRoom) error {
	c.ID = primitive.NewObjectID()
	c.CreatedAt = time.Now()
	c.UpdatedAt = c.CreatedAt
	for {
		c.AccessCode = generateAccessCode()
		count, err := r.collection.CountDocuments(ctx, bson.M{"accessCode": c.AccessCode})
		if err != nil {
			return err
		}
		if count == 0 {
			break
		}
	}
	_, err := r.collection.InsertOne(ctx, c)
	if mongo.IsDuplicateKeyError(err) {
		return ErrDuplicateAccessCode
	}
	return err
}

func (r *Repository) FindByTeacher(ctx context.Context, teacherId string) ([]ClassRoom, error) {
	cur, err := r.collection.Find(ctx, bson.M{"teacherId": teacherId})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var classes []ClassRoom
	for cur.Next(ctx) {
		var c ClassRoom
		if err := cur.Decode(&c); err != nil {
			return nil, err
		}
		classes = append(classes, c)
	}
	return classes, nil
}

func (r *Repository) FindByID(ctx context.Context, id primitive.ObjectID, teacherId string) (*ClassRoom, error) {
	var c ClassRoom
	err := r.collection.FindOne(ctx, bson.M{"_id": id, "teacherId": teacherId}).Decode(&c)
	if err == mongo.ErrNoDocuments {
		return nil, ErrNotFound
	}
	return &c, err
}

func (r *Repository) Update(ctx context.Context, id primitive.ObjectID, teacherId string, update bson.M) error {
	update["updatedAt"] = time.Now()
	res, err := r.collection.UpdateOne(ctx, bson.M{"_id": id, "teacherId": teacherId}, bson.M{"$set": update})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id primitive.ObjectID, teacherId string) error {
	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": id, "teacherId": teacherId})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}
