package crud

import (
	"fmt"
	"os"
	"strconv"
)

// TestDBConfig represents database configuration for tests
type TestDBConfig struct {
	Driver   string
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

// DefaultMySQLConfig returns default MySQL configuration for tests
func DefaultMySQLConfig() TestDBConfig {
	return TestDBConfig{
		Driver:   "mysql",
		Host:     getEnvOrDefault("TEST_MYSQL_HOST", "192.168.110.249"),
		Port:     getEnvIntOrDefault("TEST_MYSQL_PORT", 3306),
		User:     getEnvOrDefault("TEST_MYSQL_USER", "root"),
		Password: getEnvOrDefault("TEST_MYSQL_PASSWORD", "123456"),
		DBName:   getEnvOrDefault("TEST_MYSQL_DBNAME", "test"),
	}
}

// DefaultPostgresConfig returns default PostgreSQL configuration for tests
func DefaultPostgresConfig() TestDBConfig {
	return TestDBConfig{
		Driver:   "postgres",
		Host:     getEnvOrDefault("TEST_POSTGRES_HOST", "192.168.110.249"),
		Port:     getEnvIntOrDefault("TEST_POSTGRES_PORT", 5432),
		User:     getEnvOrDefault("TEST_POSTGRES_USER", "postgres"),
		Password: getEnvOrDefault("TEST_POSTGRES_PASSWORD", "yzy123"),
		DBName:   getEnvOrDefault("TEST_POSTGRES_DBNAME", "test"),
	}
}

// DSN returns the data source name for the database
func (c TestDBConfig) DSN() string {
	// Check for environment variable override
	envVar := fmt.Sprintf("TEST_%s_DSN", c.Driver)
	if dsn := os.Getenv(envVar); dsn != "" {
		return dsn
	}

	switch c.Driver {
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&multiStatements=true",
			c.User, c.Password, c.Host, c.Port, c.DBName)
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			c.Host, c.Port, c.User, c.Password, c.DBName)
	default:
		return ""
	}
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
