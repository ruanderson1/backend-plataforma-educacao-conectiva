package classroom_test

import (
	"context"
	"plataforma/internal/classroom"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// --- Mocks para os repositórios ---
type mockStudentRepo struct{}

func (m *mockStudentRepo) Create(ctx context.Context, student *classroom.Student) error { return nil }
func (m *mockStudentRepo) FindByClassroom(ctx context.Context, classroomID primitive.ObjectID) ([]classroom.Student, error) {
	return []classroom.Student{}, nil
}
func (m *mockStudentRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*classroom.Student, error) {
	return &classroom.Student{
		ID:                id,
		Nome:              "João Silva",
		Notas:             []classroom.StudentNote{},
		HistoricoNotas:    []classroom.StudentNote{},
		ObservacoesGerais: "Aluno participativo",
		Frequencia:        []classroom.AttendanceRecord{},
		Sala:              primitive.NewObjectID(),
		Role:              "aluno",
	}, nil
}
func (m *mockStudentRepo) Update(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	return nil
}
func (m *mockStudentRepo) Delete(ctx context.Context, id primitive.ObjectID) error { return nil }

type mockStudentObsRepo struct{}

func (m *mockStudentObsRepo) Create(ctx context.Context, obs *classroom.StudentObservation) error {
	return nil
}
func (m *mockStudentObsRepo) FindByStudentAndPeriod(ctx context.Context, studentID, periodo string) ([]classroom.StudentObservation, error) {
	return []classroom.StudentObservation{}, nil
}

type mockStudentLLMRepo struct{}

func (m *mockStudentLLMRepo) Create(ctx context.Context, report *classroom.StudentLLMReport) error {
	return nil
}
func (m *mockStudentLLMRepo) FindByStudentAndPeriod(ctx context.Context, studentID, periodo string) ([]classroom.StudentLLMReport, error) {
	return []classroom.StudentLLMReport{}, nil
}

type mockClassObsRepo struct{}

func (m *mockClassObsRepo) Create(ctx context.Context, obs *classroom.ClassObservation) error {
	return nil
}
func (m *mockClassObsRepo) FindByClassAndPeriod(ctx context.Context, classID, periodo string) ([]classroom.ClassObservation, error) {
	return []classroom.ClassObservation{}, nil
}

type mockClassLLMRepo struct{}

func (m *mockClassLLMRepo) Create(ctx context.Context, report *classroom.ClassLLMReport) error {
	return nil
}
func (m *mockClassLLMRepo) FindByClassAndPeriod(ctx context.Context, classID, periodo string) ([]classroom.ClassLLMReport, error) {
	return []classroom.ClassLLMReport{}, nil
}

// --- Setup atualizado para os testes ---
func setupTestHandler() *classroom.ReportHandler {
	return classroom.NewReportHandler(
		&mockStudentRepo{},
		&mockStudentObsRepo{},
		&mockStudentLLMRepo{},
		&mockClassObsRepo{},
		&mockClassLLMRepo{},
	)
}
