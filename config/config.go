package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	App      `yaml:"app"`
	DB       `yaml:"database"`
	VectorDB `yaml:"vector_database"`
	HTTP     `yaml:"http"`
	Log      `yaml:"logger"`
}

type App struct {
	Name    string `yaml:"name" env:"APP_NAME"`
	Version string `yaml:"version" env:"APP_VERSION"`
	Env     string `yaml:"env" env:"APP_ENV"`
}

type DB struct {
	URL string `yaml:"url" env:"DATABASE_URL"`
}

type VectorDB struct {
	Host   string `yaml:"host" env:"VECTOR_DATABASE_HOST"`
	Scheme string `yaml:"scheme" env:"VECTOR_DATABASE_SCHEME"`
}

type HTTP struct {
	Port string `yaml:"port" env:"HTTP_PORT"`
}

type Log struct {
	Level string `yaml:"level" env:"LOG_LEVEL"`
}

type Auth struct {
	Username string `yaml:"username" env:"AUTH_USERNAME"`
	Password string `yaml:"password" env:"AUTH_PASSWORD"`
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
		return nil, fmt.Errorf("error loading env: %w", err)
	}
	return cfg, nil
}
