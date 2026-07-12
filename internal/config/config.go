// Package config defines the configuration structure and loading mechanism for the application.
package config

import (
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/viper"
)

// ServerConfig stores settings related to the HTTP server.
type ServerConfig struct {
	Port                 string        `mapstructure:"SERVER_PORT"`
	Env                  string        `mapstructure:"SERVER_ENV"`
	RequestTimeout       time.Duration `mapstructure:"SERVER_REQUEST_TIMEOUT"`
	ShortCodeLength      int           `mapstructure:"SHORT_CODE_LENGTH"`
	MaxGenerationRetries int           `mapstructure:"MAX_GENERATION_RETRIES"`
	BaseURL              string        `mapstructure:"BASE_URL"`
	Version              string        `mapstructure:"BUILD_VERSION"`
	GitCommit            string        `mapstructure:"GIT_COMMIT"`
	BuildTime            string        `mapstructure:"BUILD_TIME"`
}

// DatabaseConfig stores PostgreSQL credentials and connection settings.
type DatabaseConfig struct {
	Host     string `mapstructure:"POSTGRES_HOST"`
	Port     string `mapstructure:"POSTGRES_PORT"`
	DBName   string `mapstructure:"POSTGRES_DB"`
	User     string `mapstructure:"POSTGRES_USER"`
	Password string `mapstructure:"POSTGRES_PASSWORD"`
}

// RedisConfig stores Redis connection coordinates.
type RedisConfig struct {
	Host string `mapstructure:"REDIS_HOST"`
	Port string `mapstructure:"REDIS_PORT"`
}

// CacheConfig stores cache-specific settings.
type CacheConfig struct {
	TTL    time.Duration `mapstructure:"CACHE_TTL"`
	Prefix string        `mapstructure:"CACHE_PREFIX"`
}

// WorkerConfig stores settings for the background worker pool.
type WorkerConfig struct {
	Count     int `mapstructure:"WORKER_COUNT"`
	QueueSize int `mapstructure:"WORKER_QUEUE_SIZE"`
}

// CleanupConfig stores settings for the background cleanup scheduler.
type CleanupConfig struct {
	Interval time.Duration `mapstructure:"CLEANUP_INTERVAL"`
}

// JWTConfig stores authentication secret keys and expiration values.
type JWTConfig struct {
	Secret           string        `mapstructure:"JWT_SECRET"`
	AccessTokenTTL   time.Duration `mapstructure:"ACCESS_TOKEN_TTL"`
	RefreshTokenTTL  time.Duration `mapstructure:"REFRESH_TOKEN_TTL"`
	Issuer           string        `mapstructure:"TOKEN_ISSUER"`
	MaxLoginAttempts int           `mapstructure:"MAX_LOGIN_ATTEMPTS"`
}

// Config is the top-level configuration container for LinkPulse.
type Config struct {
	Server   ServerConfig   `mapstructure:",squash"`
	Database DatabaseConfig `mapstructure:",squash"`
	Redis    RedisConfig    `mapstructure:",squash"`
	Cache    CacheConfig    `mapstructure:",squash"`
	JWT      JWTConfig      `mapstructure:",squash"`
	Worker   WorkerConfig   `mapstructure:",squash"`
	Cleanup  CleanupConfig  `mapstructure:",squash"`
	LogLevel string         `mapstructure:"LOG_LEVEL"`
}

// Validate checks that all configuration parameters satisfy range constraints.
func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("SERVER_PORT cannot be empty")
	}
	if c.Server.RequestTimeout <= 0 {
		return fmt.Errorf("SERVER_REQUEST_TIMEOUT must be greater than 0")
	}
	if c.Server.BaseURL == "" {
		return fmt.Errorf("BASE_URL cannot be empty")
	}
	u, err := url.ParseRequestURI(c.Server.BaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("BASE_URL '%s' is not a valid absolute URL: %w", c.Server.BaseURL, err)
	}

	if c.Database.Host == "" || c.Database.Port == "" || c.Database.DBName == "" || c.Database.User == "" {
		return fmt.Errorf("database configuration is incomplete")
	}
	if c.Redis.Host == "" || c.Redis.Port == "" {
		return fmt.Errorf("redis configuration is incomplete")
	}

	if c.Cache.TTL <= 0 {
		return fmt.Errorf("CACHE_TTL must be greater than 0")
	}

	if c.Worker.Count <= 0 {
		return fmt.Errorf("WORKER_COUNT must be greater than 0")
	}
	if c.Worker.QueueSize <= 0 {
		return fmt.Errorf("WORKER_QUEUE_SIZE must be greater than 0")
	}

	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET cannot be empty")
	}
	if c.JWT.AccessTokenTTL <= 0 {
		return fmt.Errorf("ACCESS_TOKEN_TTL must be greater than 0")
	}
	if c.JWT.RefreshTokenTTL <= c.JWT.AccessTokenTTL {
		return fmt.Errorf("REFRESH_TOKEN_TTL must be greater than ACCESS_TOKEN_TTL")
	}

	return nil
}

// LoadConfig reads the environment variables and files to populate the Config struct.
func LoadConfig() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("SERVER_ENV", "development")
	viper.SetDefault("SERVER_REQUEST_TIMEOUT", "5s")
	viper.SetDefault("SHORT_CODE_LENGTH", 7)
	viper.SetDefault("MAX_GENERATION_RETRIES", 5)
	viper.SetDefault("BASE_URL", "http://localhost:8080")
	viper.SetDefault("POSTGRES_HOST", "localhost")
	viper.SetDefault("POSTGRES_PORT", "5432")
	viper.SetDefault("POSTGRES_DB", "linkpulse_db")
	viper.SetDefault("POSTGRES_USER", "postgres")
	viper.SetDefault("POSTGRES_PASSWORD", "postgres")
	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("CACHE_TTL", "24h")
	viper.SetDefault("CACHE_PREFIX", "link:")
	viper.SetDefault("WORKER_COUNT", 5)
	viper.SetDefault("WORKER_QUEUE_SIZE", 1000)
	viper.SetDefault("CLEANUP_INTERVAL", "1h")
	viper.SetDefault("JWT_SECRET", "supersecretjwtkeythatisreallylongandsecure")
	viper.SetDefault("ACCESS_TOKEN_TTL", "15m")
	viper.SetDefault("REFRESH_TOKEN_TTL", "7d")
	viper.SetDefault("TOKEN_ISSUER", "linkpulse-api")
	viper.SetDefault("MAX_LOGIN_ATTEMPTS", 5)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("BUILD_VERSION", "1.0.0")
	viper.SetDefault("GIT_COMMIT", "unknown")
	viper.SetDefault("BUILD_TIME", "unknown")

	if err := viper.ReadInConfig(); err != nil {
		// It's okay if .env is missing in production since environment variables may be injected directly.
		fmt.Printf("Warning: Could not read .env file: %v. Relying on system environment variables.\n", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}
