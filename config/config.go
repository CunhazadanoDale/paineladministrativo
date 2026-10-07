package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const expiracaoTokenPadrao = 480 * time.Minute

type Config struct {
	AppPort      string
	DatabaseUrl  string
	CORSOrigins  []string
	JWTSecret    string
	JWTExpiracao time.Duration
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	return &Config{
		AppPort:      os.Getenv("PORT"),
		DatabaseUrl:  os.Getenv("DATABASE_URL"),
		CORSOrigins:  separarOrigens(os.Getenv("CORS_ORIGINS")),
		JWTSecret:    strings.TrimSpace(os.Getenv("JWT_SECRET")),
		JWTExpiracao: expiracaoToken(os.Getenv("JWT_EXPIRA_MINUTOS")),
	}
}

func expiracaoToken(valor string) time.Duration {
	minutos, err := strconv.Atoi(strings.TrimSpace(valor))
	if err != nil || minutos < 1 {
		return expiracaoTokenPadrao
	}

	return time.Duration(minutos) * time.Minute
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
