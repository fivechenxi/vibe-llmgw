package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	SSO      SSOConfig
	Providers ProvidersConfig
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	DSN string
}

type JWTConfig struct {
	Secret      string
	ExpireHours int `mapstructure:"expire_hours"`
}

type SSOConfig struct {
	Provider    string
	WechatWork  WechatWorkConfig `mapstructure:"wechat_work"`
}

type WechatWorkConfig struct {
	CorpID  string `mapstructure:"corp_id"`
	AgentID string `mapstructure:"agent_id"`
	Secret  string
}

type ProvidersConfig struct {
	OpenAI    ProviderConfig
	Anthropic ProviderConfig
	DeepSeek  ProviderConfig `mapstructure:"deepseek"`
	Alibaba   ProviderConfig
	Baidu     BaiduConfig
}

type ProviderConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

type BaiduConfig struct {
	APIKey    string `mapstructure:"api_key"`
	SecretKey string `mapstructure:"secret_key"`
}

func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}
	return &cfg
}