package database_test

import (
	"context"
	"testing"
	"time"

	"plataforma/internal/database"
)

// TestConnectMongoInvalidURI garante falha controlada para URI inválida.
func TestConnectMongoInvalidURI(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client, err := database.ConnectMongo(ctx, "://invalid-uri")
	if err == nil {
		if client != nil {
			_ = client.Disconnect(ctx)
		}
		t.Fatal("expected error for invalid mongo uri")
	}
}
