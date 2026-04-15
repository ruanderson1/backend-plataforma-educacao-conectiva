package classroom

import (
	"context"
	"errors"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrStudentNotFound   = errors.New("student not found")
	ErrInvalidStudent    = errors.New("invalid student payload")
	ErrInvalidNoteType   = errors.New("invalid note type")
	ErrInvalidAttendance = errors.New("invalid attendance status")
	ErrClassroomMissing  = errors.New("classroom is required")
)

type NoteType string
type AttendanceStatus string

const (
	NoteTypeProva              NoteType         = "Prova"
	NoteTypeTrabalho           NoteType         = "Trabalho"
	NoteTypeProjeto            NoteType         = "Projeto"
	NoteTypeTeste              NoteType         = "Teste"
	AttendancePresenca         AttendanceStatus = "presenca"
	AttendanceFalta            AttendanceStatus = "falta"
	AttendanceFaltaJustificada AttendanceStatus = "falta_justificada"
	StudentRole                string           = "aluno"
)

type StudentNote struct {
	Tipo        NoteType `bson:"tipo" json:"tipo"`
	Pontuacao   float64  `bson:"pontuacao" json:"pontuacao"`
	NotaMaxima  float64  `bson:"notaMaxima,omitempty" json:"notaMaxima,omitempty"`
	Peso        float64  `bson:"peso,omitempty" json:"peso,omitempty"`
	Observacoes string   `bson:"observacoes" json:"observacoes"`
}

type AttendanceRecord struct {
	Data string `bson:"data" json:"data"`
	Tipo string `bson:"tipo" json:"tipo"`
}

type StudentNotes []StudentNote
type AttendanceRecords []AttendanceRecord

// UnmarshalBSONValue keeps compatibility with old records where "notas" was a single object.
func (n *StudentNotes) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	switch t {
	case bsontype.Array:
		var list []StudentNote
		if err := bson.UnmarshalValue(t, data, &list); err != nil {
			return err
		}
		*n = list
		return nil
	case bsontype.EmbeddedDocument:
		var one StudentNote
		if err := bson.UnmarshalValue(t, data, &one); err != nil {
			return err
		}
		*n = []StudentNote{one}
		return nil
	case bsontype.Null, bsontype.Undefined:
		*n = nil
		return nil
	default:
		*n = nil
		return nil
	}
}

// UnmarshalBSONValue keeps compatibility with old records where "frequencia" was a single string.
func (a *AttendanceRecords) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	switch t {
	case bsontype.Array:
		var list []AttendanceRecord
		if err := bson.UnmarshalValue(t, data, &list); err != nil {
			return err
		}
		*a = list
		return nil
	case bsontype.EmbeddedDocument:
		var one AttendanceRecord
		if err := bson.UnmarshalValue(t, data, &one); err != nil {
			return err
		}
		*a = []AttendanceRecord{one}
		return nil
	case bsontype.String:
		var status string
		if err := bson.UnmarshalValue(t, data, &status); err != nil {
			return err
		}
		normalized := NormalizeAttendanceStatus(status)
		if normalized == "" {
			*a = nil
			return nil
		}
		*a = []AttendanceRecord{{Tipo: normalized, Data: ""}}
		return nil
	case bsontype.Null, bsontype.Undefined:
		*a = nil
		return nil
	default:
		*a = nil
		return nil
	}
}

type Student struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	Nome              string             `bson:"nome" json:"nome"`
	Notas             StudentNotes       `bson:"notas" json:"notas"`
	HistoricoNotas    []StudentNote      `bson:"historicoNotas,omitempty" json:"historicoNotas,omitempty"`
	ObservacoesGerais string             `bson:"observacoesGerais,omitempty" json:"observacoesGerais,omitempty"`
	Frequencia        AttendanceRecords  `bson:"frequencia" json:"frequencia"`
	Sala              primitive.ObjectID `bson:"sala" json:"sala"`
	Role              string             `bson:"role" json:"role"`
}

type StudentRepo struct {
	collection *mongo.Collection
}

func NewStudentRepo(db *mongo.Database) *StudentRepo {
	return &StudentRepo{collection: db.Collection("students")}
}

func IsValidNoteType(noteType NoteType) bool {
	switch strings.ToLower(strings.TrimSpace(string(noteType))) {
	case strings.ToLower(string(NoteTypeProva)), strings.ToLower(string(NoteTypeTrabalho)), strings.ToLower(string(NoteTypeProjeto)), strings.ToLower(string(NoteTypeTeste)):
		return true
	default:
		return false
	}
}

func NormalizeNoteType(noteType string) NoteType {
	switch strings.ToLower(strings.TrimSpace(noteType)) {
	case strings.ToLower(string(NoteTypeProva)):
		return NoteTypeProva
	case strings.ToLower(string(NoteTypeTrabalho)):
		return NoteTypeTrabalho
	case strings.ToLower(string(NoteTypeProjeto)):
		return NoteTypeProjeto
	case strings.ToLower(string(NoteTypeTeste)):
		return NoteTypeTeste
	default:
		return NoteType(strings.TrimSpace(noteType))
	}
}

func NormalizeNoteWeight(weight float64) float64 {
	if weight <= 0 {
		return 1
	}
	return weight
}

func NormalizeNoteMaxScore(maxScore float64) float64 {
	if maxScore <= 0 {
		return 10
	}
	return maxScore
}

func IsValidAttendanceStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case string(AttendancePresenca), string(AttendanceFalta), string(AttendanceFaltaJustificada):
		return true
	default:
		return false
	}
}

func NormalizeAttendanceStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case string(AttendancePresenca):
		return string(AttendancePresenca)
	case string(AttendanceFalta):
		return string(AttendanceFalta)
	case string(AttendanceFaltaJustificada):
		return string(AttendanceFaltaJustificada)
	default:
		return strings.TrimSpace(status)
	}
}

func NormalizeAttendanceRecord(record AttendanceRecord) (AttendanceRecord, error) {
	tipo := NormalizeAttendanceStatus(record.Tipo)
	if tipo == "" || !IsValidAttendanceStatus(tipo) {
		return AttendanceRecord{}, ErrInvalidAttendance
	}

	return AttendanceRecord{
		Data: strings.TrimSpace(record.Data),
		Tipo: tipo,
	}, nil
}

func (s *Student) Validate() error {
	if strings.TrimSpace(s.Nome) == "" {
		return ErrInvalidStudent
	}
	if s.Sala.IsZero() {
		return ErrClassroomMissing
	}
	if strings.TrimSpace(s.Role) == "" {
		s.Role = StudentRole
	}
	for i := range s.Frequencia {
		normalizedRecord, err := NormalizeAttendanceRecord(s.Frequencia[i])
		if err != nil {
			return err
		}
		s.Frequencia[i] = normalizedRecord
	}
	for i := range s.Notas {
		nota := s.Notas[i]
		if nota.Tipo != "" && !IsValidNoteType(nota.Tipo) {
			return ErrInvalidNoteType
		}
		s.Notas[i].NotaMaxima = NormalizeNoteMaxScore(nota.NotaMaxima)
		if s.Notas[i].Pontuacao > s.Notas[i].NotaMaxima {
			return ErrInvalidStudent
		}
		s.Notas[i].Peso = NormalizeNoteWeight(nota.Peso)
	}
	return nil
}

func (r *StudentRepo) Create(ctx context.Context, student *Student) error {
	if err := student.Validate(); err != nil {
		return err
	}
	student.ID = primitive.NewObjectID()

	if student.Notas == nil {
		student.Notas = StudentNotes{}
	}
	if student.Frequencia == nil {
		student.Frequencia = AttendanceRecords{}
	}

	document := bson.M{
		"_id":        student.ID,
		"nome":       student.Nome,
		"notas":      student.Notas,
		"frequencia": student.Frequencia,
		"sala":       student.Sala,
		"role":       student.Role,
	}

	if strings.TrimSpace(student.ObservacoesGerais) != "" {
		document["observacoesGerais"] = strings.TrimSpace(student.ObservacoesGerais)
	}
	if len(student.HistoricoNotas) > 0 {
		document["historicoNotas"] = student.HistoricoNotas
	}

	_, err := r.collection.InsertOne(ctx, document)
	return err
}

func (r *StudentRepo) FindByClassroom(ctx context.Context, classroomID primitive.ObjectID) ([]Student, error) {
	opts := options.Find().SetSort(bson.M{"nome": 1})
	query := bson.M{
		"$or": []bson.M{
			{"sala": classroomID},
			{"sala": classroomID.Hex()},
		},
	}
	cur, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	students := make([]Student, 0)
	for cur.Next(ctx) {
		var student Student
		if err := cur.Decode(&student); err != nil {
			return nil, err
		}
		students = append(students, student)
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	return students, nil
}

func (r *StudentRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*Student, error) {
	var student Student
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&student)
	if err == mongo.ErrNoDocuments {
		return nil, ErrStudentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &student, nil
}

func (r *StudentRepo) Update(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	res, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrStudentNotFound
	}
	return nil
}

func (r *StudentRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrStudentNotFound
	}
	return nil
}
