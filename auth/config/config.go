package config

import (
	"log"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
)

type Config struct {
	JWT JWTConfig
	DB  DBConfig
}

type JWTConfig struct {
	Access     string
	AccessExp  time.Duration
	Refresh    string
	RefreshExp time.Duration
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
		DB: DBConfig{
			Host:     getEnv("DB_HOST", log),
			Port:     getEnv("DB_PORT", log),
			User:     getEnv("DB_USER", log),
			Password: getEnv("DB_PASSWORD", log),
			Name:     getEnv("DB_NAME", log),
			SSLMode:  getEnv("DB_SSLMODE", log),
		},
		JWT: JWTConfig{
			Access:     getEnv("ACCESS_SECRET", log),
			AccessExp:  parseDurationWithDays(getEnv("ACCESS_EXP", log)),
			Refresh:    getEnv("REFRESH_SECRET", log),
			RefreshExp: parseDurationWithDays(getEnv("REFRESH_EXP", log)),
		},
	}
}

func getEnv(key string, log *zap.Logger) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	log.Error("Обязательная переменная окружения не установлена", zap.String("key", key))
	panic("missing required environment variable: " + key)
}

func parseDurationWithDays(s string) time.Duration {
	if strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(s, "d")
		days, err := time.ParseDuration(daysStr + "h")
		if err != nil {
			log.Printf("Ошибка парсинга TTL: %v", err)
			return 0
		}
		return time.Duration(24) * days
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return duration
}
