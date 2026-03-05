package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"plataforma/cmd/web/handlers"
	"plataforma/internal/auth"
	"plataforma/internal/database"

	"go.mongodb.org/mongo-driver/mongo"
)

type application struct {
	authHandler *handlers.AuthHandler
}

const (
	defaultServerAddr = ":4000"
	defaultMongoDB    = "plataforma_educacao_conectiva"
	mongoTimeout      = 10 * time.Second
)

func main() {
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

	app := &application{authHandler: handlers.NewAuthHandler(authService)}

	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      app.routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
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
