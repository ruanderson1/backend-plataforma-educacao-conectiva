package classroom

import (
	"context"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type chatThreadRepo struct {
	collection *mongo.Collection
}

func NewChatThreadRepo(db *mongo.Database) ChatThreadRepository {
	return &chatThreadRepo{collection: db.Collection("chat_threads")}
}

func (r *chatThreadRepo) Create(ctx context.Context, thread *ChatThread) error {
	if thread.ID.IsZero() {
		thread.ID = primitive.NewObjectID()
	}
	thread.CreatedAt = thread.CreatedAt.UTC().Truncate(0)
	thread.UpdatedAt = thread.UpdatedAt.UTC().Truncate(0)
	_, err := r.collection.InsertOne(ctx, thread)
	return err
}

func (r *chatThreadRepo) FindByStudent(ctx context.Context, studentID string) ([]ChatThread, error) {
	filter := bson.M{"student_id": studentID}
	cur, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"updated_at": -1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	result := make([]ChatThread, 0)
	for cur.Next(ctx) {
		var item ChatThread
		if err := cur.Decode(&item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *chatThreadRepo) FindByResponsible(ctx context.Context, responsibleID string) ([]ChatThread, error) {
	filter := bson.M{"responsible_id": responsibleID}
	cur, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"updated_at": -1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	result := make([]ChatThread, 0)
	for cur.Next(ctx) {
		var item ChatThread
		if err := cur.Decode(&item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *chatThreadRepo) FindByTeacher(ctx context.Context, teacherID string) ([]ChatThread, error) {
	filter := bson.M{"teacher_id": teacherID}
	cur, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"updated_at": -1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	result := make([]ChatThread, 0)
	for cur.Next(ctx) {
		var item ChatThread
		if err := cur.Decode(&item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *chatThreadRepo) FindByClassroom(ctx context.Context, classroomID string) ([]ChatThread, error) {
	filter := bson.M{"classroom_id": classroomID}
	cur, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"updated_at": -1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	result := make([]ChatThread, 0)
	for cur.Next(ctx) {
		var item ChatThread
		if err := cur.Decode(&item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *chatThreadRepo) FindByID(ctx context.Context, threadID string) (*ChatThread, error) {
	id, err := primitive.ObjectIDFromHex(threadID)
	if err != nil {
		return nil, err
	}
	var result ChatThread
	err = r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *chatThreadRepo) FindByParticipants(ctx context.Context, studentID, responsibleID, teacherID, classroomID string) (*ChatThread, error) {
	filter := bson.M{
		"student_id":     strings.TrimSpace(studentID),
		"responsible_id": strings.TrimSpace(responsibleID),
		"teacher_id":     strings.TrimSpace(teacherID),
		"classroom_id":   strings.TrimSpace(classroomID),
	}

	var result ChatThread
	err := r.collection.FindOne(ctx, filter).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *chatThreadRepo) UpdateActivity(ctx context.Context, threadID string, message ChatMessage) error {
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(threadID))
	if err != nil {
		return err
	}

	createdAt := message.CreatedAt.UTC().Truncate(0)
	senderRole := strings.TrimSpace(message.SenderRole)
	update := bson.M{
		"updated_at":             createdAt,
		"last_message_body":      strings.TrimSpace(message.Body),
		"last_sender_role":       senderRole,
		"last_message_at":        createdAt,
		"unread_for_teacher":     senderRole == "responsavel",
		"unread_for_responsible": senderRole == "professor",
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *chatThreadRepo) MarkRead(ctx context.Context, threadID string, readerRole string) error {
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(threadID))
	if err != nil {
		return err
	}

	field := ""
	switch strings.TrimSpace(readerRole) {
	case "professor":
		field = "unread_for_teacher"
	case "responsavel":
		field = "unread_for_responsible"
	default:
		return nil
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{field: false}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

type chatMessageRepo struct {
	collection *mongo.Collection
}

func NewChatMessageRepo(db *mongo.Database) ChatMessageRepository {
	return &chatMessageRepo{collection: db.Collection("chat_messages")}
}

func (r *chatMessageRepo) Create(ctx context.Context, message *ChatMessage) error {
	if message.ID.IsZero() {
		message.ID = primitive.NewObjectID()
	}
	message.CreatedAt = message.CreatedAt.UTC().Truncate(0)
	_, err := r.collection.InsertOne(ctx, message)
	return err
}

func (r *chatMessageRepo) FindByThread(ctx context.Context, threadID string) ([]ChatMessage, error) {
	filter := bson.M{"thread_id": threadID}
	cur, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"created_at": 1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	result := make([]ChatMessage, 0)
	for cur.Next(ctx) {
		var item ChatMessage
		if err := cur.Decode(&item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

type chatResponsibleRepo struct {
	collection *mongo.Collection
}

type chatResponsibleDocument struct {
	UserID string                 `bson:"user_id"`
	Nome   string                 `bson:"nome"`
	Email  string                 `bson:"email"`
	Filhos []chatResponsibleChild `bson:"filhos"`
}

type chatResponsibleChild struct {
	ID   string `bson:"id"`
	Nome string `bson:"nome"`
}

func NewChatResponsibleRepo(db *mongo.Database) ResponsibleRepository {
	return &chatResponsibleRepo{collection: db.Collection("responsaveis")}
}

func (r *chatResponsibleRepo) FindByStudentIDs(ctx context.Context, studentIDs []string) ([]ChatResponsibleCandidate, error) {
	wanted := make(map[string]struct{}, len(studentIDs))
	normalizedIDs := make([]string, 0, len(studentIDs))
	for _, studentID := range studentIDs {
		studentID = strings.TrimSpace(studentID)
		if studentID == "" {
			continue
		}
		if _, exists := wanted[studentID]; exists {
			continue
		}
		wanted[studentID] = struct{}{}
		normalizedIDs = append(normalizedIDs, studentID)
	}

	if len(normalizedIDs) == 0 {
		return []ChatResponsibleCandidate{}, nil
	}

	cursor, err := r.collection.Find(ctx, bson.M{"filhos.id": bson.M{"$in": normalizedIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	candidates := make([]ChatResponsibleCandidate, 0)
	seen := map[string]struct{}{}
	for cursor.Next(ctx) {
		var doc chatResponsibleDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}

		responsibleID := strings.TrimSpace(doc.UserID)
		if responsibleID == "" {
			continue
		}

		for _, child := range doc.Filhos {
			studentID := strings.TrimSpace(child.ID)
			if _, ok := wanted[studentID]; !ok {
				continue
			}

			key := studentID + "|" + responsibleID
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}

			candidates = append(candidates, ChatResponsibleCandidate{
				StudentID:        studentID,
				StudentName:      strings.TrimSpace(child.Nome),
				ResponsibleID:    responsibleID,
				ResponsibleName:  strings.TrimSpace(doc.Nome),
				ResponsibleEmail: strings.TrimSpace(doc.Email),
			})
		}
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return candidates, nil
}
