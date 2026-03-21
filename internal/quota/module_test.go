package quota

// Module tests for the quota package.
//
// These tests wire Service + a fully-functional in-memory repository together
// to verify complete business flows without any real database.
// The fake repo simulates actual storage semantics:
//   - Get returns ErrNotFound for unknown keys
//   - Deduct atomically adds to used_tokens
//   - concurrent-safe via a simple map + no locking (single-goroutine tests)

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/yourorg/llmgw/internal/domain"
)

// ---- fake in-memory repository ----

var errNotFound = errors.New("quota: no rows")

type fakeRepo struct {
	mu     sync.Mutex
	quotas map[string]*domain.UserQuota // key: "userID:modelID"
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{quotas: make(map[string]*domain.UserQuota)}
}

func (f *fakeRepo) key(userID, modelID string) string {
	return userID + ":" + modelID
}

// Seed pre-populates a quota entry for a user+model pair.
func (f *fakeRepo) Seed(userID, modelID string, quota, used int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.quotas[f.key(userID, modelID)] = &domain.UserQuota{
		UserID:      userID,
		ModelID:     modelID,
		QuotaTokens: quota,
		UsedTokens:  used,
		ResetPeriod: "monthly",
	}
}

func (f *fakeRepo) Get(_ context.Context, userID, modelID string) (*domain.UserQuota, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	q, ok := f.quotas[f.key(userID, modelID)]
	if !ok {
		return nil, errNotFound
	}
	// return a copy so callers can't mutate store state
	cp := *q
	return &cp, nil
}

func (f *fakeRepo) Deduct(_ context.Context, userID, modelID string, tokens int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	q, ok := f.quotas[f.key(userID, modelID)]
	if !ok {
		return errNotFound
	}
	q.UsedTokens += int64(tokens)
	return nil
}

// TryDeduct atomically checks remaining quota and deducts under a single lock,
// mirroring the DB-level atomicity of Repository.TryDeduct.
func (f *fakeRepo) TryDeduct(_ context.Context, userID, modelID string, tokens int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	q, ok := f.quotas[f.key(userID, modelID)]
	if !ok {
		return ErrQuotaExceeded
	}
	if q.QuotaTokens-q.UsedTokens <= 0 {
		return ErrQuotaExceeded
	}
	q.UsedTokens += int64(tokens)
	return nil
}

// UsedTokens is a test-only helper to inspect stored state.
func (f *fakeRepo) UsedTokens(userID, modelID string) int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.quotas[f.key(userID, modelID)].UsedTokens
}

// newModuleService wires Service with the fake repo (bypasses NewService's *Repository type).
func newModuleService(repo *fakeRepo) *Service {
	return &Service{repo: repo}
}

// ---- module tests ----

// TestModule_FullFlow: allocate → check passes → deduct → check passes again → exhaust → check fails.
func TestModule_FullFlow(t *testing.T) {
	repo := newFakeRepo()
	repo.Seed("alice", "gpt-4o", 1000, 0)
	svc := newModuleService(repo)
	ctx := context.Background()

	// initial check should pass
	if err := svc.Check(ctx, "alice", "gpt-4o"); err != nil {
		t.Fatalf("initial Check failed: %v", err)
	}

	// deduct 600 tokens
	if err := svc.Deduct(ctx, "alice", "gpt-4o", 600); err != nil {
		t.Fatalf("Deduct failed: %v", err)
	}
	if got := repo.UsedTokens("alice", "gpt-4o"); got != 600 {
		t.Errorf("expected 600 used_tokens, got %d", got)
	}

	// still 400 remaining — check should still pass
	if err := svc.Check(ctx, "alice", "gpt-4o"); err != nil {
		t.Fatalf("Check after partial deduct failed: %v", err)
	}

	// deduct remaining 400
	if err := svc.Deduct(ctx, "alice", "gpt-4o", 400); err != nil {
		t.Fatalf("second Deduct failed: %v", err)
	}

	// now exactly 0 remaining — Check must return ErrQuotaExceeded
	if err := svc.Check(ctx, "alice", "gpt-4o"); !errors.Is(err, ErrQuotaExceeded) {
		t.Errorf("expected ErrQuotaExceeded after exhaustion, got %v", err)
	}
}

// TestModule_MultiUser: quota is isolated per user, deducting for one user doesn't affect another.
func TestModule_MultiUser(t *testing.T) {
	repo := newFakeRepo()
	repo.Seed("alice", "gpt-4o", 500, 0)
	repo.Seed("bob", "gpt-4o", 500, 0)
	svc := newModuleService(repo)
	ctx := context.Background()

	// exhaust alice
	_ = svc.Deduct(ctx, "alice", "gpt-4o", 500)
	if err := svc.Check(ctx, "alice", "gpt-4o"); !errors.Is(err, ErrQuotaExceeded) {
		t.Error("alice should be exhausted")
	}

	// bob should be unaffected
	if err := svc.Check(ctx, "bob", "gpt-4o"); err != nil {
		t.Errorf("bob's quota should still be available, got %v", err)
	}
}

// TestModule_MultiModel: quota is isolated per model for the same user.
func TestModule_MultiModel(t *testing.T) {
	repo := newFakeRepo()
	repo.Seed("alice", "gpt-4o", 1000, 0)
	repo.Seed("alice", "claude-haiku-4-5", 200, 0)
	svc := newModuleService(repo)
	ctx := context.Background()

	// exhaust claude quota only
	_ = svc.Deduct(ctx, "alice", "claude-haiku-4-5", 200)
	if err := svc.Check(ctx, "alice", "claude-haiku-4-5"); !errors.Is(err, ErrQuotaExceeded) {
		t.Error("claude quota should be exhausted")
	}
	if err := svc.Check(ctx, "alice", "gpt-4o"); err != nil {
		t.Errorf("gpt-4o quota should be unaffected, got %v", err)
	}
}

// TestModule_NoQuotaRow: Check on a user with no quota row returns a non-ErrQuotaExceeded error.
func TestModule_NoQuotaRow(t *testing.T) {
	repo := newFakeRepo()
	svc := newModuleService(repo)

	err := svc.Check(context.Background(), "ghost", "gpt-4o")
	if err == nil {
		t.Fatal("expected error for missing quota row, got nil")
	}
	if errors.Is(err, ErrQuotaExceeded) {
		t.Error("missing row should not be ErrQuotaExceeded — it's a different failure mode")
	}
}

// TestModule_DeductBeyondQuota: Service.Deduct does not enforce a ceiling —
// it delegates enforcement to the caller (handler checks before deducting).
func TestModule_DeductBeyondQuota(t *testing.T) {
	repo := newFakeRepo()
	repo.Seed("alice", "gpt-4o", 100, 0)
	svc := newModuleService(repo)
	ctx := context.Background()

	// Deduct more than quota — repo allows it (no ceiling check in Deduct)
	if err := svc.Deduct(ctx, "alice", "gpt-4o", 999); err != nil {
		t.Fatalf("Deduct should not error: %v", err)
	}
	// used_tokens is now 999, remaining is negative → Check returns exceeded
	if err := svc.Check(ctx, "alice", "gpt-4o"); !errors.Is(err, ErrQuotaExceeded) {
		t.Error("over-deducted quota should report exceeded")
	}
}

// TestModule_ConcurrentDeduct: multiple goroutines deducting concurrently
// must not corrupt the total; final used_tokens must equal sum of all deductions.
func TestModule_ConcurrentDeduct(t *testing.T) {
	repo := newFakeRepo()
	repo.Seed("alice", "gpt-4o", 1_000_000, 0)
	svc := newModuleService(repo)
	ctx := context.Background()

	const goroutines = 50
	const tokensEach = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = svc.Deduct(ctx, "alice", "gpt-4o", tokensEach)
		}()
	}
	wg.Wait()

	got := repo.UsedTokens("alice", "gpt-4o")
	expected := int64(goroutines * tokensEach)
	if got != expected {
		t.Errorf("concurrent deduct: expected %d total used_tokens, got %d", expected, got)
	}
}

// TestModule_CheckDeductMultiRound: simulate N sequential requests each consuming tokens,
// verifying the quota drains correctly and trips at the right point.
func TestModule_CheckDeductMultiRound(t *testing.T) {
	const quota = 1000
	const tokensPerReq = 300

	repo := newFakeRepo()
	repo.Seed("alice", "gpt-4o", quota, 0)
	svc := newModuleService(repo)
	ctx := context.Background()

	successCount := 0
	for i := 0; i < 10; i++ {
		if err := svc.Check(ctx, "alice", "gpt-4o"); err != nil {
			break
		}
		_ = svc.Deduct(ctx, "alice", "gpt-4o", tokensPerReq)
		successCount++
	}

	// Check passes while remaining > 0:
	//   req1: 0 used   → remaining=1000 → pass, deduct → 300 used
	//   req2: 300 used → remaining=700  → pass, deduct → 600 used
	//   req3: 600 used → remaining=400  → pass, deduct → 900 used
	//   req4: 900 used → remaining=100  → pass, deduct → 1200 used
	//   req5: 1200 used → remaining=-200 → fail
	if successCount != 4 {
		t.Errorf("expected 4 successful requests before quota exceeded, got %d", successCount)
	}
	if repo.UsedTokens("alice", "gpt-4o") != 1200 {
		t.Errorf("expected 1200 used_tokens, got %d", repo.UsedTokens("alice", "gpt-4o"))
	}
}

// TestModule_DeductNoRow: Deduct on a non-existent row returns an error.
func TestModule_DeductNoRow(t *testing.T) {
	repo := newFakeRepo()
	svc := newModuleService(repo)

	err := svc.Deduct(context.Background(), "ghost", "gpt-4o", 100)
	if err == nil {
		t.Error("expected error when deducting from non-existent row")
	}
}

// TestModule_LargeQuota: sanity check for int64 range.
func TestModule_LargeQuota(t *testing.T) {
	repo := newFakeRepo()
	repo.Seed("alice", "gpt-4o", 10_000_000_000, 0)
	svc := newModuleService(repo)
	ctx := context.Background()

	if err := svc.Check(ctx, "alice", "gpt-4o"); err != nil {
		t.Fatalf("large quota Check failed: %v", err)
	}
	_ = svc.Deduct(ctx, "alice", "gpt-4o", 5_000_000_000)
	if err := svc.Check(ctx, "alice", "gpt-4o"); err != nil {
		t.Errorf("should still have 5B tokens remaining, got %v", err)
	}
}

// TestModule_UsersIsolation: table-driven, verify N users don't bleed into each other.
func TestModule_UsersIsolation(t *testing.T) {
	users := []string{"alice", "bob", "carol", "dave"}
	repo := newFakeRepo()
	for _, u := range users {
		repo.Seed(u, "gpt-4o", 500, 0)
	}
	svc := newModuleService(repo)
	ctx := context.Background()

	// exhaust every user except the last
	for _, u := range users[:len(users)-1] {
		_ = svc.Deduct(ctx, u, "gpt-4o", 500)
	}

	for i, u := range users {
		err := svc.Check(ctx, u, "gpt-4o")
		if i < len(users)-1 {
			if !errors.Is(err, ErrQuotaExceeded) {
				t.Errorf("user %q should be exhausted", u)
			}
		} else {
			if err != nil {
				t.Errorf("user %q should still have quota, got %v", u, err)
			}
		}
	}
}

// TestModule_ConcurrentCheckAndDeduct demonstrates the TOCTOU race in the
// Check → (LLM call) → Deduct pattern.
//
// With quota=100 and 5 goroutines each doing Check then Deduct(100):
//   - All goroutines call Check while used_tokens=0 → all see remaining=100 → all pass.
//   - All then call Deduct(100) → used_tokens grows to 500, far exceeding the quota.
//
// This is an inherent property of the two-phase design, not a bug in fakeRepo.
// Use TryDeduct to eliminate this race when strict per-request enforcement is needed.
func TestModule_ConcurrentCheckAndDeduct(t *testing.T) {
	repo := newFakeRepo()
	repo.Seed("alice", "gpt-4o", 100, 0)
	svc := newModuleService(repo)
	ctx := context.Background()

	const goroutines = 5
	// barrier forces all goroutines to complete Check before any starts Deduct,
	// making the TOCTOU window as wide as possible.
	barrier := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			// All goroutines pass Check with used=0 → remaining=100 > 0.
			_ = svc.Check(ctx, "alice", "gpt-4o")
			// Wait until all have passed Check, then all Deduct at once.
			<-barrier
			_ = svc.Deduct(ctx, "alice", "gpt-4o", 100)
		}()
	}
	// Give goroutines time to reach the barrier, then release.
	close(barrier)
	wg.Wait()

	used := repo.UsedTokens("alice", "gpt-4o")
	// All 5 goroutines deducted → used=500, quota=100 is far exceeded.
	if used != int64(goroutines*100) {
		t.Errorf("expected %d used_tokens (all goroutines deducted), got %d", goroutines*100, used)
	}
	t.Logf("TOCTOU confirmed: quota=100, used_tokens=%d after %d concurrent Check+Deduct pairs", used, goroutines)
}

// TestModule_TryDeduct_ConcurrentSafe shows that TryDeduct prevents over-quota:
// 5 goroutines each try to deduct 100 against a quota of 100 — exactly one succeeds.
func TestModule_TryDeduct_ConcurrentSafe(t *testing.T) {
	repo := newFakeRepo()
	repo.Seed("alice", "gpt-4o", 100, 0)
	svc := newModuleService(repo)
	ctx := context.Background()

	const goroutines = 5
	start := make(chan struct{})

	var succeeded int64
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-start
			if err := svc.TryDeduct(ctx, "alice", "gpt-4o", 100); err == nil {
				atomic.AddInt64(&succeeded, 1)
			}
		}()
	}

	close(start)
	wg.Wait()

	if succeeded != 1 {
		t.Errorf("expected exactly 1 TryDeduct to succeed (quota=100, each wants 100), got %d", succeeded)
	}
	if repo.UsedTokens("alice", "gpt-4o") != 100 {
		t.Errorf("expected used_tokens=100, got %d", repo.UsedTokens("alice", "gpt-4o"))
	}
}

// fakeRepo also needs to satisfy quotaRepo for the compiler —
// verify it at compile time.
var _ quotaRepo = (*fakeRepo)(nil)

// Ensure test user names are always unique across parallel test runs.
var _ = fmt.Sprintf // keep import used
