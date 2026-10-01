package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort     string
	DatabaseUrl string
	CORSOrigins []string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	return &Config{
		AppPort:     os.Getenv("PORT"),
		DatabaseUrl: os.Getenv("DATABASE_URL"),
		CORSOrigins: separarOrigens(os.Getenv("CORS_ORIGINS")),
	}
}

func separarOrigens(valor string) []string {
	origens := make([]string, 0)

	for _, parte := range strings.Split(valor, ",") {
		if origem := strings.TrimSpace(parte); origem != "" {
			origens = append(origens, origem)
		}
	}

	return origens
}
