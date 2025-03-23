// pkg/config/config.go
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Redis     RedisConfig     `mapstructure:"redis" json:"redis"`
	Kafka     KafkaConfig     `mapstructure:"kafka" json:"kafka"`
	API       APIConfig       `mapstructure:"api" json:"api"`
	Auth      AuthConfig      `mapstructure:"auth" json:"auth"`
	Supply    SupplyConfig    `mapstructure:"supply" json:"supply"`
	Processor ProcessorConfig `mapstructure:"processor" json:"processor"`
	Log       LogConfig       `mapstructure:"log" json:"log"`
	Metrics   MetricsConfig   `mapstructure:"metrics" json:"metrics"`
	Health    HealthConfig    `mapstructure:"health" json:"health"`
	Env       string          `mapstructure:"env" json:"env"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Address     string        `mapstructure:"address" json:"address"`
	Password    string        `mapstructure:"password" json:"password"`
	DB          int           `mapstructure:"db" json:"db"`
	MaxRetries  int           `mapstructure:"max_retries" json:"max_retries"`
	PoolSize    int           `mapstructure:"pool_size" json:"pool_size"`
	DialTimeout time.Duration `mapstructure:"dial_timeout" json:"dial_timeout"`
}

// KafkaConfig represents Kafka configuration
type KafkaConfig struct {
	Brokers              string        `mapstructure:"brokers" json:"brokers"`
	ConsumerGroupID      string        `mapstructure:"consumer_group_id" json:"consumer_group_id"`
	TransactionTopic     string        `mapstructure:"transaction_topic" json:"transaction_topic"`
	ConfirmedTopic       string        `mapstructure:"confirmed_topic" json:"confirmed_topic"`
	FailedTopic          string        `mapstructure:"failed_topic" json:"failed_topic"`
	SessionTimeout       time.Duration `mapstructure:"session_timeout" json:"session_timeout"`
	HeartbeatInterval    time.Duration `mapstructure:"heartbeat_interval" json:"heartbeat_interval"`
	MaxPollInterval      time.Duration `mapstructure:"max_poll_interval" json:"max_poll_interval"`
	AutoCommitInterval   time.Duration `mapstructure:"auto_commit_interval" json:"auto_commit_interval"`
	ProducerMaxRetries   int           `mapstructure:"producer_max_retries" json:"producer_max_retries"`
	ProducerRetryBackoff time.Duration `mapstructure:"producer_retry_backoff" json:"producer_retry_backoff"`
}

// APIConfig represents API server configuration
type APIConfig struct {
	Host               string        `mapstructure:"host" json:"host"`
	Port               string        `mapstructure:"port" json:"port"`
	Version            string        `mapstructure:"version" json:"version"`
	ReadTimeout        time.Duration `mapstructure:"read_timeout" json:"read_timeout"`
	WriteTimeout       time.Duration `mapstructure:"write_timeout" json:"write_timeout"`
	ShutdownTimeout    time.Duration `mapstructure:"shutdown_timeout" json:"shutdown_timeout"`
	CORSAllowedOrigins []string      `mapstructure:"cors_allowed_origins" json:"cors_allowed_origins"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	JWTSecret            string        `mapstructure:"jwt_secret" json:"jwt_secret"`
	JWTExpirationTime    time.Duration `mapstructure:"jwt_expiration_time" json:"jwt_expiration_time"`
	RefreshTokenDuration time.Duration `mapstructure:"refresh_token_duration" json:"refresh_token_duration"`
}

// SupplyConfig represents currency supply management configuration
type SupplyConfig struct {
	MinInflation   float64       `mapstructure:"min_inflation" json:"min_inflation"`
	MaxInflation   float64       `mapstructure:"max_inflation" json:"max_inflation"`
	MaxStepSize    float64       `mapstructure:"max_step_size" json:"max_step_size"`
	ReserveAddress string        `mapstructure:"reserve_address" json:"reserve_address"`
	AdjustInterval time.Duration `mapstructure:"adjust_interval" json:"adjust_interval"`
}

// ProcessorConfig represents transaction processor configuration
type ProcessorConfig struct {
	BatchSize      int           `mapstructure:"batch_size" json:"batch_size"`
	PollInterval   time.Duration `mapstructure:"poll_interval" json:"poll_interval"`
	MaxConcurrency int           `mapstructure:"max_concurrency" json:"max_concurrency"`
}

// LogConfig represents logging configuration
type LogConfig struct {
	Level        string `mapstructure:"level" json:"level"`
	Format       string `mapstructure:"format" json:"format"`
	OutputPath   string `mapstructure:"output_path" json:"output_path"`
	ServiceName  string `mapstructure:"service_name" json:"service_name"`
	Environment  string `mapstructure:"environment" json:"environment"`
	IncludeTrace bool   `mapstructure:"include_trace" json:"include_trace"`
}

// MetricsConfig represents metrics collection configuration
type MetricsConfig struct {
	Enabled     bool   `mapstructure:"enabled" json:"enabled"`
	Namespace   string `mapstructure:"namespace" json:"namespace"`
	ServiceName string `mapstructure:"service_name" json:"service_name"`
	Endpoint    string `mapstructure:"endpoint" json:"endpoint"`
	Port        string `mapstructure:"port" json:"port"`
}

// HealthConfig represents health check configuration
type HealthConfig struct {
	Enabled  bool   `mapstructure:"enabled" json:"enabled"`
	Endpoint string `mapstructure:"endpoint" json:"endpoint"`
	Port     string `mapstructure:"port" json:"port"`
	Interval string `mapstructure:"interval" json:"interval"`
}

// LoadOptions contains options for loading configuration
type LoadOptions struct {
	ConfigFile     string
	EnvPrefix      string
	FlagPrefix     string
	UseFlags       bool
	UseEnv         bool
	UseConfigFile  bool
	DefaultConfigs []string
}

// DefaultLoadOptions returns the default load options
func DefaultLoadOptions() LoadOptions {
	return LoadOptions{
		ConfigFile:    "",
		EnvPrefix:     "STATHERA",
		FlagPrefix:    "",
		UseFlags:      true,
		UseEnv:        true,
		UseConfigFile: true,
		DefaultConfigs: []string{
			"./config.yaml",
			"./config.json",
			"./config/config.yaml",
			"./config/config.json",
		},
	}
}

// Load loads the configuration from various sources with default options
func Load() (*Config, error) {
	return LoadWithOptions(DefaultLoadOptions())
}

// LoadWithOptions loads the configuration from various sources with custom options
func LoadWithOptions(opts LoadOptions) (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Try to load .env file if it exists
	if opts.UseEnv {
		godotenv.Load()
	}

	// Load from config file if specified
	if opts.UseConfigFile {
		if opts.ConfigFile != "" {
			v.SetConfigFile(opts.ConfigFile)
		} else {
			// Try default config locations
			for _, configPath := range opts.DefaultConfigs {
				if _, err := os.Stat(configPath); err == nil {
					v.SetConfigFile(configPath)
					break
				}
			}
		}

		if v.ConfigFileUsed() != "" {
			if err := v.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
					return nil, fmt.Errorf("error reading config file: %w", err)
				}
			}
		}
	}

	// Load from environment variables
	if opts.UseEnv {
		v.SetEnvPrefix(opts.EnvPrefix)
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.AutomaticEnv()
	}

	// Load from command line flags
	if opts.UseFlags {
		if err := bindFlags(v, opts.FlagPrefix); err != nil {
			return nil, fmt.Errorf("error binding flags: %w", err)
		}
	}

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate config
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	// Redis defaults
	v.SetDefault("redis.address", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.max_retries", 3)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.dial_timeout", 5*time.Second)

	// Kafka defaults
	v.SetDefault("kafka.brokers", "localhost:9092")
	v.SetDefault("kafka.consumer_group_id", "transaction_processor_group")
	v.SetDefault("kafka.transaction_topic", "transactions")
	v.SetDefault("kafka.confirmed_topic", "confirmed_transactions")
	v.SetDefault("kafka.failed_topic", "failed_transactions")
	v.SetDefault("kafka.session_timeout", 30*time.Second)
	v.SetDefault("kafka.heartbeat_interval", 3*time.Second)
	v.SetDefault("kafka.max_poll_interval", 5*time.Minute)
	v.SetDefault("kafka.auto_commit_interval", 5*time.Second)
	v.SetDefault("kafka.producer_max_retries", 3)
	v.SetDefault("kafka.producer_retry_backoff", 100*time.Millisecond)

	// API defaults
	v.SetDefault("api.host", "0.0.0.0")
	v.SetDefault("api.port", "8080")
	v.SetDefault("api.version", "v1")
	v.SetDefault("api.read_timeout", 10*time.Second)
	v.SetDefault("api.write_timeout", 10*time.Second)
	v.SetDefault("api.shutdown_timeout", 30*time.Second)
	v.SetDefault("api.cors_allowed_origins", []string{"*"})

	// Auth defaults
	v.SetDefault("auth.jwt_secret", "your_jwt_secret_here")
	v.SetDefault("auth.jwt_expiration_time", 24*time.Hour)
	v.SetDefault("auth.refresh_token_duration", 7*24*time.Hour)

	// Supply defaults
	v.SetDefault("supply.min_inflation", 1.5)
	v.SetDefault("supply.max_inflation", 3.0)
	v.SetDefault("supply.max_step_size", 0.1)
	v.SetDefault("supply.reserve_address", "system_reserve_address")
	v.SetDefault("supply.adjust_interval", 24*time.Hour)

	// Processor defaults
	v.SetDefault("processor.batch_size", 100)
	v.SetDefault("processor.poll_interval", 100*time.Millisecond)
	v.SetDefault("processor.max_concurrency", 10)

	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output_path", "stdout")
	v.SetDefault("log.service_name", "stathera")
	v.SetDefault("log.environment", "development")
	v.SetDefault("log.include_trace", true)

	// Metrics defaults
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.namespace", "stathera")
	v.SetDefault("metrics.service_name", "stathera")
	v.SetDefault("metrics.endpoint", "/metrics")
	v.SetDefault("metrics.port", "9090")

	// Health defaults
	v.SetDefault("health.enabled", true)
	v.SetDefault("health.endpoint", "/health")
	v.SetDefault("health.port", "8081")
	v.SetDefault("health.interval", "30s")

	// Environment defaults
	v.SetDefault("env", "development")
}

// bindFlags binds command line flags to viper
func bindFlags(v *viper.Viper, prefix string) error {
	flags := pflag.NewFlagSet("config", pflag.ContinueOnError)

	// Define flags
	flags.String(prefix+"config", "", "Path to config file")
	flags.String(prefix+"env", "development", "Environment (development, staging, production)")

	// Redis flags
	flags.String(prefix+"redis.address", "localhost:6379", "Redis server address")
	flags.String(prefix+"redis.password", "", "Redis password")
	flags.Int(prefix+"redis.db", 0, "Redis database number")

	// Kafka flags
	flags.String(prefix+"kafka.brokers", "localhost:9092", "Kafka broker addresses (comma-separated)")

	// API flags
	flags.String(prefix+"api.port", "8080", "API server port")
	flags.String(prefix+"api.version", "v1", "API version")

	// Auth flags
	flags.String(prefix+"auth.jwt_secret", "", "JWT secret key")

	// Supply flags
	flags.Float64(prefix+"supply.min_inflation", 1.5, "Minimum inflation rate")
	flags.Float64(prefix+"supply.max_inflation", 3.0, "Maximum inflation rate")
	flags.Float64(prefix+"supply.max_step_size", 0.1, "Maximum inflation adjustment step size")
	flags.String(prefix+"supply.reserve_address", "system_reserve_address", "Reserve address for supply management")

	// Log flags
	flags.String(prefix+"log.level", "info", "Log level (debug, info, warn, error)")
	flags.String(prefix+"log.format", "json", "Log format (json, text)")
	flags.String(prefix+"log.service_name", "stathera", "Service name for logging")
	flags.String(prefix+"log.environment", "development", "Environment for logging")
	flags.Bool(prefix+"log.include_trace", true, "Include stack traces in error logs")

	// Metrics flags
	flags.Bool(prefix+"metrics.enabled", true, "Enable metrics collection")
	flags.String(prefix+"metrics.namespace", "stathera", "Metrics namespace")
	flags.String(prefix+"metrics.service_name", "stathera", "Service name for metrics")
	flags.String(prefix+"metrics.endpoint", "/metrics", "Metrics endpoint")
	flags.String(prefix+"metrics.port", "9090", "Metrics server port")

	// Health flags
	flags.Bool(prefix+"health.enabled", true, "Enable health checks")
	flags.String(prefix+"health.endpoint", "/health", "Health check endpoint")
	flags.String(prefix+"health.port", "8081", "Health check server port")
	flags.String(prefix+"health.interval", "30s", "Health check interval")

	// Parse flags
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}

	// Bind flags to viper
	if err := v.BindPFlags(flags); err != nil {
		return err
	}

	return nil
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	var validationErrors []string

	// Validate Redis configuration
	if cfg.Redis.Address == "" {
		validationErrors = append(validationErrors, "redis.address cannot be empty")
	} else if _, err := net.ResolveTCPAddr("tcp", cfg.Redis.Address); err != nil {
		validationErrors = append(validationErrors, fmt.Sprintf("invalid redis.address: %v", err))
	}

	if cfg.Redis.DB < 0 {
		validationErrors = append(validationErrors, "redis.db must be non-negative")
	}

	if cfg.Redis.MaxRetries < 0 {
		validationErrors = append(validationErrors, "redis.max_retries must be non-negative")
	}

	if cfg.Redis.PoolSize <= 0 {
		validationErrors = append(validationErrors, "redis.pool_size must be positive")
	}

	if cfg.Redis.DialTimeout <= 0 {
		validationErrors = append(validationErrors, "redis.dial_timeout must be positive")
	}

	// Validate Kafka configuration
	if cfg.Kafka.Brokers == "" {
		validationErrors = append(validationErrors, "kafka.brokers cannot be empty")
	}

	if cfg.Kafka.ConsumerGroupID == "" {
		validationErrors = append(validationErrors, "kafka.consumer_group_id cannot be empty")
	}

	if cfg.Kafka.TransactionTopic == "" {
		validationErrors = append(validationErrors, "kafka.transaction_topic cannot be empty")
	}

	if cfg.Kafka.ConfirmedTopic == "" {
		validationErrors = append(validationErrors, "kafka.confirmed_topic cannot be empty")
	}

	if cfg.Kafka.FailedTopic == "" {
		validationErrors = append(validationErrors, "kafka.failed_topic cannot be empty")
	}

	if cfg.Kafka.SessionTimeout <= 0 {
		validationErrors = append(validationErrors, "kafka.session_timeout must be positive")
	}

	if cfg.Kafka.HeartbeatInterval <= 0 {
		validationErrors = append(validationErrors, "kafka.heartbeat_interval must be positive")
	}

	if cfg.Kafka.MaxPollInterval <= 0 {
		validationErrors = append(validationErrors, "kafka.max_poll_interval must be positive")
	}

	if cfg.Kafka.ProducerMaxRetries < 0 {
		validationErrors = append(validationErrors, "kafka.producer_max_retries must be non-negative")
	}

	// Validate API configuration
	if cfg.API.Port == "" {
		validationErrors = append(validationErrors, "api.port cannot be empty")
	} else if port, err := strconv.Atoi(cfg.API.Port); err != nil || port <= 0 || port > 65535 {
		validationErrors = append(validationErrors, "api.port must be a valid port number (1-65535)")
	}

	if cfg.API.Version == "" {
		validationErrors = append(validationErrors, "api.version cannot be empty")
	}

	if cfg.API.ReadTimeout <= 0 {
		validationErrors = append(validationErrors, "api.read_timeout must be positive")
	}

	if cfg.API.WriteTimeout <= 0 {
		validationErrors = append(validationErrors, "api.write_timeout must be positive")
	}

	if cfg.API.ShutdownTimeout <= 0 {
		validationErrors = append(validationErrors, "api.shutdown_timeout must be positive")
	}

	// Validate Auth configuration
	if cfg.Env == "production" && cfg.Auth.JWTSecret == "your_jwt_secret_here" {
		validationErrors = append(validationErrors, "auth.jwt_secret must be set in production environment")
	}

	if cfg.Auth.JWTExpirationTime <= 0 {
		validationErrors = append(validationErrors, "auth.jwt_expiration_time must be positive")
	}

	if cfg.Auth.RefreshTokenDuration <= 0 {
		validationErrors = append(validationErrors, "auth.refresh_token_duration must be positive")
	}

	// Validate Supply configuration
	if cfg.Supply.MinInflation < 0 {
		validationErrors = append(validationErrors, "supply.min_inflation must be non-negative")
	}

	if cfg.Supply.MaxInflation < cfg.Supply.MinInflation {
		validationErrors = append(validationErrors, "supply.max_inflation must be greater than or equal to supply.min_inflation")
	}

	if cfg.Supply.MaxStepSize <= 0 {
		validationErrors = append(validationErrors, "supply.max_step_size must be positive")
	}

	if cfg.Supply.ReserveAddress == "" {
		validationErrors = append(validationErrors, "supply.reserve_address cannot be empty")
	}

	if cfg.Supply.AdjustInterval <= 0 {
		validationErrors = append(validationErrors, "supply.adjust_interval must be positive")
	}

	// Validate Processor configuration
	if cfg.Processor.BatchSize <= 0 {
		validationErrors = append(validationErrors, "processor.batch_size must be positive")
	}

	if cfg.Processor.PollInterval <= 0 {
		validationErrors = append(validationErrors, "processor.poll_interval must be positive")
	}

	if cfg.Processor.MaxConcurrency <= 0 {
		validationErrors = append(validationErrors, "processor.max_concurrency must be positive")
	}

	// Validate Log configuration
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[strings.ToLower(cfg.Log.Level)] {
		validationErrors = append(validationErrors, "log.level must be one of: debug, info, warn, error")
	}

	validLogFormats := map[string]bool{"json": true, "text": true}
	if !validLogFormats[strings.ToLower(cfg.Log.Format)] {
		validationErrors = append(validationErrors, "log.format must be one of: json, text")
	}

	if cfg.Log.ServiceName == "" {
		validationErrors = append(validationErrors, "log.service_name cannot be empty")
	}

	// Validate Metrics configuration
	if cfg.Metrics.Enabled {
		if cfg.Metrics.Namespace == "" {
			validationErrors = append(validationErrors, "metrics.namespace cannot be empty when metrics are enabled")
		}

		if cfg.Metrics.ServiceName == "" {
			validationErrors = append(validationErrors, "metrics.service_name cannot be empty when metrics are enabled")
		}

		if cfg.Metrics.Endpoint == "" {
			validationErrors = append(validationErrors, "metrics.endpoint cannot be empty when metrics are enabled")
		}

		if cfg.Metrics.Port == "" {
			validationErrors = append(validationErrors, "metrics.port cannot be empty when metrics are enabled")
		} else if port, err := strconv.Atoi(cfg.Metrics.Port); err != nil || port <= 0 || port > 65535 {
			validationErrors = append(validationErrors, "metrics.port must be a valid port number (1-65535)")
		}
	}

	// Validate Health configuration
	if cfg.Health.Enabled {
		if cfg.Health.Endpoint == "" {
			validationErrors = append(validationErrors, "health.endpoint cannot be empty when health checks are enabled")
		}

		if cfg.Health.Port == "" {
			validationErrors = append(validationErrors, "health.port cannot be empty when health checks are enabled")
		} else if port, err := strconv.Atoi(cfg.Health.Port); err != nil || port <= 0 || port > 65535 {
			validationErrors = append(validationErrors, "health.port must be a valid port number (1-65535)")
		}

		if cfg.Health.Interval == "" {
			validationErrors = append(validationErrors, "health.interval cannot be empty when health checks are enabled")
		} else if _, err := time.ParseDuration(cfg.Health.Interval); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("invalid health.interval: %v", err))
		}
	}

	// Return validation errors if any
	if len(validationErrors) > 0 {
		return errors.New(strings.Join(validationErrors, "; "))
	}

	return nil
}

// SaveToFile saves the configuration to a file
func SaveToFile(cfg *Config, filePath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Determine file format based on extension
	var data []byte
	var err error

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".json":
		data, err = json.MarshalIndent(cfg, "", "  ")
	default:
		return fmt.Errorf("unsupported file format: %s", filepath.Ext(filePath))
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadFromFile loads the configuration from a file
func LoadFromFile(filePath string) (*Config, error) {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	// Read file
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Determine file format based on extension
	var cfg Config

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported file format: %s", filepath.Ext(filePath))
	}

	// Validate config
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	return &cfg, nil
}

// GetEnv gets an environment variable or returns a default value
func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetEnvInt gets an environment variable as an integer or returns a default value
func GetEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// GetEnvFloat gets an environment variable as a float or returns a default value
func GetEnvFloat(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue
	}

	return value
}

// GetEnvBool gets an environment variable as a boolean or returns a default value
func GetEnvBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// GetEnvDuration gets an environment variable as a duration or returns a default value
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
