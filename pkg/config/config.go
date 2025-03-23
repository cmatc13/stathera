// pkg/config/config.go
package config

import (
    "fmt"
    "os"
    "strconv"
    "strings"
)

// Config holds all configuration for the application
type Config struct {
    API      APIConfig
    Redis    RedisConfig
    Kafka    KafkaConfig
    Auth     AuthConfig
    Inflation InflationConfig
}

// APIConfig holds API-related configuration
type APIConfig struct {
    Port    string
    Version string
}

// RedisConfig holds Redis-related configuration
type RedisConfig struct {
    Address  string
    Password string
    DB       int
}

// KafkaConfig holds Kafka-related configuration
type KafkaConfig struct {
    Brokers       string
    ConsumerGroup string
}

// AuthConfig holds authentication-related configuration
type AuthConfig struct {
    JWTSecret  string
    TokenExpiry int64
}

// InflationConfig holds inflation-related configuration
type InflationConfig struct {
    MinRate       float64
    MaxRate       float64
    MaxStep       float64
    ReserveAddress string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
    config := &Config{
        API: APIConfig{
            Port:    getEnv("API_PORT", "8080"),
            Version: getEnv("API_VERSION", "v1"),
        },
        Redis: RedisConfig{
            Address:  getEnv("REDIS_ADDRESS", "localhost:6379"),
            Password: getEnv("REDIS_PASSWORD", ""),
            DB:       getIntEnv("REDIS_DB", 0),
        },
        Kafka: KafkaConfig{
            Brokers:       getEnv("KAFKA_BROKERS", "localhost:9092"),
            ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "stathera"),
        },
        Auth: AuthConfig{
            JWTSecret:  getEnv("JWT_SECRET", "your_jwt_secret_here"),
            TokenExpiry: getInt64Env("TOKEN_EXPIRY", 86400), // 24 hours in seconds
        },
        Inflation: InflationConfig{
            MinRate:       getFloat64Env("MIN_INFLATION", 1.5),
            MaxRate:       getFloat64Env("MAX_INFLATION", 3.0),
            MaxStep:       getFloat64Env("MAX_STEP_SIZE", 0.1),
            ReserveAddress: getEnv("RESERVE_ADDRESS", "system_reserve_address"),
        },
    }

    return config, nil
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    return value
}

func getIntEnv(key string, defaultValue int) int {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    intValue, err := strconv.Atoi(value)
    if err != nil {
        return defaultValue
    }
    return intValue
}

func getInt64Env(key string, defaultValue int64) int64 {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    int64Value, err := strconv.ParseInt(value, 10, 64)
    if err != nil {
        return defaultValue
    }
    return int64Value
}

func getFloat64Env(key string, defaultValue float64) float64 {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    float64Value, err := strconv.ParseFloat(value, 64)
    if err != nil {
        return defaultValue
    }
    return float64Value
}

func getBoolEnv(key string, defaultValue bool) bool {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    boolValue, err := strconv.ParseBool(value)
    if err != nil {
        return defaultValue
    }
    return boolValue
}