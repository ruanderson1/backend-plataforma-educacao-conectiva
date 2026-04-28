package classroom_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// setupTestHandler agora está em report_handler_mock.go e usa o construtor correto

func TestCreateStudentObservation(t *testing.T) {
	h := setupTestHandler()
	body := map[string]interface{}{
		"student_id":           "aluno1",
		"periodo_referencia":   "2026-01",
		"observacao_professor": "Ótimo desempenho",
		"observacao_pais":      "Muito participativo",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/reports/student-observations", bytes.NewReader(b))
	w := httptest.NewRecorder()

	h.CreateStudentObservation(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("esperado 201, obteve %d", resp.StatusCode)
	}
}

func TestListStudentObservations(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/reports/student-observations?student_id=aluno1&periodo=2026-01", nil)
	w := httptest.NewRecorder()

	h.ListStudentObservations(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("esperado 200 ou 400, obteve %d", resp.StatusCode)
	}
}

func TestCreateStudentLLMReport(t *testing.T) {
	h := setupTestHandler()
	body := map[string]interface{}{
		"student_id":             "aluno1",
		"student_observation_id": "obs1",
		"periodo_referencia":     "2026-01",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/reports/student-llm-reports", bytes.NewReader(b))
	w := httptest.NewRecorder()

	h.CreateStudentLLMReport(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("esperado 201 ou 400, obteve %d", resp.StatusCode)
	}
}

func TestListStudentLLMReports(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/reports/student-llm-reports?student_id=aluno1&periodo=2026-01", nil)
	w := httptest.NewRecorder()

	h.ListStudentLLMReports(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("esperado 200 ou 400, obteve %d", resp.StatusCode)
	}
}

func TestCreateClassObservation(t *testing.T) {
	h := setupTestHandler()
	body := map[string]interface{}{
		"class_id":                   "turma1",
		"periodo_referencia":         "2026-01",
		"observacao_professor_turma": "Turma engajada",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/reports/class-observations", bytes.NewReader(b))
	w := httptest.NewRecorder()

	h.CreateClassObservation(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("esperado 201 ou 400, obteve %d", resp.StatusCode)
	}
}

func TestListClassObservations(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/reports/class-observations?class_id=turma1&periodo=2026-01", nil)
	w := httptest.NewRecorder()

	h.ListClassObservations(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("esperado 200 ou 400, obteve %d", resp.StatusCode)
	}
}

func TestCreateClassLLMReport(t *testing.T) {
	h := setupTestHandler()
	body := map[string]interface{}{
		"class_id":             "turma1",
		"class_observation_id": "obs1",
		"periodo_referencia":   "2026-01",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/reports/class-llm-reports", bytes.NewReader(b))
	w := httptest.NewRecorder()

	h.CreateClassLLMReport(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("esperado 201 ou 400, obteve %d", resp.StatusCode)
	}
}

func TestListClassLLMReports(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/reports/class-llm-reports?class_id=turma1&periodo=2026-01", nil)
	w := httptest.NewRecorder()

	h.ListClassLLMReports(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("esperado 200 ou 400, obteve %d", resp.StatusCode)
	}
}
