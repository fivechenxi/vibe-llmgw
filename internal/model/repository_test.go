package model

// Integration tests for Repository.ListActive.
// Requires a real PostgreSQL instance with the schema applied.
//
// Set TEST_DATABASE_URL to run:
//   TEST_DATABASE_URL="postgres://..." go test ./internal/model/ -v -run TestModelRepository

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
		t.Skip("TEST_DATABASE_URL not set, skipping model repository integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// seedModel inserts a model row and registers cleanup.
func seedModel(t *testing.T, db *pgxpool.Pool, id, name, provider string, isActive bool) {
	t.Helper()
	_, err := db.Exec(context.Background(),
		`INSERT INTO models (id, name, provider, is_active) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (id) DO UPDATE SET is_active = EXCLUDED.is_active`,
		id, name, provider, isActive,
	)
	if err != nil {
		t.Fatalf("seedModel: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(context.Background(), `DELETE FROM models WHERE id=$1`, id)
	})
}

func TestModelRepository_ListActive_ReturnsActiveModels(t *testing.T) {
	db := newTestDB(t)
	ts := fmt.Sprintf("%d", time.Now().UnixNano())
	activeID := "test-model-active-" + ts
	inactiveID := "test-model-inactive-" + ts

	seedModel(t, db, activeID, activeID, "test", true)
	seedModel(t, db, inactiveID, inactiveID, "test", false)

	repo := NewRepository(db)
	models, err := repo.ListActive(context.Background())
	if err != nil {
		t.Fatalf("ListActive error: %v", err)
	}

	foundActive := false
	foundInactive := false
	for _, m := range models {
		if m.ID == activeID {
			foundActive = true
		}
		if m.ID == inactiveID {
			foundInactive = true
		}
	}
	if !foundActive {
		t.Errorf("active model %q not found in ListActive result", activeID)
	}
	if foundInactive {
		t.Errorf("inactive model %q should NOT be in ListActive result", inactiveID)
	}
}

func TestModelRepository_ListActive_AllModelsHaveIsActiveTrue(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	models, err := repo.ListActive(context.Background())
	if err != nil {
		t.Fatalf("ListActive error: %v", err)
	}
	for _, m := range models {
		if !m.IsActive {
			t.Errorf("model %q has is_active=false but was returned by ListActive", m.ID)
		}
	}
}

func TestModelRepository_ListActive_ScansAllFields(t *testing.T) {
	db := newTestDB(t)
	ts := fmt.Sprintf("%d", time.Now().UnixNano())
	id := "test-model-scan-" + ts
	seedModel(t, db, id, "Scan Test Model", "openai", true)

	repo := NewRepository(db)
	models, err := repo.ListActive(context.Background())
	if err != nil {
		t.Fatalf("ListActive error: %v", err)
	}
	for _, m := range models {
		if m.ID == id {
			if m.Name != "Scan Test Model" {
				t.Errorf("Name = %q, want 'Scan Test Model'", m.Name)
			}
			if m.Provider != "openai" {
				t.Errorf("Provider = %q, want openai", m.Provider)
			}
			if !m.IsActive {
				t.Error("IsActive should be true")
			}
			return
		}
	}
	t.Errorf("model %q not found in results", id)
}
