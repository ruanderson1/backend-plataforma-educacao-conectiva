package classroom

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatHandler struct {
	threadRepo      ChatThreadRepository
	messageRepo     ChatMessageRepository
	studentRepo     StudentRepository
	classroomRepo   ClassroomRepository
	responsibleRepo ResponsibleRepository
}

func NewChatHandler(threadRepo ChatThreadRepository, messageRepo ChatMessageRepository, studentRepo StudentRepository, classroomRepo ClassroomRepository, responsibleRepo ResponsibleRepository) *ChatHandler {
	return &ChatHandler{
		threadRepo:      threadRepo,
		messageRepo:     messageRepo,
		studentRepo:     studentRepo,
		classroomRepo:   classroomRepo,
		responsibleRepo: responsibleRepo,
	}
}

func (h *ChatHandler) ListThreads(w http.ResponseWriter, r *http.Request) {
	studentID := strings.TrimSpace(r.URL.Query().Get("student_id"))
	classroomID := strings.TrimSpace(r.URL.Query().Get("classroom_id"))
	userID := strings.TrimSpace(getUserIDFromContext(r.Context()))

	if studentID != "" {
		threads, err := h.threadRepo.FindByStudent(r.Context(), studentID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, threads)
		return
	}

	if classroomID != "" {
		threadsByClass, err := h.listThreadsByClassroom(r, classroomID, userID)
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "invalid classroom_id") {
				status = http.StatusBadRequest
			}
			if strings.Contains(err.Error(), "classroom not found") {
				status = http.StatusNotFound
			}
			if strings.Contains(err.Error(), "unauthorized") {
				status = http.StatusUnauthorized
			}
			respondError(w, status, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, threadsByClass)
		return
	}

	if userID != "" {
		threadsR, err := h.threadRepo.FindByResponsible(r.Context(), userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if len(threadsR) > 0 {
			respondJSON(w, http.StatusOK, threadsR)
			return
		}

		threadsT, err := h.threadRepo.FindByTeacher(r.Context(), userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, threadsT)
		return
	}

	respondError(w, http.StatusBadRequest, "student_id, classroom_id or authenticated user is required")
}

func (h *ChatHandler) listThreadsByClassroom(r *http.Request, classroomID string, userID string) ([]ChatThreadListItem, error) {
	classroomObjectID, err := primitive.ObjectIDFromHex(classroomID)
	if err != nil {
		return nil, errInvalidChatClassroomID()
	}

	classroom, err := h.classroomRepo.FindByIDAny(r.Context(), classroomObjectID)
	if err != nil || classroom == nil {
		return nil, errChatClassroomNotFound()
	}

	if strings.TrimSpace(userID) != "" && strings.TrimSpace(classroom.TeacherID) != "" && strings.TrimSpace(userID) != strings.TrimSpace(classroom.TeacherID) {
		return nil, errChatUnauthorized()
	}

	students, err := h.studentRepo.FindByClassroom(r.Context(), classroomObjectID)
	if err != nil {
		return nil, err
	}

	studentIDs := make([]string, 0, len(students))
	studentNamesByID := make(map[string]string, len(students))
	for _, student := range students {
		studentID := student.ID.Hex()
		studentIDs = append(studentIDs, studentID)
		studentNamesByID[studentID] = strings.TrimSpace(student.Nome)
	}

	candidates := []ChatResponsibleCandidate{}
	if h.responsibleRepo != nil {
		candidates, err = h.responsibleRepo.FindByStudentIDs(r.Context(), studentIDs)
		if err != nil {
			return nil, err
		}
	}

	threads, err := h.threadRepo.FindByClassroom(r.Context(), classroomID)
	if err != nil {
		return nil, err
	}

	itemsByKey := make(map[string]ChatThreadListItem, len(threads)+len(candidates))
	for _, thread := range threads {
		item := chatThreadToListItem(thread)
		item.StudentName = studentNamesByID[item.StudentID]
		itemsByKey[chatParticipantKey(item.StudentID, item.ResponsibleID)] = item
	}

	for _, candidate := range candidates {
		key := chatParticipantKey(candidate.StudentID, candidate.ResponsibleID)
		existing, hasExistingThread := itemsByKey[key]
		if hasExistingThread {
			if strings.TrimSpace(existing.StudentName) == "" {
				existing.StudentName = firstNonBlank(candidate.StudentName, studentNamesByID[candidate.StudentID])
			}
			if strings.TrimSpace(existing.ResponsibleName) == "" {
				existing.ResponsibleName = candidate.ResponsibleName
			}
			if strings.TrimSpace(existing.ResponsibleEmail) == "" {
				existing.ResponsibleEmail = candidate.ResponsibleEmail
			}
			itemsByKey[key] = existing
			continue
		}

		now := time.Now()
		thread := &ChatThread{
			StudentID:     candidate.StudentID,
			ResponsibleID: candidate.ResponsibleID,
			TeacherID:     classroom.TeacherID,
			ClassroomID:   classroomID,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := h.threadRepo.Create(r.Context(), thread); err != nil {
			return nil, err
		}

		itemsByKey[key] = ChatThreadListItem{
			ID:               thread.ID.Hex(),
			StudentID:        candidate.StudentID,
			StudentName:      firstNonBlank(candidate.StudentName, studentNamesByID[candidate.StudentID]),
			ResponsibleID:    candidate.ResponsibleID,
			ResponsibleName:  candidate.ResponsibleName,
			ResponsibleEmail: candidate.ResponsibleEmail,
			TeacherID:        classroom.TeacherID,
			ClassroomID:      classroomID,
			CreatedAt:        thread.CreatedAt,
			UpdatedAt:        thread.UpdatedAt,
			HasThread:        true,
		}
	}

	items := make([]ChatThreadListItem, 0, len(itemsByKey))
	for _, item := range itemsByKey {
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].UnreadForTeacher != items[j].UnreadForTeacher {
			return items[i].UnreadForTeacher
		}
		if items[i].LastMessageAt != nil || items[j].LastMessageAt != nil {
			if items[i].LastMessageAt == nil {
				return false
			}
			if items[j].LastMessageAt == nil {
				return true
			}
			if !items[i].LastMessageAt.Equal(*items[j].LastMessageAt) {
				return items[i].LastMessageAt.After(*items[j].LastMessageAt)
			}
		}
		leftStudent := strings.ToLower(firstNonBlank(items[i].StudentName, items[i].StudentID))
		rightStudent := strings.ToLower(firstNonBlank(items[j].StudentName, items[j].StudentID))
		if leftStudent == rightStudent {
			leftResponsible := strings.ToLower(firstNonBlank(items[i].ResponsibleName, items[i].ResponsibleID))
			rightResponsible := strings.ToLower(firstNonBlank(items[j].ResponsibleName, items[j].ResponsibleID))
			return leftResponsible < rightResponsible
		}
		return leftStudent < rightStudent
	})

	return items, nil
}

func (h *ChatHandler) CreateThread(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID     string `json:"student_id"`
		TeacherID     string `json:"teacher_id"`
		ClassroomID   string `json:"classroom_id"`
		ResponsibleID string `json:"responsible_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.StudentID) == "" {
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}

	userID := strings.TrimSpace(getUserIDFromContext(r.Context()))
	studentID := strings.TrimSpace(req.StudentID)
	studentObjectID, err := primitive.ObjectIDFromHex(studentID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid student_id format")
		return
	}
	student, err := h.studentRepo.FindByID(r.Context(), studentObjectID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "student not found")
		return
	}

	classroomID := strings.TrimSpace(req.ClassroomID)
	if classroomID == "" {
		if student.Sala.IsZero() {
			respondError(w, http.StatusBadRequest, "classroom_id is required")
			return
		}
		classroomID = student.Sala.Hex()
	}

	if strings.TrimSpace(req.ResponsibleID) == "" {
		req.ResponsibleID = userID
	}
	if strings.TrimSpace(req.TeacherID) == "" {
		classroom, err := h.classroomRepo.FindByIDAny(r.Context(), student.Sala)
		if err != nil || classroom == nil {
			respondError(w, http.StatusBadRequest, "classroom not found")
			return
		}
		req.TeacherID = classroom.TeacherID
	}

	if strings.TrimSpace(req.ResponsibleID) == "" || strings.TrimSpace(req.TeacherID) == "" || strings.TrimSpace(classroomID) == "" {
		respondError(w, http.StatusBadRequest, "responsible_id, teacher_id and classroom_id are required")
		return
	}

	existingThread, err := h.threadRepo.FindByParticipants(r.Context(), studentID, req.ResponsibleID, req.TeacherID, classroomID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existingThread != nil {
		respondJSON(w, http.StatusOK, existingThread)
		return
	}

	thread := &ChatThread{
		StudentID:     studentID,
		ResponsibleID: strings.TrimSpace(req.ResponsibleID),
		TeacherID:     strings.TrimSpace(req.TeacherID),
		ClassroomID:   strings.TrimSpace(classroomID),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := h.threadRepo.Create(r.Context(), thread); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, thread)
}

func (h *ChatHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	threadID := strings.TrimSpace(r.URL.Query().Get("thread_id"))
	if threadID == "" {
		respondError(w, http.StatusBadRequest, "thread_id is required")
		return
	}

	thread, err := h.threadRepo.FindByID(r.Context(), threadID)
	if err != nil {
		respondError(w, http.StatusNotFound, "thread not found")
		return
	}

	readerRole := ""
	userID := strings.TrimSpace(getUserIDFromContext(r.Context()))
	if userID != "" {
		var belongs bool
		readerRole, belongs = chatParticipantRole(thread, userID)
		if !belongs {
			respondError(w, http.StatusUnauthorized, "user does not belong to this thread")
			return
		}
	}

	messages, err := h.messageRepo.FindByThread(r.Context(), threadID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if readerRole != "" {
		if err := h.threadRepo.MarkRead(r.Context(), threadID, readerRole); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	respondJSON(w, http.StatusOK, messages)
}

func (h *ChatHandler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ThreadID   string `json:"thread_id"`
		Body       string `json:"body"`
		SenderID   string `json:"sender_id"`
		SenderRole string `json:"sender_role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.ThreadID) == "" || strings.TrimSpace(req.Body) == "" {
		respondError(w, http.StatusBadRequest, "thread_id and body are required")
		return
	}

	thread, err := h.threadRepo.FindByID(r.Context(), req.ThreadID)
	if err != nil {
		respondError(w, http.StatusNotFound, "thread not found")
		return
	}

	userID := strings.TrimSpace(getUserIDFromContext(r.Context()))
	senderID := strings.TrimSpace(req.SenderID)
	if userID != "" {
		senderID = userID
	}

	senderRole, belongs := chatParticipantRole(thread, senderID)
	if !belongs {
		respondError(w, http.StatusUnauthorized, "sender does not belong to this thread")
		return
	}

	requestedRole := strings.TrimSpace(req.SenderRole)
	if requestedRole != "" && requestedRole != senderRole {
		respondError(w, http.StatusUnauthorized, "sender role does not match authenticated user")
		return
	}

	message := &ChatMessage{
		ThreadID:   thread.ID.Hex(),
		SenderID:   senderID,
		SenderRole: senderRole,
		Body:       strings.TrimSpace(req.Body),
		CreatedAt:  time.Now(),
	}

	if err := h.messageRepo.Create(r.Context(), message); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.threadRepo.UpdateActivity(r.Context(), thread.ID.Hex(), *message); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, message)
}

func chatThreadToListItem(thread ChatThread) ChatThreadListItem {
	return ChatThreadListItem{
		ID:                   thread.ID.Hex(),
		StudentID:            strings.TrimSpace(thread.StudentID),
		ResponsibleID:        strings.TrimSpace(thread.ResponsibleID),
		TeacherID:            strings.TrimSpace(thread.TeacherID),
		ClassroomID:          strings.TrimSpace(thread.ClassroomID),
		LastMessageBody:      strings.TrimSpace(thread.LastMessageBody),
		LastSenderRole:       strings.TrimSpace(thread.LastSenderRole),
		LastMessageAt:        thread.LastMessageAt,
		UnreadForTeacher:     thread.UnreadForTeacher,
		UnreadForResponsible: thread.UnreadForResponsible,
		CreatedAt:            thread.CreatedAt,
		UpdatedAt:            thread.UpdatedAt,
		HasThread:            true,
	}
}

func chatParticipantRole(thread *ChatThread, userID string) (string, bool) {
	if thread == nil {
		return "", false
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", false
	}

	if userID == strings.TrimSpace(thread.TeacherID) {
		return "professor", true
	}
	if userID == strings.TrimSpace(thread.ResponsibleID) {
		return "responsavel", true
	}

	return "", false
}

func chatParticipantKey(studentID string, responsibleID string) string {
	return strings.TrimSpace(studentID) + "|" + strings.TrimSpace(responsibleID)
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

type chatHandlerError string

func (e chatHandlerError) Error() string {
	return string(e)
}

func errInvalidChatClassroomID() error {
	return chatHandlerError("invalid classroom_id")
}

func errChatClassroomNotFound() error {
	return chatHandlerError("classroom not found")
}

func errChatUnauthorized() error {
	return chatHandlerError("unauthorized")
}
