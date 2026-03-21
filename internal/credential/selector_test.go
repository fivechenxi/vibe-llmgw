package credential

// Unit and module tests for RoundRobinSelector.
//
// All tests use fakeCredRepo — no real database required.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/yourorg/llmgw/internal/domain"
)

// ---- fake in-memory credential repo ----

type fakeCredRepo struct {
	creds map[string][]domain.ModelCredential
	err   error
}

func newFakeCredRepo(creds ...domain.ModelCredential) *fakeCredRepo {
	f := &fakeCredRepo{creds: make(map[string][]domain.ModelCredential)}
	for _, c := range creds {
		f.creds[c.ModelID] = append(f.creds[c.ModelID], c)
	}
	return f
}

func (f *fakeCredRepo) ListActive(_ context.Context, modelID string) ([]domain.ModelCredential, error) {
	if f.err != nil {
		return nil, f.err
	}
	cs := f.creds[modelID]
	if len(cs) == 0 {
		return nil, fmt.Errorf("no active credentials for model %q", modelID)
	}
	return cs, nil
}

// helpers
func makeCreds(modelID string, n int) []domain.ModelCredential {
	cs := make([]domain.ModelCredential, n)
	for i := range cs {
		cs[i] = domain.ModelCredential{ID: i + 1, ModelID: modelID, APIKey: fmt.Sprintf("key-%d", i+1), IsActive: true}
	}
	return cs
}

func newSelector(repo credentialsLister) *RoundRobinSelector {
	return &RoundRobinSelector{repo: repo}
}

var _ credentialsLister = (*fakeCredRepo)(nil)

// ---- tests ----

func TestSelector_StickySession_SameSessionSameCred(t *testing.T) {
	repo := newFakeCredRepo(makeCreds("gpt-4o", 3)...)
	sel := newSelector(repo)
	ctx := context.Background()

	first, err := sel.Pick(ctx, "gpt-4o", "session-abc")
	if err != nil {
		t.Fatalf("Pick error: %v", err)
	}
	for i := 0; i < 10; i++ {
		got, _ := sel.Pick(ctx, "gpt-4o", "session-abc")
		if got.ID != first.ID {
			t.Errorf("pick %d: got credential %d, want %d (sticky)", i, got.ID, first.ID)
		}
	}
}

func TestSelector_StickySession_DifferentSessionsMayDiffer(t *testing.T) {
	repo := newFakeCredRepo(makeCreds("gpt-4o", 5)...)
	sel := newSelector(repo)
	ctx := context.Background()

	ids := make(map[int]struct{})
	for i := 0; i < 20; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		c, _ := sel.Pick(ctx, "gpt-4o", sessionID)
		ids[c.ID] = struct{}{}
	}
	// With 5 creds and 20 distinct sessions, at least 2 distinct credentials should be chosen.
	if len(ids) < 2 {
		t.Errorf("expected at least 2 distinct credentials across sessions, got %d", len(ids))
	}
}

func TestSelector_RoundRobin_EmptySession_Cycles(t *testing.T) {
	creds := makeCreds("gpt-4o", 3)
	repo := newFakeCredRepo(creds...)
	sel := newSelector(repo)
	ctx := context.Background()

	seen := make([]int, 3)
	for i := 0; i < 9; i++ {
		c, err := sel.Pick(ctx, "gpt-4o", "")
		if err != nil {
			t.Fatalf("Pick error: %v", err)
		}
		seen[c.ID-1]++
	}
	// Each credential should be picked exactly 3 times in 9 round-robin calls.
	for idx, count := range seen {
		if count != 3 {
			t.Errorf("cred %d picked %d times, want 3", idx+1, count)
		}
	}
}

func TestSelector_RoundRobin_SingleCred(t *testing.T) {
	repo := newFakeCredRepo(makeCreds("gpt-4o", 1)...)
	sel := newSelector(repo)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		c, err := sel.Pick(ctx, "gpt-4o", "")
		if err != nil {
			t.Fatalf("Pick error: %v", err)
		}
		if c.ID != 1 {
			t.Errorf("expected ID=1, got %d", c.ID)
		}
	}
}

func TestSelector_RepoError_Propagates(t *testing.T) {
	dbErr := errors.New("db timeout")
	repo := &fakeCredRepo{creds: nil, err: dbErr}
	sel := newSelector(repo)

	_, err := sel.Pick(context.Background(), "gpt-4o", "")
	if !errors.Is(err, dbErr) {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestSelector_NoCredentials_ReturnsError(t *testing.T) {
	repo := newFakeCredRepo() // empty
	sel := newSelector(repo)

	_, err := sel.Pick(context.Background(), "gpt-4o", "sess-1")
	if err == nil {
		t.Error("expected error for model with no credentials, got nil")
	}
}

func TestSelector_MultiModel_IndependentCounters(t *testing.T) {
	repo := newFakeCredRepo(
		append(makeCreds("gpt-4o", 2), makeCreds("claude-haiku-4-5", 3)...)...,
	)
	sel := newSelector(repo)
	ctx := context.Background()

	// 4 calls for gpt-4o (2-cred pool → RR: 1,2,1,2)
	for i := 0; i < 4; i++ {
		c, err := sel.Pick(ctx, "gpt-4o", "")
		if err != nil {
			t.Fatalf("gpt-4o Pick error: %v", err)
		}
		expected := (i % 2) + 1
		if c.ID != expected {
			t.Errorf("gpt-4o pick %d: got ID=%d, want %d", i, c.ID, expected)
		}
	}

	// 3 calls for claude (3-cred pool → RR: 1,2,3)
	for i := 0; i < 3; i++ {
		c, err := sel.Pick(ctx, "claude-haiku-4-5", "")
		if err != nil {
			t.Fatalf("claude Pick error: %v", err)
		}
		expected := i + 1
		if c.ID != expected {
			t.Errorf("claude pick %d: got ID=%d, want %d", i, c.ID, expected)
		}
	}
}

func TestSelector_ConcurrentPick_AllValid(t *testing.T) {
	repo := newFakeCredRepo(makeCreds("gpt-4o", 4)...)
	sel := newSelector(repo)
	ctx := context.Background()

	const goroutines = 50
	var wg sync.WaitGroup
	var errCount atomic.Int32
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			c, err := sel.Pick(ctx, "gpt-4o", "")
			if err != nil {
				errCount.Add(1)
				return
			}
			if c.ID < 1 || c.ID > 4 {
				errCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if errCount.Load() != 0 {
		t.Errorf("%d goroutines got invalid credential or error", errCount.Load())
	}
}

func TestSelector_SessionSticky_ConcurrentSameSession(t *testing.T) {
	repo := newFakeCredRepo(makeCreds("gpt-4o", 5)...)
	sel := newSelector(repo)
	ctx := context.Background()

	const goroutines = 30
	var wg sync.WaitGroup
	ids := make([]int, goroutines)
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		idx := i
		go func() {
			defer wg.Done()
			c, _ := sel.Pick(ctx, "gpt-4o", "fixed-session")
			ids[idx] = c.ID
		}()
	}
	wg.Wait()

	first := ids[0]
	for i, id := range ids {
		if id != first {
			t.Errorf("goroutine %d got credential %d, want %d (sticky)", i, id, first)
		}
	}
}
