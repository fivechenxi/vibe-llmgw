package credential

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/llmgw/internal/config"
)

// modelAPIKeyFunc maps each model ID to the function that returns its API key from config.
// Must stay in sync with proxy/router.go's model registration.
var modelAPIKeyFunc = map[string]func(*config.Config) string{
	"gpt-4o":                    func(c *config.Config) string { return c.Providers.OpenAI.APIKey },
	"gpt-4o-mini":               func(c *config.Config) string { return c.Providers.OpenAI.APIKey },
	"gpt-5":                     func(c *config.Config) string { return c.Providers.OpenAI.APIKey },
	"claude-3-5-sonnet":         func(c *config.Config) string { return c.Providers.Anthropic.APIKey },
	"claude-3-haiku":            func(c *config.Config) string { return c.Providers.Anthropic.APIKey },
	"claude-haiku-4-5":          func(c *config.Config) string { return c.Providers.Anthropic.APIKey },
	"claude-3-5-haiku-20241022": func(c *config.Config) string { return c.Providers.Anthropic.APIKey },
	"deepseek-v3":               func(c *config.Config) string { return c.Providers.DeepSeek.APIKey },
	"deepseek-r1":               func(c *config.Config) string { return c.Providers.DeepSeek.APIKey },
	"qwen-max":                  func(c *config.Config) string { return c.Providers.Alibaba.APIKey },
	"qwen-plus":                 func(c *config.Config) string { return c.Providers.Alibaba.APIKey },
	"qwen3-max-2026-01-23":      func(c *config.Config) string { return c.Providers.Alibaba.APIKey },
	"qwen3.5-plus":              func(c *config.Config) string { return c.Providers.Alibaba.APIKey },
}

// SyncFromConfig upserts API keys from config into the model_credentials table at startup.
// Rows with label="config" are replaced on each startup to keep keys current.
// Empty api_key means "remove config credential for this model".
func SyncFromConfig(ctx context.Context, cfg *config.Config, db *pgxpool.Pool) {
	synced := 0
	for modelID, keyFn := range modelAPIKeyFunc {
		apiKey := keyFn(cfg)
		if err := syncOneModelConfigCredential(ctx, db, modelID, apiKey); err != nil {
			log.Printf("credential sync: sync config cred for %s: %v", modelID, err)
			continue
		}
		if apiKey != "" {
			synced++
		}
	}
	if synced > 0 {
		log.Printf("credential sync: synced %d model(s) from config", synced)
	}
}

func syncOneModelConfigCredential(ctx context.Context, db *pgxpool.Pool, modelID, apiKey string) error {
	return withTx(ctx, db, func(tx pgx.Tx) error {
		return syncOneModelConfigCredentialWithTx(ctx, tx, modelID, apiKey)
	})
}

func syncOneModelConfigCredentialWithTx(ctx context.Context, tx pgx.Tx, modelID, apiKey string) error {
	if _, err := tx.Exec(ctx,
		`DELETE FROM model_credentials WHERE model_id = $1 AND label = 'config'`,
		modelID,
	); err != nil {
		return err
	}
	if apiKey == "" {
		return nil
	}
	_, err := tx.Exec(ctx,
		`INSERT INTO model_credentials (model_id, api_key, label, is_active) VALUES ($1, $2, 'config', true)`,
		modelID, apiKey,
	)
	return err
}

func withTx(ctx context.Context, db *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
