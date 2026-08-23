package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"plataforma/cmd/web/handlers"
	"plataforma/internal/auth"
	"plataforma/internal/classroom"
	"plataforma/internal/database"
	"plataforma/internal/users"

	"go.mongodb.org/mongo-driver/mongo"
)

type application struct {
	authHandler      *handlers.AuthHandler
	classroomHandler *handlers.ClassroomHandler
	profileHandler   *handlers.ProfileHandler
	reportHandler    *handlers.ReportHandler
	chatHandler      *classroom.ChatHandler
}

const (
	defaultServerAddr = ":4000"
	defaultMongoDB    = "plataforma_educacao_conectiva"
	mongoTimeout      = 10 * time.Second
	reportTimeout     = 120 * time.Second
)

func main() {
	// Carrega variáveis do .env se existir
	_ = godotenv.Load()

	// Carrega as configurações principais com fallback para valores padrão.
	mongoURI := strings.TrimSpace(os.Getenv("MONGO_URI"))
	if mongoURI == "" {
		log.Fatal("MONGO_URI is required")
	}

	mongoDB := strings.TrimSpace(os.Getenv("MONGO_DB"))
	if mongoDB == "" {
		mongoDB = defaultMongoDB
	}

	serverAddr := strings.TrimSpace(os.Getenv("SERVER_ADDR"))
	if serverAddr == "" {
		serverAddr = defaultServerAddr
	}

	// Inicializa conexões externas e garante encerramento limpo.
	client := mustConnectMongo(mongoURI)
	defer disconnectMongo(client)

	// Monta as dependências da aplicação e inicia o servidor HTTP.
	authService, err := auth.NewService(client.Database(mongoDB))
	if err != nil {
		log.Fatal(err)
	}

	// Inicializa dependências classroom
	classroomRepo := classroom.NewRepository(client.Database(mongoDB))
	studentRepo := classroom.NewStudentRepo(client.Database(mongoDB))
	classroomService := classroom.NewService(classroomRepo, studentRepo)
	classroomHandler := handlers.NewClassroomHandler(
		classroom.NewHandler(classroomService),
		classroom.NewStudentHandler(classroomService),
	)

	// Dependências de perfil do professor
	usersRepo, _ := users.NewRepository(client.Database(mongoDB))
	profileRepo := users.NewProfileRepository(client.Database(mongoDB))
	profileService := users.NewProfileService(profileRepo)
	profileHandler := handlers.NewProfileHandler(profileService, usersRepo)

	// Inicializa dependências de relatórios (observações e LLM)
	reportRepo := classroom.NewReportRepository(client.Database(mongoDB), studentRepo)
	reportHandler := handlers.NewReportHandler(
		classroom.NewReportHandler(
			studentRepo,
			reportRepo.StudentObsRepo,
			reportRepo.StudentLLMRepo,
			reportRepo.ClassObsRepo,
			reportRepo.ClassLLMRepo,
		),
	)

	chatThreadRepo := classroom.NewChatThreadRepo(client.Database(mongoDB))
	chatMessageRepo := classroom.NewChatMessageRepo(client.Database(mongoDB))
	chatResponsibleRepo := classroom.NewChatResponsibleRepo(client.Database(mongoDB))
	chatHandler := classroom.NewChatHandler(chatThreadRepo, chatMessageRepo, studentRepo, classroomRepo, chatResponsibleRepo)

	app := &application{
		authHandler:      handlers.NewAuthHandler(authService),
		classroomHandler: classroomHandler,
		profileHandler:   profileHandler,
		reportHandler:    reportHandler,
		chatHandler:      chatHandler,
	}

	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      app.routes(),
		ReadTimeout:  reportTimeout,
		WriteTimeout: reportTimeout,
		IdleTimeout:  30 * time.Second,
	}

	log.Printf("starting web server on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func mustConnectMongo(mongoURI string) *mongo.Client {
	// Usa timeout para evitar bloqueios indefinidos na conexão inicial.
	ctx, cancel := context.WithTimeout(context.Background(), mongoTimeout)
	defer cancel()

	client, err := database.ConnectMongo(ctx, mongoURI)
	if err != nil {
		log.Fatal(err)
	}

	return client
}

func disconnectMongo(client *mongo.Client) {
	// Fecha conexão com timeout controlado durante o shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), mongoTimeout)
	defer cancel()
	_ = client.Disconnect(ctx)
}
