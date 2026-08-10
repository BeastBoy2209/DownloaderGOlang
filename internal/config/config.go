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

// при отсутсвии переменных крашнется
func Load() *Config {
	
	if err := godotenv.Load(); err != nil {
		log.Printf(".env file not exist: %v", err)
	}
	var cfg Config
	if err := env.Parse(&cfg); err != nil{
		log.Fatalf("somethig wrong with credentials in .env: %v", err)
	}
	return &cfg
}