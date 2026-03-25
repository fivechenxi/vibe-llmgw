package credential

import (
	"context"
	"log"

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
// Only models with a non-empty api_key in config are synced.
// Rows with label="config" are replaced on each startup to keep keys current.
func SyncFromConfig(ctx context.Context, cfg *config.Config, db *pgxpool.Pool) {
	synced := 0
	for modelID, keyFn := range modelAPIKeyFunc {
		apiKey := keyFn(cfg)
		if apiKey == "" {
			continue
		}

		// Replace the existing config-sourced credential for this model (if any).
		if _, err := db.Exec(ctx,
			`DELETE FROM model_credentials WHERE model_id = $1 AND label = 'config'`,
			modelID,
		); err != nil {
			log.Printf("credential sync: delete old config cred for %s: %v", modelID, err)
			continue
		}

		if _, err := db.Exec(ctx,
			`INSERT INTO model_credentials (model_id, api_key, label, is_active) VALUES ($1, $2, 'config', true)`,
			modelID, apiKey,
		); err != nil {
			log.Printf("credential sync: insert config cred for %s: %v", modelID, err)
			continue
		}
		synced++
	}
	if synced > 0 {
		log.Printf("credential sync: synced %d model(s) from config", synced)
	}
}
