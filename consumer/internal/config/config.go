package config

import (
	"fmt"

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

type App struct {
	Name    string `yaml:"name" env:"APP_NAME"`
	Version string `yaml:"version" env:"APP_VERSION"`
	Env     string `yaml:"env" env:"APP_ENV"`
}

type Kafka struct {
	Brokers       string `yaml:"brokers" env:"KAFKA_BROKERS"`
	Topic         string `yaml:"topic" env:"KAFKA_TOPIC"`
	ConsumerGroup string `yaml:"consumer_group" env:"KAFKA_CONSUMER_GROUP"`
	BatchSize     int    `yaml:"batch_size" env:"KAFKA_BATCH_SIZE"`
	BatchTimeout  string `yaml:"batch_timeout" env:"KAFKA_BATCH_TIMEOUT"`
}

type ClickHouse struct {
	Host     string `yaml:"host" env:"CLICKHOUSE_HOST"`
	Port     int    `yaml:"port" env:"CLICKHOUSE_PORT"`
	Database string `yaml:"database" env:"CLICKHOUSE_DATABASE"`
	Username string `yaml:"username" env:"CLICKHOUSE_USERNAME"`
	Password string `yaml:"password" env:"CLICKHOUSE_PASSWORD"`
	Table    string `yaml:"table" env:"CLICKHOUSE_TABLE"`
}

type Server struct {
	Port         int    `yaml:"port" env:"SERVER_PORT"`
	ReadTimeout  string `yaml:"read_timeout" env:"SERVER_READ_TIMEOUT"`
	WriteTimeout string `yaml:"write_timeout" env:"SERVER_WRITE_TIMEOUT"`
}

type Log struct {
	Level string `yaml:"level" env:"LOG_LEVEL"`
	JSON  bool   `yaml:"json" env:"LOG_JSON"`
}

type Metrics struct {
	Enabled bool   `yaml:"enabled" env:"METRICS_ENABLED"`
	Port    string `yaml:"port" env:"METRICS_PORT"`
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
