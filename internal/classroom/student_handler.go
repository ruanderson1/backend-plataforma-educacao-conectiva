package classroom

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StudentService interface {
	CreateStudent(ctx context.Context, teacherId string, student *Student) (map[string]interface{}, error)
	ListStudentsByClassroom(ctx context.Context, teacherId string, classroomID primitive.ObjectID) ([]map[string]interface{}, error)
	GetStudentByID(ctx context.Context, teacherId string, id primitive.ObjectID) (map[string]interface{}, error)
	UpdateStudent(ctx context.Context, teacherId string, id primitive.ObjectID, update map[string]interface{}) error
	DeleteStudent(ctx context.Context, teacherId string, id primitive.ObjectID) error
}

type StudentHandler struct {
	service StudentService
}

func NewStudentHandler(service StudentService) *StudentHandler {
	return &StudentHandler{service: service}
}

func (h *StudentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Nome              string            `json:"nome"`
		Notas             StudentNote       `json:"notas"`
		ObservacoesGerais string            `json:"observacoesGerais"`
		Frequencia        AttendanceRecords `json:"frequencia"`
		Sala              string            `json:"sala"`
		Role              string            `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Nome) == "" || strings.TrimSpace(req.Sala) == "" {
		respondError(w, http.StatusBadRequest, "nome and sala are required")
		return
	}

	salaID, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Sala))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid sala")
		return
	}

	if req.Notas.Tipo != "" && !IsValidNoteType(req.Notas.Tipo) {
		respondError(w, http.StatusBadRequest, ErrInvalidNoteType.Error())
		return
	}
	req.Notas.Tipo = NormalizeNoteType(string(req.Notas.Tipo))
	req.Notas.NotaMaxima = NormalizeNoteMaxScore(req.Notas.NotaMaxima)
	req.Notas.Peso = NormalizeNoteWeight(req.Notas.Peso)

	notas := StudentNotes{}
	if req.Notas.Tipo != "" || req.Notas.Pontuacao != 0 || req.Notas.NotaMaxima != 0 || strings.TrimSpace(req.Notas.Observacoes) != "" {
		if req.Notas.Pontuacao > req.Notas.NotaMaxima {
			respondError(w, http.StatusBadRequest, "pontuacao cannot be greater than notaMaxima")
			return
		}
		notas = append(notas, req.Notas)
	}

	frequencia := AttendanceRecords{}
	if len(req.Frequencia) > 0 {
		frequencia = append(frequencia, req.Frequencia...)
	}

	student := &Student{
		Nome:              req.Nome,
		Notas:             notas,
		ObservacoesGerais: strings.TrimSpace(req.ObservacoesGerais),
		Frequencia:        frequencia,
		Sala:              salaID,
		Role:              strings.TrimSpace(req.Role),
	}

	teacherID := getUserIDFromContext(r.Context())
	payload, err := h.service.CreateStudent(r.Context(), teacherID, student)
	if err != nil {
		h.writeStudentServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, payload)
}

func (h *StudentHandler) ListByClassroom(w http.ResponseWriter, r *http.Request) {
	salaParam := strings.TrimSpace(r.URL.Query().Get("sala"))
	if salaParam == "" {
		respondError(w, http.StatusBadRequest, "sala query param is required")
		return
	}

	salaID, err := primitive.ObjectIDFromHex(salaParam)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid sala")
		return
	}

	teacherID := getUserIDFromContext(r.Context())
	students, err := h.service.ListStudentsByClassroom(r.Context(), teacherID, salaID)
	if err != nil {
		h.writeStudentServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, students)
}

func (h *StudentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr, _ := r.Context().Value("studentID").(string)
	studentID, err := primitive.ObjectIDFromHex(strings.TrimSpace(idStr))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	teacherID := getUserIDFromContext(r.Context())
	student, err := h.service.GetStudentByID(r.Context(), teacherID, studentID)
	if err != nil {
		h.writeStudentServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, student)
}

func (h *StudentHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr, _ := r.Context().Value("studentID").(string)
	studentID, err := primitive.ObjectIDFromHex(strings.TrimSpace(idStr))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if nomeRaw, ok := req["nome"]; ok {
		nome, ok := nomeRaw.(string)
		if !ok || strings.TrimSpace(nome) == "" {
			respondError(w, http.StatusBadRequest, "nome cannot be empty")
			return
		}
	}

	teacherID := getUserIDFromContext(r.Context())
	if err := h.service.UpdateStudent(r.Context(), teacherID, studentID, req); err != nil {
		h.writeStudentServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

func (h *StudentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr, _ := r.Context().Value("studentID").(string)
	studentID, err := primitive.ObjectIDFromHex(strings.TrimSpace(idStr))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	teacherID := getUserIDFromContext(r.Context())
	if err := h.service.DeleteStudent(r.Context(), teacherID, studentID); err != nil {
		h.writeStudentServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *StudentHandler) writeStudentServiceError(w http.ResponseWriter, err error) {
	switch err {
	case ErrInvalidStudent, ErrClassroomMissing, ErrInvalidNoteType, ErrInvalidAttendance:
		respondError(w, http.StatusBadRequest, err.Error())
	case ErrNotFound, ErrStudentNotFound:
		respondError(w, http.StatusNotFound, err.Error())
	default:
		respondError(w, http.StatusInternalServerError, err.Error())
	}
}
