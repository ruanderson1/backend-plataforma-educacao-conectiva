package classroom

import (
	"context"
)

type StudentLLMReportRepository interface {
	Create(ctx context.Context, report *StudentLLMReport) error
	FindByStudentAndPeriod(ctx context.Context, studentID string, periodo string) ([]StudentLLMReport, error)
}
