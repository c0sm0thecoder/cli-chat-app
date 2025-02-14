package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUrl     string
	JwtSecret string
	RedisUrl  string
	Port      string
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading environment variables:", err)
	}

	cfg := Config{
		DBUrl:     os.Getenv("PG_URL"),
		JwtSecret: os.Getenv("JWT_SECRET"),
		RedisUrl:  os.Getenv("REDIS_URL"),
		Port:      "PORT",
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	return cfg
}
