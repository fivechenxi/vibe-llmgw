package db

// Tests for db.Connect.
//
// Unit test: Connect returns an error when the host is unreachable.
// Integration test: Connect succeeds with a real DB (skip without TEST_DATABASE_URL).

import (
	"os"
	"testing"
)

// TestConnect_UnreachableHost verifies that Connect returns a non-nil error
// when the PostgreSQL server cannot be reached.
// We use connect_timeout=1 in the DSN so the test completes quickly.
func TestConnect_UnreachableHost(t *testing.T) {
	dsn := "postgres://postgres:postgres@127.0.0.1:19999/nonexistent?connect_timeout=1"
	_, err := Connect(dsn)
	if err == nil {
		t.Error("expected error connecting to unreachable host, got nil")
	}
}

// TestConnect_InvalidScheme verifies that a completely unparseable DSN returns
// an error without hanging.
func TestConnect_InvalidScheme(t *testing.T) {
	_, err := Connect("not-a-dsn://whatever")
	if err == nil {
		t.Error("expected error for invalid DSN scheme, got nil")
	}
}

// TestConnect_Success is an integration test; skipped unless TEST_DATABASE_URL is set.
func TestConnect_Success(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping db integration test")
	}
	pool, err := Connect(dsn)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer pool.Close()

	// Pool should be non-nil and functional.
	stat := pool.Stat()
	if stat.TotalConns() < 0 {
		t.Error("unexpected negative TotalConns")
	}
}
