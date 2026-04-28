package classroom

import (
	"context"
)

type ClassLLMReportRepository interface {
	Create(ctx context.Context, report *ClassLLMReport) error
	FindByClassAndPeriod(ctx context.Context, classID string, periodo string) ([]ClassLLMReport, error)
}
