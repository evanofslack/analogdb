package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	App  `yaml:"app"`
	DB   `yaml:"postgres"`
	HTTP `yaml:"http"`
	Log  `yaml:"logger"`
}

type App struct {
	Name    string `yaml:"name" env:"APP_NAME"`
	Version string `yaml:"version" env:"APP_VERSION"`
}

type DB struct {
	URL string `yaml:"url" env:"DATABASE_URL"`
}

type HTTP struct {
	Port string `yaml:"port" env:"PORT"`
}

type Log struct {
	Level string `yaml:"level" env:"LOG_LEVEL"`
}

func New(path string) (*Config, error) {
	cfg := &Config{}

	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return nil, fmt.Errorf("Error loading config: %w", err)
	}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, fmt.Errorf("Error loading env: %w", err)
	}
	return cfg, nil
}
