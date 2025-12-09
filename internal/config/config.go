package config

import "os"

type Config struct {
	HTTPPort      string
	DBURL         string
	OpenAIAPIKey  string
	OpenAIBaseURL string
}

func Load() *Config {
	return &Config{
		HTTPPort:      getEnv("HTTP_PORT", ":8080"),
		DBURL:         getEnv("DB_URL", ""),
		OpenAIAPIKey:  getEnv("OPENAI_API_KEY", ""),
		OpenAIBaseURL: getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
	}
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
