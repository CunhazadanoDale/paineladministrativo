package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const expiracaoTokenPadrao = 480 * time.Minute

const storageDiscoPadrao = "disco"

type Config struct {
	AppPort           string
	DatabaseUrl       string
	CORSOrigins       []string
	JWTSecret         string
	JWTExpiracao      time.Duration
	StorageDriver     string
	StorageDir        string
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2Bucket          string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	return &Config{
		AppPort:           os.Getenv("PORT"),
		DatabaseUrl:       os.Getenv("DATABASE_URL"),
		CORSOrigins:       separarOrigens(os.Getenv("CORS_ORIGINS")),
		JWTSecret:         strings.TrimSpace(os.Getenv("JWT_SECRET")),
		JWTExpiracao:      expiracaoToken(os.Getenv("JWT_EXPIRA_MINUTOS")),
		StorageDriver:     storageDriver(os.Getenv("STORAGE_DRIVER")),
		StorageDir:        valorOuPadrao(os.Getenv("STORAGE_DIR"), "storage_local"),
		R2AccountID:       strings.TrimSpace(os.Getenv("R2_ACCOUNT_ID")),
		R2AccessKeyID:     strings.TrimSpace(os.Getenv("R2_ACCESS_KEY_ID")),
		R2SecretAccessKey: strings.TrimSpace(os.Getenv("R2_SECRET_ACCESS_KEY")),
		R2Bucket:          strings.TrimSpace(os.Getenv("R2_BUCKET")),
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

func storageDriver(valor string) string {
	valor = strings.ToLower(strings.TrimSpace(valor))
	if valor == "" {
		return storageDiscoPadrao
	}

	return valor
}

func valorOuPadrao(valor, padrao string) string {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return padrao
	}

	return valor
}
