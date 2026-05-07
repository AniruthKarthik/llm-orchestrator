package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server         ServerConfig
	Workers        WorkersConfig
	LLM            LLMConfig
	Postgres       PostgresConfig
	Queue          QueueConfig
	CircuitBreaker CircuitBreakerConfig
	RateLimit      RateLimitConfig
}

type ServerConfig struct {
	Addr          string
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	IdleTimeout   time.Duration
	ShutdownGrace time.Duration
}

type WorkersConfig struct {
	Count int
}

type LLMConfig struct {
	Provider       string
	OpenAIKey      string
	OpenAIModel    string
	AnthropicKey   string
	AnthropicModel string
	Timeout        time.Duration
	MaxRetries     int
}

type PostgresConfig struct {
	Enabled bool
	DSN     string
}

type QueueConfig struct {
	BufferSize int
}

type CircuitBreakerConfig struct {
	FailureThreshold int
	SuccessThreshold int
	Timeout          time.Duration
}

type RateLimitConfig struct {
	RequestsPerMinute int
}

func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Addr:          getEnv("ADDR", ":8080"),
			ReadTimeout:   5 * time.Second,
			WriteTimeout:  10 * time.Second,
			IdleTimeout:   120 * time.Second,
			ShutdownGrace: 10 * time.Second,
		},
		Workers: WorkersConfig{
			Count: getEnvInt("WORKER_COUNT", 4),
		},
		LLM: LLMConfig{
			Provider:       getEnv("LLM_PROVIDER", "mock"),
			OpenAIKey:      os.Getenv("OPENAI_API_KEY"),
			OpenAIModel:    getEnv("OPENAI_MODEL", "gpt-4"),
			AnthropicKey:   os.Getenv("ANTHROPIC_API_KEY"),
			AnthropicModel: getEnv("ANTHROPIC_MODEL", "claude-3-opus-20240229"),
			Timeout:        60 * time.Second,
			MaxRetries:     3,
		},
		Postgres: PostgresConfig{
			Enabled: os.Getenv("POSTGRES_DSN") != "",
			DSN:     os.Getenv("POSTGRES_DSN"),
		},
		Queue: QueueConfig{
			BufferSize: getEnvInt("QUEUE_BUFFER_SIZE", 100),
		},
		CircuitBreaker: CircuitBreakerConfig{
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          30 * time.Second,
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: getEnvInt("RATE_LIMIT", 60),
		},
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}
