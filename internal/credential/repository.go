package credential

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/llmgw/internal/domain"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ListActive returns all active credentials for the given model, ordered by id.
func (r *Repository) ListActive(ctx context.Context, modelID string) ([]domain.ModelCredential, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, model_id, api_key, label, is_active, created_at
		 FROM model_credentials
		 WHERE model_id = $1 AND is_active = true
		 ORDER BY id`,
		modelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []domain.ModelCredential
	for rows.Next() {
		var c domain.ModelCredential
		if err := rows.Scan(&c.ID, &c.ModelID, &c.APIKey, &c.Label, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	if len(creds) == 0 {
		return nil, fmt.Errorf("no active credentials for model %q", modelID)
	}
	return creds, nil
}
