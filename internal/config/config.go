package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Qdrant    QdrantConfig    `yaml:"qdrant"`
	OpenAI    OpenAIConfig    `yaml:"openai"`
	Auth      AuthConfig      `yaml:"auth"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
}

type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

type QdrantConfig struct {
	URL          string        `yaml:"url"`
	APIKey       string        `yaml:"api_key"`
	SyncInterval time.Duration `yaml:"sync_interval"`
}

type OpenAIConfig struct {
	APIKey         string `yaml:"api_key"`
	EmbeddingModel string `yaml:"embedding_model"`
}

type AuthConfig struct {
	SessionTTL time.Duration `yaml:"session_ttl"`
	BcryptCost int           `yaml:"bcrypt_cost"`
}

type RateLimitConfig struct {
	RequestsPerMinute int `yaml:"requests_per_minute"`
	LoginPerMinute    int `yaml:"login_per_minute"`
}

// Load reads and validates the YAML config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("config: parse: %w", err)
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) validate() error {
	if c.Database.DSN == "" {
		return fmt.Errorf("config: database.dsn is required")
	}
	if c.Qdrant.URL == "" {
		return fmt.Errorf("config: qdrant.url is required")
	}
	if c.OpenAI.APIKey == "" {
		return fmt.Errorf("config: openai.api_key is required")
	}
	if c.Server.Addr == "" {
		c.Server.Addr = ":8080"
	}
	if c.Qdrant.SyncInterval == 0 {
		c.Qdrant.SyncInterval = 60 * time.Second
	}
	if c.Auth.SessionTTL == 0 {
		c.Auth.SessionTTL = 24 * time.Hour
	}
	if c.Auth.BcryptCost == 0 {
		c.Auth.BcryptCost = 12
	}
	return nil
}