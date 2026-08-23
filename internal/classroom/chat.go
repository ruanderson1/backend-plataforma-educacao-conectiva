package classroom

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChatThread representa o "canal" entre um responsável, professor e aluno.
type ChatThread struct {
	ID                   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StudentID            string             `bson:"student_id" json:"student_id"`
	ResponsibleID        string             `bson:"responsible_id" json:"responsible_id"`
	TeacherID            string             `bson:"teacher_id" json:"teacher_id"`
	ClassroomID          string             `bson:"classroom_id" json:"classroom_id"`
	LastMessageBody      string             `bson:"last_message_body,omitempty" json:"last_message_body,omitempty"`
	LastSenderRole       string             `bson:"last_sender_role,omitempty" json:"last_sender_role,omitempty"`
	LastMessageAt        *time.Time         `bson:"last_message_at,omitempty" json:"last_message_at,omitempty"`
	UnreadForTeacher     bool               `bson:"unread_for_teacher,omitempty" json:"unread_for_teacher"`
	UnreadForResponsible bool               `bson:"unread_for_responsible,omitempty" json:"unread_for_responsible"`
	CreatedAt            time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt            time.Time          `bson:"updated_at" json:"updated_at"`
}

// ChatMessage representa a mensagem do chat entre professor e responsável.
type ChatMessage struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ThreadID   string             `bson:"thread_id" json:"thread_id"`
	SenderID   string             `bson:"sender_id" json:"sender_id"`
	SenderRole string             `bson:"sender_role" json:"sender_role"`
	Body       string             `bson:"body" json:"body"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
}

type ChatThreadListItem struct {
	ID                   string     `json:"id"`
	StudentID            string     `json:"student_id"`
	StudentName          string     `json:"student_name,omitempty"`
	ResponsibleID        string     `json:"responsible_id"`
	ResponsibleName      string     `json:"responsible_name,omitempty"`
	ResponsibleEmail     string     `json:"responsible_email,omitempty"`
	TeacherID            string     `json:"teacher_id"`
	ClassroomID          string     `json:"classroom_id"`
	LastMessageBody      string     `json:"last_message_body,omitempty"`
	LastSenderRole       string     `json:"last_sender_role,omitempty"`
	LastMessageAt        *time.Time `json:"last_message_at,omitempty"`
	UnreadForTeacher     bool       `json:"unread_for_teacher"`
	UnreadForResponsible bool       `json:"unread_for_responsible"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	HasThread            bool       `json:"has_thread"`
}

type ChatResponsibleCandidate struct {
	StudentID        string
	StudentName      string
	ResponsibleID    string
	ResponsibleName  string
	ResponsibleEmail string
}

type ClassroomRepository interface {
	FindByIDAny(ctx context.Context, id primitive.ObjectID) (*ClassRoom, error)
}

type ChatThreadRepository interface {
	Create(ctx context.Context, thread *ChatThread) error
	FindByStudent(ctx context.Context, studentID string) ([]ChatThread, error)
	FindByResponsible(ctx context.Context, responsibleID string) ([]ChatThread, error)
	FindByTeacher(ctx context.Context, teacherID string) ([]ChatThread, error)
	FindByClassroom(ctx context.Context, classroomID string) ([]ChatThread, error)
	FindByID(ctx context.Context, threadID string) (*ChatThread, error)
	FindByParticipants(ctx context.Context, studentID, responsibleID, teacherID, classroomID string) (*ChatThread, error)
	UpdateActivity(ctx context.Context, threadID string, message ChatMessage) error
	MarkRead(ctx context.Context, threadID string, readerRole string) error
}

type ChatMessageRepository interface {
	Create(ctx context.Context, message *ChatMessage) error
	FindByThread(ctx context.Context, threadID string) ([]ChatMessage, error)
}

type ResponsibleRepository interface {
	FindByStudentIDs(ctx context.Context, studentIDs []string) ([]ChatResponsibleCandidate, error)
}
