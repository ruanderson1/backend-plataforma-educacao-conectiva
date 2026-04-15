package classroom

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func toFloat64(value interface{}) (float64, bool) {
	number, ok := value.(float64)
	if !ok {
		return 0, false
	}
	return number, true
}

func normalizeAttendanceRecordsFromUpdate(value interface{}, existing AttendanceRecords) (AttendanceRecords, error) {
	parseRecordMap := func(raw map[string]interface{}) (AttendanceRecord, error) {
		tipoRaw, ok := raw["tipo"]
		if !ok {
			return AttendanceRecord{}, ErrInvalidAttendance
		}
		tipo, ok := tipoRaw.(string)
		if !ok {
			return AttendanceRecord{}, ErrInvalidAttendance
		}

		data := ""
		if dataRaw, hasData := raw["data"]; hasData {
			dataString, ok := dataRaw.(string)
			if !ok {
				return AttendanceRecord{}, ErrInvalidAttendance
			}
			data = strings.TrimSpace(dataString)
		}

		record, err := NormalizeAttendanceRecord(AttendanceRecord{Data: data, Tipo: tipo})
		if err != nil {
			return AttendanceRecord{}, err
		}
		if record.Data == "" {
			record.Data = time.Now().Format("2006-01-02")
		}
		return record, nil
	}

	switch typed := value.(type) {
	case []interface{}:
		normalized := make(AttendanceRecords, 0, len(typed))
		for _, item := range typed {
			recordMap, ok := item.(map[string]interface{})
			if !ok {
				return nil, ErrInvalidAttendance
			}
			record, err := parseRecordMap(recordMap)
			if err != nil {
				return nil, err
			}
			normalized = append(normalized, record)
		}
		return normalized, nil
	case map[string]interface{}:
		record, err := parseRecordMap(typed)
		if err != nil {
			return nil, err
		}
		normalized := append(append(AttendanceRecords{}, existing...), record)
		return normalized, nil
	case string:
		tipo := NormalizeAttendanceStatus(typed)
		if !IsValidAttendanceStatus(tipo) {
			return nil, ErrInvalidAttendance
		}
		normalized := append(append(AttendanceRecords{}, existing...), AttendanceRecord{
			Data: time.Now().Format("2006-01-02"),
			Tipo: tipo,
		})
		return normalized, nil
	default:
		return nil, ErrInvalidAttendance
	}
}

type Service struct {
	repo        *Repository
	studentRepo StudentRepository
}

type StudentRepository interface {
	Create(ctx context.Context, student *Student) error
	FindByClassroom(ctx context.Context, classroomID primitive.ObjectID) ([]Student, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*Student, error)
	Update(ctx context.Context, id primitive.ObjectID, update bson.M) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

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

func (s *Service) CreateStudent(ctx context.Context, teacherId string, student *Student) (map[string]interface{}, error) {
	if err := student.Validate(); err != nil {
		return nil, err
	}
	if err := s.ensureClassroomExists(ctx, student.Sala); err != nil {
		return nil, err
	}
	if err := s.studentRepo.Create(ctx, student); err != nil {
		return nil, err
	}
	return studentToResponse(student), nil
}

func (s *Service) ListStudentsByClassroom(ctx context.Context, teacherId string, classroomID primitive.ObjectID) ([]map[string]interface{}, error) {
	if err := s.ensureClassroomExists(ctx, classroomID); err != nil {
		return nil, err
	}
	students, err := s.studentRepo.FindByClassroom(ctx, classroomID)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(students))
	for _, student := range students {
		studentCopy := student
		result = append(result, studentToResponse(&studentCopy))
	}
	return result, nil
}

func (s *Service) GetStudentByID(ctx context.Context, teacherId string, id primitive.ObjectID) (map[string]interface{}, error) {
	student, err := s.studentRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureClassroomExists(ctx, student.Sala); err != nil {
		return nil, err
	}
	return studentToResponse(student), nil
}

func (s *Service) UpdateStudent(ctx context.Context, teacherId string, id primitive.ObjectID, update map[string]interface{}) error {
	student, err := s.studentRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.ensureClassroomExists(ctx, student.Sala); err != nil {
		return err
	}

	normalized := bson.M{}
	if nome, ok := update["nome"]; ok {
		normalized["nome"] = nome
	}
	if observacoesGerais, ok := update["observacoesGerais"]; ok {
		normalized["observacoesGerais"] = observacoesGerais
	}
	if role, ok := update["role"]; ok {
		normalized["role"] = role
	}
	if frequencia, ok := update["frequencia"]; ok {
		normalizedFrequency, err := normalizeAttendanceRecordsFromUpdate(frequencia, student.Frequencia)
		if err != nil {
			return err
		}
		normalized["frequencia"] = normalizedFrequency
	}
	if notas, ok := update["notas"]; ok {
		normalized["notas"] = notas
	}
	if historicoNotas, ok := update["historicoNotas"]; ok {
		normalized["historicoNotas"] = historicoNotas
	}

	if len(normalized) == 0 {
		return ErrInvalidStudent
	}

	if notasRaw, ok := normalized["notas"]; ok {
		cleanNotes := make([]bson.M, 0)

		validateAndAppend := func(nota map[string]interface{}) error {
			cleanNote := bson.M{}
			notaMaxima := 10.0
			if tipoRaw, hasTipo := nota["tipo"]; hasTipo {
				tipo, ok := tipoRaw.(string)
				if !ok || !IsValidNoteType(NoteType(tipo)) {
					return ErrInvalidNoteType
				}
				cleanNote["tipo"] = string(NormalizeNoteType(tipo))
			}
			if notaMaximaRaw, hasNotaMaxima := nota["notaMaxima"]; hasNotaMaxima {
				parsedNotaMaxima, ok := toFloat64(notaMaximaRaw)
				if !ok {
					return ErrInvalidStudent
				}
				notaMaxima = NormalizeNoteMaxScore(parsedNotaMaxima)
				cleanNote["notaMaxima"] = notaMaxima
			}
			if pontuacaoRaw, hasPontuacao := nota["pontuacao"]; hasPontuacao {
				pontuacao, ok := toFloat64(pontuacaoRaw)
				if !ok {
					return ErrInvalidStudent
				}
				if pontuacao > notaMaxima {
					return ErrInvalidStudent
				}
				cleanNote["pontuacao"] = pontuacao
			}
			if pesoRaw, hasPeso := nota["peso"]; hasPeso {
				peso, ok := toFloat64(pesoRaw)
				if !ok {
					return ErrInvalidStudent
				}
				cleanNote["peso"] = NormalizeNoteWeight(peso)
			}
			if observacoesRaw, hasObservacoes := nota["observacoes"]; hasObservacoes {
				cleanNote["observacoes"] = observacoesRaw
			}
			if len(cleanNote) > 0 {
				cleanNotes = append(cleanNotes, cleanNote)
			}
			return nil
		}

		switch notasTyped := notasRaw.(type) {
		case map[string]interface{}:
			if err := validateAndAppend(notasTyped); err != nil {
				return err
			}
		case []interface{}:
			for _, item := range notasTyped {
				nota, ok := item.(map[string]interface{})
				if !ok {
					return ErrInvalidStudent
				}
				if err := validateAndAppend(nota); err != nil {
					return err
				}
			}
		default:
			return ErrInvalidStudent
		}

		normalized["notas"] = cleanNotes
	}
	if historicoNotasRaw, ok := normalized["historicoNotas"]; ok {
		notas, ok := historicoNotasRaw.([]interface{})
		if !ok {
			return ErrInvalidStudent
		}
		for _, item := range notas {
			nota, ok := item.(map[string]interface{})
			if !ok {
				return ErrInvalidStudent
			}
			notaMaxima := 10.0
			if tipoRaw, hasTipo := nota["tipo"]; hasTipo {
				tipo, ok := tipoRaw.(string)
				if !ok || !IsValidNoteType(NoteType(tipo)) {
					return ErrInvalidNoteType
				}
				nota["tipo"] = string(NormalizeNoteType(tipo))
			}
			if notaMaximaRaw, hasNotaMaxima := nota["notaMaxima"]; hasNotaMaxima {
				parsedNotaMaxima, ok := toFloat64(notaMaximaRaw)
				if !ok {
					return ErrInvalidStudent
				}
				notaMaxima = NormalizeNoteMaxScore(parsedNotaMaxima)
				nota["notaMaxima"] = notaMaxima
			}
			if pontuacaoRaw, hasPontuacao := nota["pontuacao"]; hasPontuacao {
				pontuacao, ok := toFloat64(pontuacaoRaw)
				if !ok {
					return ErrInvalidStudent
				}
				if pontuacao > notaMaxima {
					return ErrInvalidStudent
				}
			}
			if pesoRaw, hasPeso := nota["peso"]; hasPeso {
				peso, ok := toFloat64(pesoRaw)
				if !ok {
					return ErrInvalidStudent
				}
				nota["peso"] = NormalizeNoteWeight(peso)
			}
		}
	}

	return s.studentRepo.Update(ctx, id, normalized)
}

func (s *Service) DeleteStudent(ctx context.Context, teacherId string, id primitive.ObjectID) error {
	student, err := s.studentRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.ensureClassroomExists(ctx, student.Sala); err != nil {
		return err
	}
	return s.studentRepo.Delete(ctx, id)
}

func (s *Service) ensureTeacherOwnsClassroom(ctx context.Context, classroomID primitive.ObjectID, teacherId string) error {
	_, err := s.repo.FindByID(ctx, classroomID, teacherId)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) ensureClassroomExists(ctx context.Context, classroomID primitive.ObjectID) error {
	_, err := s.repo.FindByIDAny(ctx, classroomID)
	if err != nil {
		return err
	}
	return nil
}

func studentToResponse(student *Student) map[string]interface{} {
	return map[string]interface{}{
		"_id":               student.ID.Hex(),
		"nome":              student.Nome,
		"notas":             student.Notas,
		"historicoNotas":    student.HistoricoNotas,
		"observacoesGerais": student.ObservacoesGerais,
		"frequencia":        student.Frequencia,
		"sala":              student.Sala.Hex(),
		"role":              student.Role,
	}
}
