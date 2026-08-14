package config

import (
	"log"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Server Server `envPrefix:"SERVER_"`
	DB     DB     `envPrefix:"DB_"`
}

type Server struct {
	Port uint16 `env:"PORT,notEmpty"`
}

type DB struct {
	Host     string `env:"HOST,notEmpty"`
	Port     uint16 `env:"PORT,notEmpty"`
	User     string `env:"USER,notEmpty"`
	Password string `env:"PASSWORD,notEmpty"`
	Name     string `env:"NAME,notEmpty"`
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Printf(".env file not found: %v", err)
	}
	var cfg Config
	err = env.Parse(&cfg)
	if err != nil {
		log.Fatalf("failed to parse environment configuration: %v", err)
	}

	return &cfg
}
