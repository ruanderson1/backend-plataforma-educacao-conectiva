package classroom

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service struct {
	repo        *Repository
	studentRepo StudentRepository // pode ser nil se não usar
}

type StudentRepository interface{}

func NewService(repo *Repository, studentRepo StudentRepository) *Service {
	return &Service{repo: repo, studentRepo: studentRepo}
}

func (s *Service) Create(ctx context.Context, c *ClassRoom) error {
	return s.repo.Create(ctx, c)
}

func (s *Service) ListByTeacher(ctx context.Context, teacherId string) ([]map[string]interface{}, error) {
	classes, err := s.repo.FindByTeacher(ctx, teacherId)
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for _, c := range classes {
		result = append(result, map[string]interface{}{
			"_id":        c.ID.Hex(),
			"name":       c.Name,
			"accessCode": c.AccessCode,
			"yearGrade":  c.YearGrade,
			"teacherId":  c.TeacherID,
			"createdAt":  c.CreatedAt,
			"updatedAt":  c.UpdatedAt,
		})
	}
	return result, nil
}

func (s *Service) GetByID(ctx context.Context, id primitive.ObjectID, teacherId string) (map[string]interface{}, error) {
	c, err := s.repo.FindByID(ctx, id, teacherId)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"_id":        c.ID.Hex(),
		"name":       c.Name,
		"accessCode": c.AccessCode,
		"yearGrade":  c.YearGrade,
		"teacherId":  c.TeacherID,
		"createdAt":  c.CreatedAt,
		"updatedAt":  c.UpdatedAt,
	}, nil
}

func (s *Service) Update(ctx context.Context, id primitive.ObjectID, teacherId string, update map[string]interface{}) error {
	return s.repo.Update(ctx, id, teacherId, bson.M(update))
}

func (s *Service) Delete(ctx context.Context, id primitive.ObjectID, teacherId string) error {
	return s.repo.Delete(ctx, id, teacherId)
}
