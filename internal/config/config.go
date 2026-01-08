package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort              string
	DBHost                string
	DBPort                string
	DBUser                string
	DBPassword            string
	DBName                string
	DBURL                 string
	OpenAIAPIKey          string
	OpenAIBaseURL         string
	OpenAIEmbeddingModel  string
	OpenAITimeout         time.Duration
	EmbeddingDim          int
}

func Load() *Config {
	DBHost := getEnv("DB_HOST", "localhost")
	DBPort := getEnv("DB_PORT", "5432")
	DBUser := getEnv("DB_USER", "postgres")
	DBPassword := getEnv("DB_PASSWORD", "postgres")
	DBName := getEnv("DB_NAME", "postgres")
	DBURL := "postgres://" + DBUser + ":" + DBPassword + "@" + DBHost + ":" + DBPort + "/" + DBName

	timeout := 30
	if timeoutStr := os.Getenv("OPENAI_TIMEOUT"); timeoutStr != "" {
		if t, err := strconv.Atoi(timeoutStr); err == nil && t > 0 {
			timeout = t
		}
	}

	embeddingDim := 1536
	if dimStr := os.Getenv("EMBEDDING_DIM"); dimStr != "" {
		if d, err := strconv.Atoi(dimStr); err == nil && d > 0 {
			embeddingDim = d
		}
	}

	return &Config{
		HTTPPort:             getEnv("HTTP_PORT", ":8080"),
		DBHost:               DBHost,
		DBPort:               DBPort,
		DBUser:               DBUser,
		DBPassword:           DBPassword,
		DBName:               DBName,
		DBURL:                DBURL,
		OpenAIAPIKey:         getEnv("OPENAI_API_KEY", ""),
		OpenAIBaseURL:        getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OpenAIEmbeddingModel: getEnv("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small"),
		OpenAITimeout:        time.Duration(timeout) * time.Second,
		EmbeddingDim:         embeddingDim,
	}
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
