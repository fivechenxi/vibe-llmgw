package model

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

func (r *Repository) ListActive(ctx context.Context) ([]domain.Model, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, provider, is_active FROM models WHERE is_active=true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []domain.Model
	for rows.Next() {
		var m domain.Model
		if err := rows.Scan(&m.ID, &m.Name, &m.Provider, &m.IsActive); err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, nil
}