package config

import "os"

type Config struct {
	HTTPPort      string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBURL         string
	OpenAIAPIKey  string
	OpenAIBaseURL string
}

func Load() *Config {
	DBHost := getEnv("DB_HOST", "localhost")
	DBPort := getEnv("DB_PORT", "5432")
	DBUser := getEnv("DB_USER", "postgres")
	DBPassword := getEnv("DB_PASSWORD", "postgres")
	DBName := getEnv("DB_NAME", "postgres")
	DBURL := "postgres://" + DBUser + ":" + DBPassword + "@" + DBHost + ":" + DBPort + "/" + DBName
	return &Config{
		HTTPPort:      getEnv("HTTP_PORT", ":8080"),
		DBHost:         DBHost,
		DBPort:         DBPort,
		DBUser:         DBUser,
		DBPassword:     DBPassword,
		DBName:         DBName,
		DBURL:          DBURL,
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
