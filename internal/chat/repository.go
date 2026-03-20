package chat

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/llmgw/internal/domain"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Save(ctx context.Context, log *domain.ChatLog) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO chat_logs
		  (id, user_id, session_id, model_id, request_at, response_at,
		   request_messages, response_content, input_tokens, output_tokens, status, error_message)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		log.ID, log.UserID, log.SessionID, log.ModelID,
		log.RequestAt, log.ResponseAt,
		log.RequestMessages, log.ResponseContent,
		log.InputTokens, log.OutputTokens,
		log.Status, log.ErrorMessage,
	)
	return err
}

func (r *Repository) ListSessions(ctx context.Context, userID string) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT session_id FROM chat_logs WHERE user_id=$1 ORDER BY session_id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		sessions = append(sessions, id)
	}
	return sessions, nil
}

func (r *Repository) GetSession(ctx context.Context, userID string, sessionID uuid.UUID) ([]domain.ChatLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, session_id, model_id, request_at, response_at,
		        request_messages, response_content, input_tokens, output_tokens, status, error_message
		 FROM chat_logs WHERE user_id=$1 AND session_id=$2 ORDER BY request_at`,
		userID, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []domain.ChatLog
	for rows.Next() {
		var l domain.ChatLog
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.SessionID, &l.ModelID,
			&l.RequestAt, &l.ResponseAt,
			&l.RequestMessages, &l.ResponseContent,
			&l.InputTokens, &l.OutputTokens,
			&l.Status, &l.ErrorMessage,
		); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}