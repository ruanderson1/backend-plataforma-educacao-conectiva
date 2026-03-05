package auth_test

import (
	"context"
	"os"
	"testing"
	"time"

	"plataforma/internal/auth"
	"plataforma/internal/database"
)

// TestAuthServiceIntegration valida o fluxo completo: cadastro, login, me e logout.
func TestAuthServiceIntegration(t *testing.T) {
	// Usa variável dedicada para não depender do banco principal da aplicação.
	mongoURI := os.Getenv("MONGO_URI_TEST")
	if mongoURI == "" {
		t.Skip("set MONGO_URI_TEST to run integration tests")
	}

	databaseName := "plataforma_test_auth_" + time.Now().UTC().Format("20060102150405")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := database.ConnectMongo(ctx, mongoURI)
	if err != nil {
		t.Fatalf("connect mongo: %v", err)
	}
	defer func() {
		// Limpeza do banco de teste para manter execução idempotente.
		_ = client.Database(databaseName).Drop(context.Background())
		_ = client.Disconnect(context.Background())
	}()

	service, err := auth.NewService(client.Database(databaseName))
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	registeredUser, err := service.Register(auth.RegisterInput{
		Name:     "Professor Teste",
		Email:    "prof.teste@plataforma.dev",
		Password: "123456",
		Role:     auth.RoleProfessor,
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if registeredUser.ID == "" {
		t.Fatal("expected registered user id")
	}

	token, loggedUser, err := service.Login(auth.LoginInput{
		Email:    "prof.teste@plataforma.dev",
		Password: "123456",
		Role:     auth.RoleProfessor,
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if token == "" || loggedUser.ID == "" {
		t.Fatal("expected non-empty token and user")
	}

	me, err := service.UserByToken(token)
	if err != nil {
		t.Fatalf("user by token: %v", err)
	}
	if me.Email != "prof.teste@plataforma.dev" {
		t.Fatalf("unexpected user email: %s", me.Email)
	}

	service.Logout(token)
	_, err = service.UserByToken(token)
	if err == nil {
		t.Fatal("expected unauthorized after logout")
	}
}
