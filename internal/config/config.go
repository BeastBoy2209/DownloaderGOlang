package config

import (
	"log"
	"github.com/caarlos0/env/v11"
)

type Config struct {
	ServerPort string `env:"SERVER_PORT,required"`

//бдшка
	DBHost     string `env:"DB_HOST,required"`
	DBPort     string `env:"DB_PORT,required"`
	DBUser     string `env:"DB_USER,required"`
	DBPassword string `env:"DB_PASSWORD,required"`
	DBName     string `env:"DB_NAME,required"`
}

// при отсутсвии переменных крашнется
func Load() *Config {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("ошбибка, не хватает переменных: %v", err)
	}
	return &cfg
}