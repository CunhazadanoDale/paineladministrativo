package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort     string
	DatabaseUrl string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	return &Config{
		AppPort:     os.Getenv("PORT"),
		DatabaseUrl: os.Getenv("DATABASE_URL"),
	}
}