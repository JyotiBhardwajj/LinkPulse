package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	t.Run("Valid Config succeeds", func(t *testing.T) {
		c := &Config{
			Server: ServerConfig{
				Port:           "8080",
				RequestTimeout: 5 * time.Second,
				BaseURL:        "https://linkpulse.com",
			},
			Database: DatabaseConfig{
				Host:   "localhost",
				Port:   "5432",
				DBName: "linkpulse_db",
				User:   "postgres",
			},
			Redis: RedisConfig{
				Host: "localhost",
				Port: "6379",
			},
			Cache: CacheConfig{
				TTL: 24 * time.Hour,
			},
			Worker: WorkerConfig{
				Count:     5,
				QueueSize: 1000,
			},
			JWT: JWTConfig{
				Secret:             "supersecretjwtkeythatisreallylong",
				AccessTokenTTL:     15 * time.Minute,
				RefreshTokenTTL:    7 * 24 * time.Hour,
				MaxSessionsPerUser: 10,
			},
		}
		assert.NoError(t, c.Validate())
	})

	t.Run("Empty Port fails", func(t *testing.T) {
		c := Config{}
		err := c.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SERVER_PORT cannot be empty")
	})

	t.Run("Invalid BaseURL fails", func(t *testing.T) {
		c := Config{
			Server: ServerConfig{
				Port:           "8080",
				RequestTimeout: 5 * time.Second,
				BaseURL:        "not-a-valid-url",
			},
		}
		err := c.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is not a valid absolute URL")
	})

	t.Run("Refresh token TTL smaller than access token TTL fails", func(t *testing.T) {
		c := Config{
			Server: ServerConfig{
				Port:           "8080",
				RequestTimeout: 5 * time.Second,
				BaseURL:        "https://linkpulse.com",
			},
			Database: DatabaseConfig{
				Host:   "localhost",
				Port:   "5432",
				DBName: "linkpulse_db",
				User:   "postgres",
			},
			Redis: RedisConfig{
				Host: "localhost",
				Port: "6379",
			},
			Cache: CacheConfig{
				TTL: 24 * time.Hour,
			},
			Worker: WorkerConfig{
				Count:     5,
				QueueSize: 1000,
			},
			JWT: JWTConfig{
				Secret:             "secret",
				AccessTokenTTL:     15 * time.Minute,
				RefreshTokenTTL:    5 * time.Minute, // smaller
				MaxSessionsPerUser: 10,
			},
		}
		err := c.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be greater than ACCESS_TOKEN_TTL")
	})

	t.Run("Max session count negative fails", func(t *testing.T) {
		c := Config{
			Server: ServerConfig{
				Port:           "8080",
				RequestTimeout: 5 * time.Second,
				BaseURL:        "https://linkpulse.com",
			},
			Database: DatabaseConfig{
				Host:   "localhost",
				Port:   "5432",
				DBName: "linkpulse_db",
				User:   "postgres",
			},
			Redis: RedisConfig{
				Host: "localhost",
				Port: "6379",
			},
			Cache: CacheConfig{
				TTL: 24 * time.Hour,
			},
			Worker: WorkerConfig{
				Count:     5,
				QueueSize: 1000,
			},
			JWT: JWTConfig{
				Secret:             "secret",
				AccessTokenTTL:     15 * time.Minute,
				RefreshTokenTTL:    24 * time.Hour,
				MaxSessionsPerUser: -5, // invalid
			},
		}
		err := c.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MAX_SESSIONS_PER_USER must be greater than 0")
	})
}
