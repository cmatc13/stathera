# Configuration System

This package provides a centralized configuration system for the Stathera application. It supports multiple configuration sources with validation and sensible defaults.

## Features

- **Multiple Configuration Sources**: Load configuration from environment variables, configuration files, and command-line flags.
- **Validation**: Comprehensive validation for all configuration parameters.
- **Sensible Defaults**: Reasonable default values for all configuration parameters.
- **Type Safety**: Strongly typed configuration with proper Go types.
- **Extensibility**: Easy to extend with new configuration parameters.

## Configuration Sources

The configuration system supports the following sources, in order of precedence (highest to lowest):

1. **Command-line Flags**: Highest precedence, overrides all other sources.
2. **Environment Variables**: Overrides configuration files and defaults.
3. **Configuration Files**: Overrides defaults.
4. **Default Values**: Used when no other source provides a value.

## Usage

### Basic Usage

```go
// Load configuration with default options
cfg, err := config.Load()
if err != nil {
    log.Fatalf("Failed to load configuration: %v", err)
}

// Use the configuration
redisClient := redis.NewClient(&redis.Options{
    Addr: cfg.Redis.Address,
    Password: cfg.Redis.Password,
    DB: cfg.Redis.DB,
})
```

### Custom Options

```go
// Set up custom load options
opts := config.DefaultLoadOptions()
opts.ConfigFile = "/path/to/config.json"
opts.EnvPrefix = "MYAPP"

// Load configuration with custom options
cfg, err := config.LoadWithOptions(opts)
if err != nil {
    log.Fatalf("Failed to load configuration: %v", err)
}
```

### Command-line Flags

The configuration system automatically registers and parses command-line flags for all configuration parameters. For example:

```bash
./myapp --redis.address=redis:6379 --api.port=8081
```

### Environment Variables

Environment variables are automatically mapped to configuration parameters using the format `PREFIX_SECTION_PARAMETER`. For example:

```bash
export STATHERA_REDIS_ADDRESS=redis:6379
export STATHERA_API_PORT=8081
```

### Configuration Files

The configuration system supports JSON and YAML configuration files. Example JSON configuration:

```json
{
  "redis": {
    "address": "redis:6379",
    "password": "",
    "db": 0
  },
  "api": {
    "port": "8081",
    "version": "v1"
  }
}
```

## Configuration Parameters

### Redis Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `address` | string | `localhost:6379` | Redis server address |
| `password` | string | `""` | Redis password |
| `db` | int | `0` | Redis database number |
| `max_retries` | int | `3` | Maximum number of retries |
| `pool_size` | int | `10` | Connection pool size |
| `dial_timeout` | duration | `5s` | Dial timeout |

### Kafka Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `brokers` | string | `localhost:9092` | Kafka broker addresses (comma-separated) |
| `consumer_group_id` | string | `transaction_processor_group` | Consumer group ID |
| `transaction_topic` | string | `transactions` | Topic for incoming transactions |
| `confirmed_topic` | string | `confirmed_transactions` | Topic for confirmed transactions |
| `failed_topic` | string | `failed_transactions` | Topic for failed transactions |
| `session_timeout` | duration | `30s` | Session timeout |
| `heartbeat_interval` | duration | `3s` | Heartbeat interval |
| `max_poll_interval` | duration | `5m` | Maximum poll interval |
| `auto_commit_interval` | duration | `5s` | Auto commit interval |
| `producer_max_retries` | int | `3` | Maximum number of producer retries |
| `producer_retry_backoff` | duration | `100ms` | Producer retry backoff |

### API Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `host` | string | `0.0.0.0` | API server host |
| `port` | string | `8080` | API server port |
| `version` | string | `v1` | API version |
| `read_timeout` | duration | `10s` | Read timeout |
| `write_timeout` | duration | `10s` | Write timeout |
| `shutdown_timeout` | duration | `30s` | Shutdown timeout |
| `cors_allowed_origins` | []string | `["*"]` | CORS allowed origins |

### Auth Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `jwt_secret` | string | `your_jwt_secret_here` | JWT secret key |
| `jwt_expiration_time` | duration | `24h` | JWT expiration time |
| `refresh_token_duration` | duration | `168h` | Refresh token duration |

### Supply Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `min_inflation` | float64 | `1.5` | Minimum inflation rate |
| `max_inflation` | float64 | `3.0` | Maximum inflation rate |
| `max_step_size` | float64 | `0.1` | Maximum inflation adjustment step size |
| `reserve_address` | string | `system_reserve_address` | Reserve address for supply management |
| `adjust_interval` | duration | `24h` | Inflation adjustment interval |

### Processor Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `batch_size` | int | `100` | Batch size for processing transactions |
| `poll_interval` | duration | `100ms` | Poll interval for checking new transactions |
| `max_concurrency` | int | `10` | Maximum number of concurrent processing goroutines |

### Log Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `level` | string | `info` | Log level (debug, info, warn, error) |
| `format` | string | `json` | Log format (json, text) |
| `output_path` | string | `stdout` | Log output path |

### Environment

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `env` | string | `development` | Environment (development, staging, production) |
