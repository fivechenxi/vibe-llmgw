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

func (r *Repository) Get(ctx context.Context, userID, modelID string) (*domain.UserQuota, error) {
	q := &domain.UserQuota{}
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, model_id, quota_tokens, used_tokens, reset_period, reset_date
		 FROM user_quotas WHERE user_id=$1 AND model_id=$2`,
		userID, modelID,
	).Scan(&q.ID, &q.UserID, &q.ModelID, &q.QuotaTokens, &q.UsedTokens, &q.ResetPeriod, &q.ResetDate)
	return q, err
}

func (r *Repository) Deduct(ctx context.Context, userID, modelID string, tokens int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE user_quotas SET used_tokens = used_tokens + $1
		 WHERE user_id=$2 AND model_id=$3`,
		tokens, userID, modelID,
	)
	return err
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