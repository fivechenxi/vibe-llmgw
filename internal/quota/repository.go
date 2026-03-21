package quota

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/llmgw/internal/domain"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Get reads the current quota row for display or pre-flight checks.
//
// Replica-lag warning: this SELECT must execute on the primary node.
// If the pgxpool DSN points to a load-balancer that routes reads to a
// replica, used_tokens may be stale (lagging behind recent Deduct calls).
// For display purposes this is acceptable; for quota enforcement use
// TryDeduct instead, which atomically checks and updates on the primary.
func (r *Repository) Get(ctx context.Context, userID, modelID string) (*domain.UserQuota, error) {
	q := &domain.UserQuota{}
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, model_id, quota_tokens, used_tokens, reset_period, reset_date
		 FROM user_quotas WHERE user_id=$1 AND model_id=$2`,
		userID, modelID,
	).Scan(&q.ID, &q.UserID, &q.ModelID, &q.QuotaTokens, &q.UsedTokens, &q.ResetPeriod, &q.ResetDate)
	return q, err
}

// Deduct blindly increments used_tokens after a completed request.
// It does NOT enforce the quota ceiling — callers must call Check first.
//
// TOCTOU note: the Check → LLM-call → Deduct sequence is not atomic.
// Concurrent requests for the same user+model can all pass Check before
// any Deduct runs, allowing momentary over-quota usage.  If strict
// enforcement is required, use TryDeduct instead.
func (r *Repository) Deduct(ctx context.Context, userID, modelID string, tokens int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE user_quotas SET used_tokens = used_tokens + $1
		 WHERE user_id=$2 AND model_id=$3`,
		tokens, userID, modelID,
	)
	return err
}

// TryDeduct atomically checks the remaining quota and, if sufficient,
// increments used_tokens in a single UPDATE statement.
//
// Because the check and the write happen in the same SQL statement on the
// primary, this eliminates both the TOCTOU race (no gap between read and
// write) and the replica-lag problem (UPDATE always targets the primary).
//
// Returns ErrQuotaExceeded when:
//   - quota_tokens - used_tokens <= 0 (quota exhausted), OR
//   - no row exists for the user+model pair.
//
// Callers that need to distinguish "no row" from "quota exhausted" should
// call Get first; TryDeduct conflates both into ErrQuotaExceeded intentionally
// to keep the hot path to a single round-trip.
func (r *Repository) TryDeduct(ctx context.Context, userID, modelID string, tokens int) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE user_quotas SET used_tokens = used_tokens + $1
		 WHERE user_id=$2 AND model_id=$3 AND quota_tokens - used_tokens > 0`,
		tokens, userID, modelID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrQuotaExceeded
	}
	return nil
}

func (r *Repository) ListByUser(ctx context.Context, userID string) ([]domain.UserQuota, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, model_id, quota_tokens, used_tokens, reset_period, reset_date
		 FROM user_quotas WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quotas []domain.UserQuota
	for rows.Next() {
		var q domain.UserQuota
		if err := rows.Scan(&q.ID, &q.UserID, &q.ModelID, &q.QuotaTokens, &q.UsedTokens, &q.ResetPeriod, &q.ResetDate); err != nil {
			return nil, err
		}
		quotas = append(quotas, q)
	}
	return quotas, nil
}