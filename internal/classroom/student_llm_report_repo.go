package classroom

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type studentLLMReportRepo struct {
	collection *mongo.Collection
}

func NewStudentLLMReportRepo(db *mongo.Database) StudentLLMReportRepository {
	return &studentLLMReportRepo{collection: db.Collection("student_llm_reports")}
}

func (r *studentLLMReportRepo) Create(ctx context.Context, report *StudentLLMReport) error {
	_, err := r.collection.InsertOne(ctx, report)
	return err
}

func (r *studentLLMReportRepo) FindByStudentAndPeriod(ctx context.Context, studentID string, periodo string) ([]StudentLLMReport, error) {
	filter := bson.M{"student_id": studentID}
	if periodo != "" {
		filter["periodo_referencia"] = periodo
	}
	cur, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var result []StudentLLMReport
	for cur.Next(ctx) {
		var rep StudentLLMReport
		if err := cur.Decode(&rep); err != nil {
			return nil, err
		}
		result = append(result, rep)
	}
	return result, nil
}
