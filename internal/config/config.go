package config

import (
	"net"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort                   string
	DBHost                     string
	DBPort                     string
	DBUser                     string
	DBPassword                 string
	DBName                     string
	DBURL                      string
	OpenAIAPIKey               string
	OpenAIBaseURL              string
	OpenAIEmbeddingModel       string
	OpenAITimeout              time.Duration
	EmbeddingDim               int
	MemoryImportanceThreshold  float32
	MemoryMaxContentLen        int
	WriteMaxItems              int
	ReadMaxItems               int
	ReadMaxQueryLen            int
}

func Load() *Config {
	DBHost := getEnv("DB_HOST", "localhost")
	DBPort := getEnv("DB_PORT", "5432")
	DBUser := getEnv("DB_USER", "postgres")
	DBPassword := getEnv("DB_PASSWORD", "postgres")
	DBName := getEnv("DB_NAME", "postgres")

	DBURL := os.Getenv("DB_URL")
	if DBURL == "" {
		u := &url.URL{
			Scheme: "postgres",
			User:   url.UserPassword(DBUser, DBPassword),
			Host:   net.JoinHostPort(DBHost, DBPort),
			Path:   DBName,
		}
		DBURL = u.String()
	}

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

	importanceThreshold := float32(0.3)
	if thresholdStr := os.Getenv("MEMORY_IMPORTANCE_THRESHOLD"); thresholdStr != "" {
		if t, err := strconv.ParseFloat(thresholdStr, 32); err == nil && t >= 0 && t <= 1 {
			importanceThreshold = float32(t)
		}
	}

	maxContentLen := 10000
	if maxLenStr := os.Getenv("MEMORY_MAX_CONTENT_LEN"); maxLenStr != "" {
		if l, err := strconv.Atoi(maxLenStr); err == nil && l > 0 {
			maxContentLen = l
		}
	}

	maxItems := 100
	if maxItemsStr := os.Getenv("WRITE_MAX_ITEMS"); maxItemsStr != "" {
		if m, err := strconv.Atoi(maxItemsStr); err == nil && m > 0 {
			maxItems = m
		}
	}

	readMaxItems := 50
	if readMaxItemsStr := os.Getenv("READ_MAX_ITEMS"); readMaxItemsStr != "" {
		if m, err := strconv.Atoi(readMaxItemsStr); err == nil && m > 0 {
			readMaxItems = m
		}
	}

	readMaxQueryLen := 10000
	if readMaxQueryLenStr := os.Getenv("READ_MAX_QUERY_LEN"); readMaxQueryLenStr != "" {
		if l, err := strconv.Atoi(readMaxQueryLenStr); err == nil && l > 0 {
			readMaxQueryLen = l
		}
	}

	return &Config{
		HTTPPort:                  getEnv("HTTP_PORT", ":8080"),
		DBHost:                    DBHost,
		DBPort:                    DBPort,
		DBUser:                    DBUser,
		DBPassword:                DBPassword,
		DBName:                    DBName,
		DBURL:                     DBURL,
		OpenAIAPIKey:              getEnv("OPENAI_API_KEY", ""),
		OpenAIBaseURL:             getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OpenAIEmbeddingModel:      getEnv("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small"),
		OpenAITimeout:             time.Duration(timeout) * time.Second,
		EmbeddingDim:              embeddingDim,
		MemoryImportanceThreshold: importanceThreshold,
		MemoryMaxContentLen:       maxContentLen,
		WriteMaxItems:             maxItems,
		ReadMaxItems:              readMaxItems,
		ReadMaxQueryLen:           readMaxQueryLen,
	}
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
