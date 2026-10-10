package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	App        `yaml:"app"`
	DB         `yaml:"database"`
	Redis      `yaml:"redis"`
	VectorDB   `yaml:"vector_database"`
	HTTP       `yaml:"http"`
	Log        `yaml:"logger"`
	Auth       `yaml:"auth"`
	Metrics    `yaml:"metrics"`
	Tracing    `yaml:"tracing"`
	Kafka      `yaml:"kafka"`
	ClickHouse `yaml:"clickhouse"`
}

type App struct {
	Name             string `yaml:"name" env:"APP_NAME"`
	Version          string `yaml:"version" env:"APP_VERSION"`
	Env              string `yaml:"env" env:"APP_ENV"`
	CacheEnabled     bool   `yaml:"cache_enabled" env:"CACHE_ENABLED"`
	RateLimitEnabled bool   `yaml:"rate_limit_enabled" env:"RATE_LIMIT_ENABLED"`
	// RateLimitWebPerMinute caps all web requests together, 0 turns the cap off
	RateLimitWebPerMinute int `yaml:"rate_limit_web_per_minute" env:"RATE_LIMIT_WEB_PER_MINUTE" env-default:"10000"`
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
	AdminUsername   string `yaml:"admin_username" env:"AUTH_ADMIN_USERNAME"`
	AdminPassword   string `yaml:"admin_password" env:"AUTH_ADMIN_PASSWORD"`
	ScraperUsername string `yaml:"scraper_username" env:"AUTH_SCRAPER_USERNAME"`
	ScraperPassword string `yaml:"scraper_password" env:"AUTH_SCRAPER_PASSWORD"`
	WebUsername     string `yaml:"web_username" env:"AUTH_WEB_USERNAME"`
	WebPassword     string `yaml:"web_password" env:"AUTH_WEB_PASSWORD"`

	// legacy, removed after the rollout. Username is admin, RateLimitUsername is web
	Username          string `yaml:"username" env:"AUTH_USERNAME"`
	Password          string `yaml:"password" env:"AUTH_PASSWORD"`
	RateLimitUsername string `yaml:"rate_limit_username" env:"RATE_LIMIT_AUTH_USERNAME"`
	RateLimitPassword string `yaml:"rate_limit_password" env:"RATE_LIMIT_AUTH_PASSWORD"`
}

type Credential struct {
	Name     string
	Role     string
	Legacy   bool
	Username string
	Password string
}

func (a Auth) Credentials() []Credential {
	all := []Credential{
		{Name: "admin", Role: "admin", Username: a.AdminUsername, Password: a.AdminPassword},
		{Name: "scraper", Role: "scraper", Username: a.ScraperUsername, Password: a.ScraperPassword},
		{Name: "web", Role: "web", Username: a.WebUsername, Password: a.WebPassword},
		{Name: "legacy admin", Role: "admin", Legacy: true, Username: a.Username, Password: a.Password},
		{Name: "legacy rate limit", Role: "web", Legacy: true, Username: a.RateLimitUsername, Password: a.RateLimitPassword},
	}
	enabled := make([]Credential, 0, len(all))
	for _, c := range all {
		if c.Username != "" && c.Password != "" {
			enabled = append(enabled, c)
		}
	}
	return enabled
}

// Validate refuses identical pairs, since the first match would win and a web
// pair equal to the admin pair would quietly act as admin. It returns warnings
// for pairs that share a username with different passwords
func (a Auth) Validate() (warnings []string, err error) {
	creds := a.Credentials()
	for i := range creds {
		for j := i + 1; j < len(creds); j++ {
			if creds[i].Username != creds[j].Username {
				continue
			}
			if creds[i].Password == creds[j].Password {
				return nil, fmt.Errorf("auth: %s and %s credentials are the same", creds[i].Name, creds[j].Name)
			}
			warnings = append(warnings, fmt.Sprintf("auth: %s and %s share a username", creds[i].Name, creds[j].Name))
		}
	}
	return warnings, nil
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

type ClickHouse struct {
	Enabled  bool   `yaml:"enabled" env:"CLICKHOUSE_ENABLED"`
	Host     string `yaml:"host" env:"CLICKHOUSE_HOST"`
	Port     int    `yaml:"port" env:"CLICKHOUSE_PORT" env-default:"9000"`
	Database string `yaml:"database" env:"CLICKHOUSE_DATABASE"`
	Username string `yaml:"username" env:"CLICKHOUSE_USERNAME"`
	Password string `yaml:"password" env:"CLICKHOUSE_PASSWORD"`
	Table    string `yaml:"table" env:"CLICKHOUSE_TABLE" env-default:"httprequests"`
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
