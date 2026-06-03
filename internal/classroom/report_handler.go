package classroom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// --- STUDENT OBSERVATIONS ---
// POST /api/reports/student-observations
func (h *ReportHandler) CreateStudentObservation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID           string `json:"student_id"`
		PeriodoReferencia   string `json:"periodo_referencia"`
		ObservacaoProfessor string `json:"observacao_professor"`
		ObservacaoPais      string `json:"observacao_pais"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.StudentID) == "" || strings.TrimSpace(req.PeriodoReferencia) == "" {
		respondError(w, http.StatusBadRequest, "student_id and periodo_referencia are required")
		return
	}
	obs := &StudentObservation{
		StudentID:           req.StudentID,
		PeriodoReferencia:   req.PeriodoReferencia,
		ObservacaoProfessor: req.ObservacaoProfessor,
		ObservacaoPais:      req.ObservacaoPais,
		CreatedAt:           time.Now(),
	}
	if err := h.studentObsRepo.Create(r.Context(), obs); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, obs)
}

// GET /api/reports/student-observations?student_id=...&periodo=...
func (h *ReportHandler) ListStudentObservations(w http.ResponseWriter, r *http.Request) {
	studentID := strings.TrimSpace(r.URL.Query().Get("student_id"))
	periodo := strings.TrimSpace(r.URL.Query().Get("periodo"))
	if studentID == "" {
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}
	obs, err := h.studentObsRepo.FindByStudentAndPeriod(r.Context(), studentID, periodo)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, obs)
}

// --- STUDENT LLM REPORTS ---
// POST /api/reports/student-llm-reports
// Espera: { "student_id": "...", "student_observation_id": "...", "periodo_referencia": "..." }
// O backend busca todos os dados do Student (notas, frequência) e observações, envia à LLM
func (h *ReportHandler) CreateStudentLLMReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID            string `json:"student_id"`
		StudentObservationID string `json:"student_observation_id"`
		PeriodoReferencia    string `json:"periodo_referencia"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.StudentID) == "" || strings.TrimSpace(req.StudentObservationID) == "" || strings.TrimSpace(req.PeriodoReferencia) == "" {
		respondError(w, http.StatusBadRequest, "student_id, student_observation_id and periodo_referencia are required")
		return
	}

	// 1. Buscar observações do aluno (colecionar todas as observações dele neste período)
	studentObsList, err := h.studentObsRepo.FindByStudentAndPeriod(r.Context(), req.StudentID, req.PeriodoReferencia)
	if err != nil || len(studentObsList) == 0 {
		respondError(w, http.StatusBadRequest, "Student observation not found")
		return
	}

	// 2. Buscar Student completo do banco com notas e frequência
	studentID, err := primitive.ObjectIDFromHex(req.StudentID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid student_id format")
		return
	}

	student, err := h.studentRepo.FindByID(r.Context(), studentID)
	if err != nil {
		respondError(w, http.StatusNotFound, "student not found")
		return
	}

	// 3. Montar payload COMPLETO com dados reais
	llmPayload := h.buildStudentLLMPayload(student, studentObsList, req.PeriodoReferencia)

	// 3. Chamar microserviço LLM
	llmURL := os.Getenv("LLM_SERVICE_URL")
	if llmURL == "" {
		llmURL = "http://localhost:8000"
	}
	llmURL = llmURL + "/reports/student"

	llmReqBody, _ := json.Marshal(llmPayload)
	fmt.Printf("[DEBUG] LLM PAYLOAD: %s\n", string(llmReqBody))
	httpReq, _ := http.NewRequest(http.MethodPost, llmURL, bytes.NewReader(llmReqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	llmResp, err := client.Do(httpReq)
	if err != nil {
		respondError(w, http.StatusBadGateway, fmt.Sprintf("Erro ao chamar LLM: %v", err))
		return
	}
	defer llmResp.Body.Close()

	respBody, _ := io.ReadAll(llmResp.Body)
	if llmResp.StatusCode != http.StatusOK {
		respondError(w, http.StatusBadGateway, fmt.Sprintf("LLM erro: %s - %s", llmResp.Status, string(respBody)))
		return
	}

	// 4. Mapear resposta aninhada e criar relatório
	var llmResult struct {
		Academico Academico `json:"academico"`
		Emocional Emocional `json:"emocional"`
		Risco     struct {
			RiscoDesempenhoBaixo string `json:"risco_desempenho_baixo"`
			RiscoDesengajamento  string `json:"risco_desengajamento"`
			NecessitaIntervencao bool   `json:"necessita_intervencao"`
		} `json:"risco"`
		SaidaLLM SaidaLLM `json:"saida_llm"`
	}
	if err := json.Unmarshal(respBody, &llmResult); err != nil {
		respondError(w, http.StatusBadGateway, "Resposta inválida da LLM")
		return
	}

	llmReport := &StudentLLMReport{
		StudentID:            req.StudentID,
		StudentObservationID: req.StudentObservationID,
		PeriodoReferencia:    req.PeriodoReferencia,
		CreatedAt:            time.Now(),
		Academico:            llmResult.Academico,
		Emocional:            llmResult.Emocional,
		Risco: Risco{
			RiscoDesempenhoBaixo: llmResult.Risco.RiscoDesempenhoBaixo,
			RiscoDesengajamento:  llmResult.Risco.RiscoDesengajamento,
			NecessitaIntervencao: fmt.Sprintf("%v", llmResult.Risco.NecessitaIntervencao),
		},
		SaidaLLM: llmResult.SaidaLLM,
	}

	if err := h.studentLLMRepo.Create(r.Context(), llmReport); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, llmReport)
}

// GET /api/reports/student-llm-reports?student_id=...&periodo=...
func (h *ReportHandler) ListStudentLLMReports(w http.ResponseWriter, r *http.Request) {
	studentID := strings.TrimSpace(r.URL.Query().Get("student_id"))
	periodo := strings.TrimSpace(r.URL.Query().Get("periodo"))
	if studentID == "" {
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}
	reports, err := h.studentLLMRepo.FindByStudentAndPeriod(r.Context(), studentID, periodo)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, reports)
}

// --- CLASS OBSERVATIONS ---
// POST /api/reports/class-observations
func (h *ReportHandler) CreateClassObservation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClassID                  string `json:"class_id"`
		PeriodoReferencia        string `json:"periodo_referencia"`
		ObservacaoProfessorTurma string `json:"observacao_professor_turma"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.ClassID) == "" || strings.TrimSpace(req.PeriodoReferencia) == "" {
		respondError(w, http.StatusBadRequest, "class_id and periodo_referencia are required")
		return
	}
	obs := &ClassObservation{
		ClassID:                  req.ClassID,
		PeriodoReferencia:        req.PeriodoReferencia,
		ObservacaoProfessorTurma: req.ObservacaoProfessorTurma,
		CreatedAt:                time.Now(),
	}
	if err := h.classObsRepo.Create(r.Context(), obs); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, obs)
}

// GET /api/reports/class-observations?class_id=...&periodo=...
func (h *ReportHandler) ListClassObservations(w http.ResponseWriter, r *http.Request) {
	classID := strings.TrimSpace(r.URL.Query().Get("class_id"))
	periodo := strings.TrimSpace(r.URL.Query().Get("periodo"))
	if classID == "" {
		respondError(w, http.StatusBadRequest, "class_id is required")
		return
	}
	obs, err := h.classObsRepo.FindByClassAndPeriod(r.Context(), classID, periodo)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, obs)
}

// --- CLASS LLM REPORTS ---
// POST /api/reports/class-llm-reports
func (h *ReportHandler) CreateClassLLMReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClassID            string `json:"class_id"`
		ClassObservationID string `json:"class_observation_id"`
		PeriodoReferencia  string `json:"periodo_referencia"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.ClassID) == "" || strings.TrimSpace(req.ClassObservationID) == "" || strings.TrimSpace(req.PeriodoReferencia) == "" {
		respondError(w, http.StatusBadRequest, "class_id, class_observation_id and periodo_referencia are required")
		return
	}

	classID, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.ClassID))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid class_id format")
		return
	}

	// 1. Buscar observação da turma
	classObsList, err := h.classObsRepo.FindByClassAndPeriod(r.Context(), req.ClassID, req.PeriodoReferencia)
	if err != nil || len(classObsList) == 0 {
		respondError(w, http.StatusBadRequest, "Class observation not found")
		return
	}
	classObs := classObsList[0]

	// 2. Buscar alunos reais da turma
	students, err := h.studentRepo.FindByClassroom(r.Context(), classID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(students) == 0 {
		respondError(w, http.StatusBadRequest, "class has no students")
		return
	}

	// 3. Montar lista de alunos para payload
	studentsList := buildClassStudentSummaries(students)

	// 4. Montar payload para LLM
	llmPayload := map[string]interface{}{
		"class_id":                   req.ClassID,
		"periodo_referencia":         req.PeriodoReferencia,
		"observacao_professor_turma": classObs.ObservacaoProfessorTurma,
		"students":                   studentsList,
	}

	// 5. Chamar microserviço LLM
	llmURL := os.Getenv("LLM_SERVICE_URL")
	if llmURL == "" {
		llmURL = "http://localhost:8000"
	}
	llmURL = llmURL + "/reports/class"

	llmReqBody, _ := json.Marshal(llmPayload)
	fmt.Printf("[INFO] class_llm_report.start class_id=%s periodo=%s students=%d\n", req.ClassID, req.PeriodoReferencia, len(studentsList))
	httpReq, _ := http.NewRequest(http.MethodPost, llmURL, bytes.NewReader(llmReqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	llmResp, err := client.Do(httpReq)
	if err != nil {
		respondError(w, http.StatusBadGateway, fmt.Sprintf("Erro ao chamar LLM: %v", err))
		return
	}
	defer llmResp.Body.Close()

	respBody, _ := io.ReadAll(llmResp.Body)
	fmt.Printf("[INFO] class_llm_report.llm_response class_id=%s status=%s bytes=%d\n", req.ClassID, llmResp.Status, len(respBody))
	if llmResp.StatusCode != http.StatusOK {
		respondError(w, http.StatusBadGateway, fmt.Sprintf("LLM erro: %s - %s", llmResp.Status, string(respBody)))
		return
	}

	var llmResult struct {
		AgregadoAlunos struct {
			DesempenhoMedioTurma        string `json:"desempenho_medio_turma"`
			PrincipaisDificuldadesTurma string `json:"principais_dificuldades_turma"`
			NivelEngajamentoTurma       string `json:"nivel_engajamento_turma"`
		} `json:"agregado_alunos"`
		RiscoColetivo struct {
			RiscoDesengajamentoTurma     string `json:"risco_desengajamento_turma"`
			NecessitaIntervencaoColetiva bool   `json:"necessita_intervencao_coletiva"`
		} `json:"risco_coletivo"`
		SaidaLLMTurma struct {
			ResumoLLMTurma                 string `json:"resumo_llm_turma"`
			RecomendacaoParaProfessorTurma string `json:"recomendacao_para_professor_turma"`
			PlanoAcaoTurma                 string `json:"plano_acao_turma"`
		} `json:"saida_llm_turma"`
	}
	if err := json.Unmarshal(respBody, &llmResult); err != nil {
		fmt.Printf("[ERROR] class_llm_report.invalid_json class_id=%s err=%v\n", req.ClassID, err)
		respondError(w, http.StatusBadGateway, "Resposta inválida da LLM")
		return
	}

	// 6. Mapear resposta da LLM e criar relatório
	llmReport := &ClassLLMReport{
		ClassID:            req.ClassID,
		ClassObservationID: req.ClassObservationID,
		PeriodoReferencia:  req.PeriodoReferencia,
		CreatedAt:          time.Now(),
		AgregadoAlunos: AgregadoAlunos{
			DesempenhoMedioTurma:        llmResult.AgregadoAlunos.DesempenhoMedioTurma,
			PrincipaisDificuldadesTurma: llmResult.AgregadoAlunos.PrincipaisDificuldadesTurma,
			NivelEngajamentoTurma:       llmResult.AgregadoAlunos.NivelEngajamentoTurma,
		},
		RiscoColetivo: RiscoColetivo{
			RiscoDesengajamentoTurma:     llmResult.RiscoColetivo.RiscoDesengajamentoTurma,
			NecessitaIntervencaoColetiva: fmt.Sprintf("%v", llmResult.RiscoColetivo.NecessitaIntervencaoColetiva),
		},
		SaidaLLMTurma: SaidaLLMTurma{
			ResumoLLMTurma:                 llmResult.SaidaLLMTurma.ResumoLLMTurma,
			RecomendacaoParaProfessorTurma: llmResult.SaidaLLMTurma.RecomendacaoParaProfessorTurma,
			PlanoAcaoTurma:                 llmResult.SaidaLLMTurma.PlanoAcaoTurma,
		},
	}

	if err := h.classLLMRepo.Create(r.Context(), llmReport); err != nil {
		fmt.Printf("[ERROR] class_llm_report.persist_failed class_id=%s err=%v\n", req.ClassID, err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	fmt.Printf("[INFO] class_llm_report.success class_id=%s periodo=%s\n", req.ClassID, req.PeriodoReferencia)
	respondJSON(w, http.StatusCreated, llmReport)
}

type classStudentSummary struct {
	StudentID                string `json:"student_id"`
	DesempenhoGeral          string `json:"desempenho_geral"`
	Engajamento              string `json:"engajamento"`
	RiscoDesengajamento      string `json:"risco_desengajamento"`
	DificuldadesAprendizagem string `json:"dificuldades_aprendizagem"`
}

func buildClassStudentSummaries(students []Student) []classStudentSummary {
	summaries := make([]classStudentSummary, 0, len(students))
	for _, student := range students {
		desempenho := summarizePerformance(student.Notas)
		engajamento := summarizeEngagement(student.Frequencia)
		risco := summarizeClassRisk(desempenho, engajamento)
		dificuldades := strings.TrimSpace(student.ObservacoesGerais)
		if dificuldades == "" {
			dificuldades = "sem observacoes gerais registradas"
		}
		summaries = append(summaries, classStudentSummary{
			StudentID:                student.ID.Hex(),
			DesempenhoGeral:          desempenho,
			Engajamento:              engajamento,
			RiscoDesengajamento:      risco,
			DificuldadesAprendizagem: dificuldades,
		})
	}
	return summaries
}

func summarizePerformance(notes StudentNotes) string {
	if len(notes) == 0 {
		return "medio"
	}

	var totalRatio float64
	for _, note := range notes {
		maxScore := note.NotaMaxima
		if maxScore <= 0 {
			maxScore = 10
		}
		totalRatio += note.Pontuacao / maxScore
	}
	averageRatio := totalRatio / float64(len(notes))
	switch {
	case averageRatio >= 0.8:
		return "alto"
	case averageRatio >= 0.6:
		return "medio"
	default:
		return "baixo"
	}
}

func summarizeEngagement(records AttendanceRecords) string {
	if len(records) == 0 {
		return "medio"
	}

	var presencas int
	var totalRegistros int
	for _, record := range records {
		switch strings.ToLower(strings.TrimSpace(record.Tipo)) {
		case string(AttendancePresenca):
			presencas++
			totalRegistros++
		case string(AttendanceFalta), string(AttendanceFaltaJustificada):
			totalRegistros++
		}
	}
	if totalRegistros == 0 {
		return "medio"
	}

	presenceRate := float64(presencas) / float64(totalRegistros)
	switch {
	case presenceRate >= 0.8:
		return "alto"
	case presenceRate >= 0.6:
		return "medio"
	default:
		return "baixo"
	}
}

func summarizeClassRisk(desempenho string, engajamento string) string {
	if desempenho == "baixo" && engajamento == "baixo" {
		return "alto"
	}
	if desempenho == "baixo" || engajamento == "baixo" {
		return "medio"
	}
	return "baixo"
}

// GET /api/reports/class-llm-reports?class_id=...&periodo=...
func (h *ReportHandler) ListClassLLMReports(w http.ResponseWriter, r *http.Request) {
	classID := strings.TrimSpace(r.URL.Query().Get("class_id"))
	periodo := strings.TrimSpace(r.URL.Query().Get("periodo"))
	if classID == "" {
		respondError(w, http.StatusBadRequest, "class_id is required")
		return
	}
	reports, err := h.classLLMRepo.FindByClassAndPeriod(r.Context(), classID, periodo)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, reports)
}

type ReportHandler struct {
	studentRepo    StudentRepository
	studentObsRepo StudentObservationRepository
	studentLLMRepo StudentLLMReportRepository
	classObsRepo   ClassObservationRepository
	classLLMRepo   ClassLLMReportRepository
}

func NewReportHandler(studentRepo StudentRepository, studentObsRepo StudentObservationRepository, studentLLMRepo StudentLLMReportRepository, classObsRepo ClassObservationRepository, classLLMRepo ClassLLMReportRepository) *ReportHandler {
	return &ReportHandler{
		studentRepo:    studentRepo,
		studentObsRepo: studentObsRepo,
		studentLLMRepo: studentLLMRepo,
		classObsRepo:   classObsRepo,
		classLLMRepo:   classLLMRepo,
	}
}

// buildStudentLLMPayload constrói o payload completo para enviar à LLM
// Inclui dados reais do Student: notas, frequência, observações
func (h *ReportHandler) buildStudentLLMPayload(student *Student, observations []StudentObservation, periodo string) map[string]interface{} {
	// Calcular estatísticas de notas
	var totalPoints float64
	var noteCount int
	var faltas int
	var presencas int

	for _, note := range student.Notas {
		totalPoints += note.Pontuacao
		noteCount++
	}

	for _, freq := range student.Frequencia {
		if freq.Tipo == "falta" || freq.Tipo == "falta_justificada" {
			faltas++
		} else if freq.Tipo == "presenca" {
			presencas++
		}
	}

	media := 0.0
	if noteCount > 0 {
		media = totalPoints / float64(noteCount)
	}

	// Compilar observações (professor + pais)
	observacoesProfessor := ""
	observacoesPais := ""
	for _, obs := range observations {
		if obs.ObservacaoProfessor != "" {
			observacoesProfessor += obs.ObservacaoProfessor + " | "
		}
		if obs.ObservacaoPais != "" {
			observacoesPais += obs.ObservacaoPais + " | "
		}
	}

	// Montar payload COMPLETO com dados reais
	// Garante que detalhes de frequência nunca seja null
	freqDetalhes := student.Frequencia
	if freqDetalhes == nil {
		freqDetalhes = make([]AttendanceRecord, 0)
	}
	return map[string]interface{}{
		"student_id":         student.ID.Hex(),
		"nome":               student.Nome,
		"periodo_referencia": periodo,
		"notas": map[string]interface{}{
			"total":    noteCount,
			"media":    media,
			"detalhes": student.Notas,
		},
		"frequencia": map[string]interface{}{
			"total_presencas": presencas,
			"total_faltas":    faltas,
			"total_registros": presencas + faltas,
			"detalhes":        freqDetalhes,
		},
		"observacoes": map[string]interface{}{
			"professor": observacoesProfessor,
			"pais":      observacoesPais,
			"geral":     student.ObservacoesGerais,
		},
	}
}

// Helper functions para mapear resposta da LLM
func toString(val interface{}) string {
	if val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", val)
}

func toBool(val interface{}) bool {
	if val == nil {
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return false
}

// Exemplos de endpoints a serem implementados:
// POST /api/reports/student-observations
// GET /api/reports/student-observations?student_id=...&periodo=...
// POST /api/reports/student-llm-reports
// GET /api/reports/student-llm-reports?student_id=...&periodo=...
// POST /api/reports/class-observations
// GET /api/reports/class-observations?class_id=...&periodo=...
// POST /api/reports/class-llm-reports
// GET /api/reports/class-llm-reports?class_id=...&periodo=...

// Implemente os handlers conforme o padrão do projeto.
