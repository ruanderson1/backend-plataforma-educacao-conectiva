package classroom

import (
	"go.mongodb.org/mongo-driver/mongo"
)

type ReportRepository struct {
	StudentRepo    StudentRepository
	StudentObsRepo StudentObservationRepository
	StudentLLMRepo StudentLLMReportRepository
	ClassObsRepo   ClassObservationRepository
	ClassLLMRepo   ClassLLMReportRepository
}

func NewReportRepository(db *mongo.Database, studentRepo StudentRepository) *ReportRepository {
	return &ReportRepository{
		StudentRepo:    studentRepo,
		StudentObsRepo: NewStudentObservationRepo(db),
		StudentLLMRepo: NewStudentLLMReportRepo(db),
		ClassObsRepo:   NewClassObservationRepo(db),
		ClassLLMRepo:   NewClassLLMReportRepo(db),
	}
}
