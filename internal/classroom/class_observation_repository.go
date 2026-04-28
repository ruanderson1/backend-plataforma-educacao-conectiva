package classroom

import (
	"context"
)

type ClassObservationRepository interface {
	Create(ctx context.Context, obs *ClassObservation) error
	FindByClassAndPeriod(ctx context.Context, classID string, periodo string) ([]ClassObservation, error)
}
