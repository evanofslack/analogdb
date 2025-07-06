package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	App        `yaml:"app"`
	Kafka      `yaml:"kafka"`
	ClickHouse `yaml:"clickhouse"`
	Server     `yaml:"server"`
	Log        `yaml:"log"`
	Metrics    `yaml:"metrics"`
}

func New(path string) (*Config, error) {
	cfg := &Config{}

	if err := godotenv.Load(); err != nil {
		fmt.Println("Could not load .env file")
	}
	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}
	return cfg, nil
}

type App struct {
	Name    string `yaml:"name" env:"APP_NAME"`
	Version string `yaml:"version" env:"APP_VERSION"`
	Env     string `yaml:"env" env:"APP_ENV"`
}

type Kafka struct {
	BrokersRaw       string `yaml:"brokers" env:"KAFKA_BROKERS"`
	Topic         string `yaml:"topic" env:"KAFKA_TOPIC"`
	ConsumerGroup string `yaml:"consumer_group" env:"KAFKA_CONSUMER_GROUP"`
	BatchSize     int    `yaml:"batch_size" env:"KAFKA_BATCH_SIZE"`
	BatchTimeoutRaw  string `yaml:"batch_timeout" env:"KAFKA_BATCH_TIMEOUT"`
}

func (k *Kafka) Brokers() []string {
	return strings.Split(k.BrokersRaw, ",")
}

func (k *Kafka) BatchTimeout() (time.Duration, error) {
	return time.ParseDuration(k.BatchTimeoutRaw)
}

type ClickHouse struct {
	Host             string `yaml:"host" env:"CLICKHOUSE_HOST"`
	Port             int    `yaml:"port" env:"CLICKHOUSE_PORT"`
	Database         string `yaml:"database" env:"CLICKHOUSE_DATABASE"`
	Username         string `yaml:"username" env:"CLICKHOUSE_USERNAME"`
	Password         string `yaml:"password" env:"CLICKHOUSE_PASSWORD"`
	Table            string `yaml:"table" env:"CLICKHOUSE_TABLE"`
	MigrationEnabled bool   `yaml:"migration_enabled" env:"CLICKHOUSE_MIGRATION_ENABLED"`
	MigrationPath    string `yaml:"migration_path" env:"CLICKHOUSE_MIGRATION_PATH"`
}

type Server struct {
	Port int `yaml:"port" env:"SERVER_PORT"`
}

type Log struct {
	Level string `yaml:"level" env:"LOG_LEVEL"`
	Env   string `yaml:"env" env:"LOG_ENV"`
}

type Metrics struct {
	Enabled bool   `yaml:"enabled" env:"METRICS_ENABLED"`
	Port    string `yaml:"port" env:"METRICS_PORT"`
}
