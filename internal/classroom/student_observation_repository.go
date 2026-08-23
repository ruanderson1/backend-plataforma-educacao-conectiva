package classroom

import (
	"context"
)

type StudentObservationRepository interface {
	Create(ctx context.Context, obs *StudentObservation) error
	FindByStudentAndPeriod(ctx context.Context, studentID string, periodo string) ([]StudentObservation, error)
	UpdateByID(ctx context.Context, observationID string, updates map[string]interface{}) (*StudentObservation, error)
	DeleteByID(ctx context.Context, observationID string) error
}
