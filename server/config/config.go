package config

import (
	"cmp"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment string
	HTTPAddr    string
	LogLevel    string
	CORSOrigins []string
	Database    Database
	Redis       Redis
	JWT         JWT
}

type Database struct {
	Driver          string
	URL             string
	AutoMigrate     bool
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type Redis struct {
	Addr     string
	Password string
	DB       int
}

type JWT struct {
	Secret string
	Issuer string
	TTL    time.Duration
}

func Load() (Config, error) {
	environment := cmp.Or(os.Getenv("APP_ENV"), "development")
	driver := cmp.Or(os.Getenv("DB_DRIVER"), "postgres")
	if driver != "postgres" && driver != "mysql" {
		return Config{}, fmt.Errorf("DB_DRIVER must be postgres or mysql, got %q", driver)
	}

	autoMigrate, err := envBool("AUTO_MIGRATE", true)
	if err != nil {
		return Config{}, err
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" && environment != "production" {
		jwtSecret = "development-only-secret-change-before-release"
	}
	if len(jwtSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}

	jwtTTL, err := envDuration("JWT_TTL", 24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	if jwtTTL <= 0 {
		return Config{}, fmt.Errorf("JWT_TTL must be positive")
	}
	connMaxLifetime, err := envDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute)
	if err != nil {
		return Config{}, err
	}

	redisDB, err := envInt("REDIS_DB", 0)
	if err != nil {
		return Config{}, err
	}
	maxOpen, err := envInt("DB_MAX_OPEN_CONNS", 25)
	if err != nil {
		return Config{}, err
	}
	maxIdle, err := envInt("DB_MAX_IDLE_CONNS", 5)
	if err != nil {
		return Config{}, err
	}
	if maxOpen < 1 || maxIdle < 0 {
		return Config{}, fmt.Errorf("database connection limits must be non-negative and DB_MAX_OPEN_CONNS must be positive")
	}

	return Config{
		Environment: environment,
		HTTPAddr:    cmp.Or(os.Getenv("HTTP_ADDR"), ":8080"),
		LogLevel:    cmp.Or(os.Getenv("LOG_LEVEL"), "info"),
		CORSOrigins: splitList(cmp.Or(os.Getenv("CORS_ORIGINS"), "http://localhost:5173")),
		Database: Database{
			Driver:          driver,
			URL:             cmp.Or(os.Getenv("DATABASE_URL"), defaultDatabaseURL(driver)),
			AutoMigrate:     autoMigrate,
			MaxOpenConns:    maxOpen,
			MaxIdleConns:    maxIdle,
			ConnMaxLifetime: connMaxLifetime,
		},
		Redis: Redis{
			Addr:     cmp.Or(os.Getenv("REDIS_ADDR"), "localhost:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       redisDB,
		},
		JWT: JWT{
			Secret: jwtSecret,
			Issuer: cmp.Or(os.Getenv("JWT_ISSUER"), "go-vue-starter"),
			TTL:    jwtTTL,
		},
	}, nil
}

func defaultDatabaseURL(driver string) string {
	if driver == "mysql" {
		return "app:app@tcp(localhost:3306)/app?parseTime=true&charset=utf8mb4"
	}
	return "postgres://app:app@localhost:5432/app?sslmode=disable"
}

func envBool(name string, fallback bool) (bool, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parsing %s: %w", name, err)
	}
	return parsed, nil
}

func envInt(name string, fallback int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parsing %s: %w", name, err)
	}
	return parsed, nil
}

func envDuration(name string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parsing %s: %w", name, err)
	}
	return parsed, nil
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
