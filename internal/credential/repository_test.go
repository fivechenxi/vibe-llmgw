package credential

// Integration tests for Repository.ListActive.
// Requires a real PostgreSQL instance with the schema applied.
//
// Set TEST_DATABASE_URL to run:
//   TEST_DATABASE_URL="postgres://..." go test ./internal/credential/ -v -run TestRepository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping credential repository integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// seedModel ensures a models row exists (FK dependency).
func seedModel(t *testing.T, db *pgxpool.Pool, modelID string) {
	t.Helper()
	_, err := db.Exec(context.Background(),
		`INSERT INTO models (id, name, provider) VALUES ($1, $2, 'test')
		 ON CONFLICT (id) DO NOTHING`,
		modelID, modelID,
	)
	if err != nil {
		t.Fatalf("seedModel: %v", err)
	}
}

// seedCredential inserts a model_credentials row and registers cleanup.
func seedCredential(t *testing.T, db *pgxpool.Pool, modelID, apiKey string, isActive bool) int {
	t.Helper()
	ctx := context.Background()
	var id int
	err := db.QueryRow(ctx,
		`INSERT INTO model_credentials (model_id, api_key, label, is_active, created_at)
		 VALUES ($1, $2, 'test-label', $3, $4) RETURNING id`,
		modelID, apiKey, isActive, time.Now(),
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedCredential: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(context.Background(),
			`DELETE FROM model_credentials WHERE id=$1`, id)
	})
	return id
}

func TestRepository_ListActive_ReturnsActiveOnly(t *testing.T) {
	db := newTestDB(t)
	modelID := fmt.Sprintf("test-model-%d", time.Now().UnixNano())
	seedModel(t, db, modelID)

	activeID := seedCredential(t, db, modelID, "sk-active", true)
	_ = seedCredential(t, db, modelID, "sk-inactive", false)

	repo := NewRepository(db)
	creds, err := repo.ListActive(context.Background(), modelID)
	if err != nil {
		t.Fatalf("ListActive error: %v", err)
	}
	if len(creds) != 1 {
		t.Fatalf("expected 1 active credential, got %d", len(creds))
	}
	if creds[0].ID != activeID {
		t.Errorf("got credential ID %d, want %d", creds[0].ID, activeID)
	}
	if creds[0].APIKey != "sk-active" {
		t.Errorf("APIKey = %q, want sk-active", creds[0].APIKey)
	}
	if !creds[0].IsActive {
		t.Error("returned credential should have IsActive=true")
	}
}

func TestRepository_ListActive_OrderedByID(t *testing.T) {
	db := newTestDB(t)
	modelID := fmt.Sprintf("test-model-order-%d", time.Now().UnixNano())
	seedModel(t, db, modelID)

	id1 := seedCredential(t, db, modelID, "sk-1", true)
	id2 := seedCredential(t, db, modelID, "sk-2", true)
	id3 := seedCredential(t, db, modelID, "sk-3", true)

	repo := NewRepository(db)
	creds, err := repo.ListActive(context.Background(), modelID)
	if err != nil {
		t.Fatalf("ListActive error: %v", err)
	}
	if len(creds) != 3 {
		t.Fatalf("expected 3 credentials, got %d", len(creds))
	}
	if creds[0].ID != id1 || creds[1].ID != id2 || creds[2].ID != id3 {
		t.Errorf("credentials not ordered by id: got %v %v %v, want %v %v %v",
			creds[0].ID, creds[1].ID, creds[2].ID, id1, id2, id3)
	}
}

func TestRepository_ListActive_NoActiveCredentials_ReturnsError(t *testing.T) {
	db := newTestDB(t)
	modelID := fmt.Sprintf("test-model-inactive-%d", time.Now().UnixNano())
	seedModel(t, db, modelID)
	_ = seedCredential(t, db, modelID, "sk-inactive", false)

	repo := NewRepository(db)
	_, err := repo.ListActive(context.Background(), modelID)
	if err == nil {
		t.Error("expected error when no active credentials, got nil")
	}
}

func TestRepository_ListActive_UnknownModel_ReturnsError(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)

	_, err := repo.ListActive(context.Background(), "no-such-model-xyz")
	if err == nil {
		t.Error("expected error for unknown model, got nil")
	}
}
