package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"plataforma/internal/classroom"
	"plataforma/internal/database"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const seedTimeout = 10 * time.Second

type seedStudent struct {
	name        string
	noteType    classroom.NoteType
	score       float64
	observacoes string
	role        string
	classroomID primitive.ObjectID
}

func main() {
	_ = godotenv.Load()

	mongoURI := strings.TrimSpace(os.Getenv("MONGO_URI"))
	if mongoURI == "" {
		log.Fatal("MONGO_URI is required")
	}

	mongoDB := strings.TrimSpace(os.Getenv("MONGO_DB"))
	if mongoDB == "" {
		mongoDB = "plataforma_educacao_conectiva"
	}

	ctx, cancel := context.WithTimeout(context.Background(), seedTimeout)
	defer cancel()

	client, err := database.ConnectMongo(ctx, mongoURI)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	db := client.Database(mongoDB)
	classroomIDs, err := resolveClassroomIDs(ctx, db)
	if err != nil {
		log.Fatal(err)
	}
	if len(classroomIDs) == 0 {
		log.Fatal("no classrooms found; set SEED_CLASSROOM_IDS")
	}

	studentRepo := classroom.NewStudentRepo(db)
	students := []seedStudent{
		{name: "Ana Souza", noteType: classroom.NoteTypeProva, score: 8.5, observacoes: "Bom desempenho", role: classroom.StudentRole, classroomID: classroomIDs[0]},
		{name: "Bruno Lima", noteType: classroom.NoteTypeTrabalho, score: 6.8, observacoes: "Precisa revisar conteudo", role: classroom.StudentRole, classroomID: classroomIDs[0]},
		{name: "Carlos Pereira", noteType: classroom.NoteTypeProjeto, score: 9.2, observacoes: "Excelente apresentacao", role: classroom.StudentRole, classroomID: classroomIDs[min(1, len(classroomIDs)-1)]},
	}

	inserted := 0
	for _, item := range students {
		student := &classroom.Student{
			Nome: item.name,
			Notas: classroom.StudentNotes{{
				Tipo:        item.noteType,
				Pontuacao:   item.score,
				Observacoes: item.observacoes,
			}},
			Sala: item.classroomID,
			Role: item.role,
		}

		alreadyExists, err := studentExists(ctx, db, student.Nome, student.Sala)
		if err != nil {
			log.Fatalf("failed to check existing student %q: %v", item.name, err)
		}
		if alreadyExists {
			fmt.Printf("skipped student %s (already exists in classroom %s)\n", student.Nome, student.Sala.Hex())
			continue
		}

		if err := studentRepo.Create(ctx, student); err != nil {
			log.Fatalf("failed to insert student %q: %v", item.name, err)
		}

		inserted++
		fmt.Printf("inserted student %s (%s)\n", student.Nome, student.ID.Hex())
	}

	fmt.Printf("done: inserted %d students across %d classroom(s)\n", inserted, len(classroomIDs))
}

func resolveClassroomIDs(ctx context.Context, db *mongo.Database) ([]primitive.ObjectID, error) {
	if raw := strings.TrimSpace(os.Getenv("SEED_CLASSROOM_IDS")); raw != "" {
		parts := strings.Split(raw, ",")
		ids := make([]primitive.ObjectID, 0, len(parts))
		for _, part := range parts {
			id, err := primitive.ObjectIDFromHex(strings.TrimSpace(part))
			if err != nil {
				return nil, fmt.Errorf("invalid SEED_CLASSROOM_IDS entry %q: %w", part, err)
			}
			ids = append(ids, id)
		}
		return ids, nil
	}

	collection := db.Collection("classrooms")
	cur, err := collection.Find(
		ctx,
		bson.M{},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	ids := make([]primitive.ObjectID, 0)
	for cur.Next(ctx) {
		var classroomDoc struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := cur.Decode(&classroomDoc); err != nil {
			return nil, err
		}
		ids = append(ids, classroomDoc.ID)
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}

func studentExists(ctx context.Context, db *mongo.Database, name string, classroomID primitive.ObjectID) (bool, error) {
	count, err := db.Collection("students").CountDocuments(ctx, bson.M{
		"nome": name,
		"$or": []bson.M{
			{"sala": classroomID},
			{"sala": classroomID.Hex()},
		},
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
