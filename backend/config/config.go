package config

import (
	"fmt"
	"time"

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
	Tracing  `yaml:"tracing"`
	Kafka    `yaml:"kafka"`
}

type App struct {
	Name             string `yaml:"name" env:"APP_NAME"`
	Version          string `yaml:"version" env:"APP_VERSION"`
	Env              string `yaml:"env" env:"APP_ENV"`
	CacheEnabled     bool   `yaml:"cache_enabled" env:"CACHE_ENABLED"`
	RateLimitEnabled bool   `yaml:"rate_limit_enabled" env:"RATE_LIMIT_ENABLED"`
}

type DB struct {
	URL              string        `yaml:"url" env:"DATABASE_URL"`
	MigrationEnabled bool          `yaml:"migration_enabled" env:"MIGRATION_ENABLED"`
	MigrationPath    string        `yaml:"migration_path" env:"MIGRATION_PATH"`
	MaxOpenConns     int           `yaml:"max_open_conns" env:"DATABASE_MAX_OPEN_CONNS" env-default:"20"`
	MaxIdleConns     int           `yaml:"max_idle_conns" env:"DATABASE_MAX_IDLE_CONNS" env-default:"10"`
	ConnMaxLifetime  time.Duration `yaml:"conn_max_lifetime" env:"DATABASE_CONN_MAX_LIFETIME" env-default:"30m"`
	ConnMaxIdleTime  time.Duration `yaml:"conn_max_idle_time" env:"DATABASE_CONN_MAX_IDLE_TIME" env-default:"5m"`
}

type Redis struct {
	URL string `yaml:"url" env:"REDIS_URL"`
}

type VectorDB struct {
	Host   string `yaml:"host" env:"VECTOR_DATABASE_HOST"`
	Scheme string `yaml:"scheme" env:"VECTOR_DATABASE_SCHEME"`
}

type HTTP struct {
	Port               string        `yaml:"port" env:"HTTP_PORT"`
	TrustedProxies     []string      `yaml:"trusted_proxies" env:"HTTP_TRUSTED_PROXIES" env-separator:","`
	ShutdownDrainDelay time.Duration `yaml:"shutdown_drain_delay" env:"SHUTDOWN_DRAIN_DELAY"`
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

type Tracing struct {
	Enabled  bool   `yaml:"enabled" env:"TRACING_ENABLED"`
	Endpoint string `yaml:"endpoint" env:"TRACING_ENDPOINT"`
}

type Kafka struct {
	Enabled bool   `yaml:"enabled" env:"KAFKA_ENABLED"`
	Topic   string `yaml:"topic" env:"KAFKA_TOPIC"`
	Brokers string `yaml:"brokers" env:"KAFKA_BROKERS"`

	QueueSize    int           `yaml:"queue_size" env:"KAFKA_QUEUE_SIZE" env-default:"10000"`
	BatchSize    int           `yaml:"batch_size" env:"KAFKA_BATCH_SIZE" env-default:"100"`
	BatchTimeout time.Duration `yaml:"batch_timeout" env:"KAFKA_BATCH_TIMEOUT" env-default:"1s"`
}

func New(path string) (*Config, error) {
	cfg := &Config{}

	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if err := godotenv.Load(); err != nil {
		fmt.Println("Could not load .env file")
	}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}
	return cfg, nil
}
