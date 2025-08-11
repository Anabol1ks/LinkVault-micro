package config

import (
	"os"
	"strings"

	"go.uber.org/zap"
)

type Config struct {
	Port     string
	DB       DBConfig
	Domain   string
	AuthAddr string

	KafkaBrokers []string
	KafkaTopic   string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func Load(log *zap.Logger) *Config {
	return &Config{
		Port: getEnv("APP_PORT", log),
		DB: DBConfig{
			Host:     getEnv("DB_HOST", log),
			Port:     getEnv("DB_PORT", log),
			User:     getEnv("DB_USER", log),
			Password: getEnv("DB_PASSWORD", log),
			Name:     getEnv("DB_NAME", log),
			SSLMode:  getEnv("DB_SSLMODE", log),
		},
		KafkaBrokers: splitAndTrim(os.Getenv("KAFKA_BROKERS")),
		KafkaTopic:   getEnv("KAFKA_TOPIC_EMAIL", log),

		Domain:   getEnv("DOMAIN", log),
		AuthAddr: getEnv("AUTH_SERVICE_ADDR", log),
	}
}

func getEnv(key string, log *zap.Logger) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	log.Error("Обязательная переменная окружения не установлена", zap.String("key", key))
	panic("missing required environment variable: " + key)
}

func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := []string{}
	for _, p := range strings.Split(s, ",") {
		pt := strings.TrimSpace(p)
		if pt != "" {
			parts = append(parts, pt)
		}
	}
	return parts
}
