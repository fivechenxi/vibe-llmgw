package credential

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"

	"github.com/yourorg/llmgw/internal/domain"
)

// Selector picks a credential for a given model and session.
type Selector interface {
	// Pick selects a credential for modelID.
	// sessionID is used for session-sticky selection: the same non-empty sessionID
	// always maps to the same credential (hash-based), so all turns in a conversation
	// are routed to the same backend account.
	// When sessionID is empty a global round-robin counter is used instead.
	Pick(ctx context.Context, modelID, sessionID string) (*domain.ModelCredential, error)
}

// credentialsLister is the narrow DB interface RoundRobinSelector needs.
// *Repository satisfies it; tests can substitute a fake.
type credentialsLister interface {
	ListActive(ctx context.Context, modelID string) ([]domain.ModelCredential, error)
}

// RoundRobinSelector implements Selector.
// - Non-empty sessionID: hash(sessionID) % len(creds) — deterministic, no state needed.
// - Empty sessionID: per-model atomic counter (classic RR).
type RoundRobinSelector struct {
	repo     credentialsLister
	counters sync.Map // model_id -> *atomic.Uint64
}

func NewRoundRobinSelector(repo *Repository) *RoundRobinSelector {
	return &RoundRobinSelector{repo: repo}
}

func (s *RoundRobinSelector) Pick(ctx context.Context, modelID, sessionID string) (*domain.ModelCredential, error) {
	creds, err := s.repo.ListActive(ctx, modelID)
	if err != nil {
		return nil, err
	}
	if len(creds) == 0 {
		return nil, fmt.Errorf("no active credentials for model %q", modelID)
	}

	var idx uint64
	if sessionID != "" {
		// Session-sticky: same session always maps to the same credential.
		h := fnv.New64a()
		_, _ = h.Write([]byte(sessionID))
		idx = h.Sum64() % uint64(len(creds))
	} else {
		// No session: fall back to round-robin for load distribution.
		counter := s.counterFor(modelID)
		idx = (counter.Add(1) - 1) % uint64(len(creds))
	}

	picked := creds[idx]
	return &picked, nil
}

func (s *RoundRobinSelector) counterFor(modelID string) *atomic.Uint64 {
	v, _ := s.counters.LoadOrStore(modelID, &atomic.Uint64{})
	return v.(*atomic.Uint64)
}
