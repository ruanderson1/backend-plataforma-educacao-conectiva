package classroom

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type classLLMReportRepo struct {
	collection *mongo.Collection
}

func NewClassLLMReportRepo(db *mongo.Database) ClassLLMReportRepository {
	return &classLLMReportRepo{collection: db.Collection("class_llm_reports")}
}

func (r *classLLMReportRepo) Create(ctx context.Context, report *ClassLLMReport) error {
	_, err := r.collection.InsertOne(ctx, report)
	return err
}

func (r *classLLMReportRepo) FindByClassAndPeriod(ctx context.Context, classID string, periodo string) ([]ClassLLMReport, error) {
	filter := bson.M{"class_id": classID}
	if periodo != "" {
		filter["periodo_referencia"] = periodo
	}
	cur, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var result []ClassLLMReport
	for cur.Next(ctx) {
		var rep ClassLLMReport
		if err := cur.Decode(&rep); err != nil {
			return nil, err
		}
		result = append(result, rep)
	}
	return result, nil
}
