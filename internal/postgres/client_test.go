package postgres

import (
	"context"
	"gitlab.scruzzi.com/root/postgresqlui/internal/models"
	"strings"
	"testing"
)

func TestConnectionErrorsAreActionable(t *testing.T) {
	_, err := Connect(context.Background(), models.Server{Host: "127.0.0.1", Port: 1, User: "none", Password: "secret", SSLMode: "disable"})
	if err == nil || !strings.Contains(err.Error(), "verify reachability") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatal("password leaked")
	}
}
