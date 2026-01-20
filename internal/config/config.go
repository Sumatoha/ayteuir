package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	App      AppConfig
	Threads  ThreadsConfig
	MongoDB  MongoDBConfig
	OpenAI   OpenAIConfig
	Security SecurityConfig
	Log      LogConfig
}

type AppConfig struct {
	Env         string
	Port        int
	Host        string
	BaseURL     string
	FrontendURL string
}

type ThreadsConfig struct {
	AppID              string
	AppSecret          string
	RedirectURI        string
	WebhookVerifyToken string
	APIVersion         string
}

type MongoDBConfig struct {
	URI            string
	Database       string
	TimeoutSeconds int
}

type OpenAIConfig struct {
	APIKey         string
	Model          string
	MaxTokens      int
	TimeoutSeconds int
}

type SecurityConfig struct {
	JWTSecret      string
	JWTExpiryHours int
	EncryptionKey  string
}

type LogConfig struct {
	Level  string
	Format string
}

func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Env:         getEnv("APP_ENV", "development"),
			Port:        getEnvInt("APP_PORT", 8080),
			Host:        getEnv("APP_HOST", "0.0.0.0"),
			BaseURL:     getEnv("APP_BASE_URL", "http://localhost:8080"),
			FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
		},
		Threads: ThreadsConfig{
			AppID:              getEnv("THREADS_APP_ID", ""),
			AppSecret:          getEnv("THREADS_APP_SECRET", ""),
			RedirectURI:        getEnv("THREADS_REDIRECT_URI", ""),
			WebhookVerifyToken: getEnv("THREADS_WEBHOOK_VERIFY_TOKEN", ""),
			APIVersion:         getEnv("THREADS_API_VERSION", "v21.0"),
		},
		MongoDB: MongoDBConfig{
			URI:            getEnv("MONGODB_URI", ""),
			Database:       getEnv("MONGODB_DATABASE", "ayteuir"),
			TimeoutSeconds: getEnvInt("MONGODB_TIMEOUT_SECONDS", 10),
		},
		OpenAI: OpenAIConfig{
			APIKey:         getEnv("OPENAI_API_KEY", ""),
			Model:          getEnv("OPENAI_MODEL", "gpt-4o"),
			MaxTokens:      getEnvInt("OPENAI_MAX_TOKENS", 500),
			TimeoutSeconds: getEnvInt("OPENAI_TIMEOUT_SECONDS", 30),
		},
		Security: SecurityConfig{
			JWTSecret:      getEnv("JWT_SECRET", ""),
			JWTExpiryHours: getEnvInt("JWT_EXPIRY_HOURS", 24),
			EncryptionKey:  getEnv("ENCRYPTION_KEY", ""),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.App.Env == "production" {
		if c.Threads.AppID == "" {
			return fmt.Errorf("THREADS_APP_ID is required in production")
		}
		if c.Threads.AppSecret == "" {
			return fmt.Errorf("THREADS_APP_SECRET is required in production")
		}
		if c.MongoDB.URI == "" {
			return fmt.Errorf("MONGODB_URI is required in production")
		}
		if c.OpenAI.APIKey == "" {
			return fmt.Errorf("OPENAI_API_KEY is required in production")
		}
		if c.Security.JWTSecret == "" || len(c.Security.JWTSecret) < 32 {
			return fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
		}
		if c.Security.EncryptionKey == "" || len(c.Security.EncryptionKey) != 32 {
			return fmt.Errorf("ENCRYPTION_KEY must be exactly 32 characters in production")
		}
	}
	return nil
}

func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

func (c *Config) MongoTimeout() time.Duration {
	return time.Duration(c.MongoDB.TimeoutSeconds) * time.Second
}

func (c *Config) OpenAITimeout() time.Duration {
	return time.Duration(c.OpenAI.TimeoutSeconds) * time.Second
}

func (c *Config) JWTExpiry() time.Duration {
	return time.Duration(c.Security.JWTExpiryHours) * time.Hour
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
