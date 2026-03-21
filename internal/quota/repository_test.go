package quota

// Integration tests for Repository.
// Requires a real PostgreSQL instance with the schema applied.
//
// Set TEST_DATABASE_URL to run:
//   TEST_DATABASE_URL="postgres://user:pass@localhost:5432/llmgw_test?sslmode=disable" \
//     go test ./internal/quota/ -v -run TestRepository
//
// The tests insert rows under a unique test user prefix and clean up after themselves.

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
		t.Skip("TEST_DATABASE_URL not set, skipping repository integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

// seedQuota inserts a test quota row and registers cleanup.
// Uses a unique userID per test to avoid cross-test interference.
func seedQuota(t *testing.T, db *pgxpool.Pool, userID, modelID string, quota, used int64) {
	t.Helper()
	ctx := context.Background()

	// Ensure test user exists (users table has FK constraint)
	_, err := db.Exec(ctx,
		`INSERT INTO users (id, email, name) VALUES ($1, $2, $3)
		 ON CONFLICT (id) DO NOTHING`,
		userID, userID+"@test.local", "Test User",
	)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	// Ensure test model exists
	_, err = db.Exec(ctx,
		`INSERT INTO models (id, name, provider) VALUES ($1, $2, $3)
		 ON CONFLICT (id) DO NOTHING`,
		modelID, modelID, "test",
	)
	if err != nil {
		t.Fatalf("seed model: %v", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO user_quotas (user_id, model_id, quota_tokens, used_tokens, reset_period, reset_date)
		 VALUES ($1, $2, $3, $4, 'monthly', $5)
		 ON CONFLICT (user_id, model_id) DO UPDATE
		   SET quota_tokens = EXCLUDED.quota_tokens,
		       used_tokens   = EXCLUDED.used_tokens`,
		userID, modelID, quota, used, time.Now().AddDate(0, 1, 0),
	)
	if err != nil {
		t.Fatalf("seed quota: %v", err)
	}

	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = db.Exec(ctx, `DELETE FROM user_quotas WHERE user_id=$1`, userID)
		_, _ = db.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID)
	})
}

func TestRepository_Get_Exists(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	userID := fmt.Sprintf("test-get-%d", time.Now().UnixNano())
	modelID := "mock"

	seedQuota(t, db, userID, modelID, 5000, 1200)

	q, err := repo.Get(context.Background(), userID, modelID)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if q.QuotaTokens != 5000 {
		t.Errorf("expected quota_tokens=5000, got %d", q.QuotaTokens)
	}
	if q.UsedTokens != 1200 {
		t.Errorf("expected used_tokens=1200, got %d", q.UsedTokens)
	}
	if q.Remaining() != 3800 {
		t.Errorf("expected Remaining()=3800, got %d", q.Remaining())
	}
	if q.UserID != userID {
		t.Errorf("expected user_id=%q, got %q", userID, q.UserID)
	}
}

func TestRepository_Get_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)

	_, err := repo.Get(context.Background(), "no-such-user", "no-such-model")
	if err == nil {
		t.Error("expected error for missing row, got nil")
	}
}

func TestRepository_Deduct_UpdatesUsedTokens(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	userID := fmt.Sprintf("test-deduct-%d", time.Now().UnixNano())
	modelID := "mock"

	seedQuota(t, db, userID, modelID, 10000, 0)

	if err := repo.Deduct(context.Background(), userID, modelID, 300); err != nil {
		t.Fatalf("Deduct error: %v", err)
	}

	q, err := repo.Get(context.Background(), userID, modelID)
	if err != nil {
		t.Fatalf("Get after Deduct error: %v", err)
	}
	if q.UsedTokens != 300 {
		t.Errorf("expected used_tokens=300 after deduct, got %d", q.UsedTokens)
	}
}

func TestRepository_Deduct_Accumulates(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	userID := fmt.Sprintf("test-accumulate-%d", time.Now().UnixNano())
	modelID := "mock"

	seedQuota(t, db, userID, modelID, 10000, 0)

	_ = repo.Deduct(context.Background(), userID, modelID, 100)
	_ = repo.Deduct(context.Background(), userID, modelID, 200)
	_ = repo.Deduct(context.Background(), userID, modelID, 50)

	q, _ := repo.Get(context.Background(), userID, modelID)
	if q.UsedTokens != 350 {
		t.Errorf("expected used_tokens=350, got %d", q.UsedTokens)
	}
}

func TestRepository_ListByUser_ReturnsAll(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	userID := fmt.Sprintf("test-list-%d", time.Now().UnixNano())

	seedQuota(t, db, userID, "mock", 1000, 0)
	// seed a second model — reuse the unique user, seed a second model row directly
	_, _ = db.Exec(context.Background(),
		`INSERT INTO models (id, name, provider) VALUES ('mock2', 'mock2', 'test') ON CONFLICT DO NOTHING`)
	_, _ = db.Exec(context.Background(),
		`INSERT INTO user_quotas (user_id, model_id, quota_tokens, used_tokens, reset_period, reset_date)
		 VALUES ($1, 'mock2', 2000, 0, 'monthly', $2)
		 ON CONFLICT (user_id, model_id) DO NOTHING`,
		userID, time.Now().AddDate(0, 1, 0),
	)

	quotas, err := repo.ListByUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUser error: %v", err)
	}
	if len(quotas) < 2 {
		t.Errorf("expected at least 2 quota rows, got %d", len(quotas))
	}
}

func TestRepository_ListByUser_NoRows(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)

	quotas, err := repo.ListByUser(context.Background(), "ghost-user-xyz")
	if err != nil {
		t.Fatalf("ListByUser should return empty slice for unknown user, got error: %v", err)
	}
	if len(quotas) != 0 {
		t.Errorf("expected empty slice, got %d rows", len(quotas))
	}
}
