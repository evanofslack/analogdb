package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	App      `yaml:"app"`
	DB       `yaml:"database"`
	Redis    `yaml:"redis"`
	VectorDB `yaml:"vector_database"`
	HTTP     `yaml:"http"`
	Log      `yaml:"logger"`
	Auth     `yaml:"auth"`
	Metrics  `yaml:"metrics"`
}

type App struct {
	Name             string `yaml:"name" env:"APP_NAME"`
	Version          string `yaml:"version" env:"APP_VERSION"`
	Env              string `yaml:"env" env:"APP_ENV"`
	CacheEnabled     bool   `yaml:"cache_enabled" env:"CACHE_ENABLED"`
	RateLimitEnabled bool   `yaml:"rate_limit_enabled" env:"RATE_LIMIT_ENABLED"`
}

type DB struct {
	URL string `yaml:"url" env:"DATABASE_URL"`
}

type Redis struct {
	URL string `yaml:"url" env:"REDIS_URL"`
}

type VectorDB struct {
	Host   string `yaml:"host" env:"VECTOR_DATABASE_HOST"`
	Scheme string `yaml:"scheme" env:"VECTOR_DATABASE_SCHEME"`
}

type HTTP struct {
	Port string `yaml:"port" env:"HTTP_PORT"`
}

type Log struct {
	Level      string `yaml:"level" env:"LOG_LEVEL"`
	WebhookURL string `yaml:"webhook" env:"WEBHOOK_URL"`
}

type Auth struct {
	Username          string `yaml:"username" env:"AUTH_USERNAME"`
	Password          string `yaml:"password" env:"AUTH_PASSWORD"`
	RateLimitUsername string `yaml:"rate_limit_username" env:"RATE_LIMIT_AUTH_USERNAME"`
	RateLimitPassword string `yaml:"rate_limit_password" env:"RATE_LIMIT_AUTH_PASSWORD"`
}

type Metrics struct {
	Enabled bool   `yaml:"enabled" env:"METRICS_ENABLED"`
	Port    string `yaml:"port" env:"METRICS_PORT"`
}

func New(path string) (*Config, error) {
	cfg := &Config{}

	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}
	if err := godotenv.Load(); err != nil {
		fmt.Println("Could not load .env file")
	}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, fmt.Errorf("Error loading env: %w", err)
	}
	return cfg, nil
}
