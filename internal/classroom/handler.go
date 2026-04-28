package classroom

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ClassroomService interface {
	Create(ctx context.Context, c *ClassRoom) error
	ListByTeacher(ctx context.Context, teacherId string, includeClosed bool) ([]map[string]interface{}, error)
	GetByID(ctx context.Context, id primitive.ObjectID, teacherId string) (map[string]interface{}, error)
	Update(ctx context.Context, id primitive.ObjectID, teacherId string, update map[string]interface{}) error
	Delete(ctx context.Context, id primitive.ObjectID, teacherId string) error
}

type Handler struct {
	service ClassroomService
}

// Funções utilitárias para resposta e contexto
func respondError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	w.Write([]byte(`{"error":"` + msg + `"}`))
}

func respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func getUserIDFromContext(ctx context.Context) string {
	userID, _ := ctx.Value("userID").(string)
	return userID
}

func getParam(r *http.Request, key string) string {
	if key == "id" {
		if v := r.Context().Value("id"); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

func NewHandler(service ClassroomService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		YearGrade string `json:"yearGrade"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.YearGrade) == "" {
		respondError(w, http.StatusBadRequest, "name and yearGrade are required")
		return
	}
	teacherId := getUserIDFromContext(r.Context())
	classRoom := &ClassRoom{
		Name:      req.Name,
		YearGrade: req.YearGrade,
		TeacherID: teacherId,
	}
	if err := h.service.Create(r.Context(), classRoom); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"id": classRoom.ID.Hex(), "accessCode": classRoom.AccessCode})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	teacherId := getUserIDFromContext(r.Context())
	includeClosed := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("includeClosed")), "true")
	classes, err := h.service.ListByTeacher(r.Context(), teacherId, includeClosed)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, classes)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := getParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	teacherId := getUserIDFromContext(r.Context())
	classRoom, err := h.service.GetByID(r.Context(), id, teacherId)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, classRoom)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := getParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if name, ok := req["name"]; ok && strings.TrimSpace(name.(string)) == "" {
		respondError(w, http.StatusBadRequest, "name cannot be empty")
		return
	}
	if yearGrade, ok := req["yearGrade"]; ok && strings.TrimSpace(yearGrade.(string)) == "" {
		respondError(w, http.StatusBadRequest, "yearGrade cannot be empty")
		return
	}
	teacherId := getUserIDFromContext(r.Context())
	if err := h.service.Update(r.Context(), id, teacherId, req); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := getParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	teacherId := getUserIDFromContext(r.Context())
	if err := h.service.Delete(r.Context(), id, teacherId); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}
