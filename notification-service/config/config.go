package config

import (
	"os"
	"strconv"

	"go.uber.org/zap"
)

type Config struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string

	TMPLDir string
}

func Load(log *zap.Logger) *Config {
	return &Config{
		SMTPHost:     getEnv("SMTP_HOST", log),
		SMTPPort:     getEnvInt("SMTP_PORT", log),
		SMTPUser:     getEnv("SMTP_USER", log),
		SMTPPassword: getEnv("SMTP_PASSWORD", log),
		SMTPFrom:     getEnv("SMTP_FROM", log),
		TMPLDir:      getEnv("TMPL_DIR", log),
	}
}

func getEnv(key string, log *zap.Logger) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	log.Error("Обязательная переменная окружения не установлена", zap.String("key", key))
	panic("missing required environment variable: " + key)
}

func getEnvInt(key string, log *zap.Logger) int {
	valStr := getEnv(key, log)
	val, err := strconv.Atoi(valStr)
	if err != nil {
		log.Error("Ошибка преобразования переменной окружения в int", zap.String("key", key), zap.Error(err))
		panic("invalid int value for environment variable: " + key)
	}
	return val
}
