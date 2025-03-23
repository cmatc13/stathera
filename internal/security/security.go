// internal/security/security.go
package security

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	// Password hashing cost
	bcryptCost = 14

	// Rate limiting keys
	rateLimitKeyPrefix = "ratelimit:"

	// Brute force protection
	failedLoginKeyPrefix   = "failedlogin:"
	maxFailedLoginAttempts = 5
	loginLockoutDuration   = 15 * time.Minute

	// API key prefix
	apiKeyPrefix = "apikey:"

	// CSRF token prefix
	csrfTokenPrefix     = "csrf:"
	csrfTokenExpiration = 1 * time.Hour
)

// SecurityManager handles security-related functionality
type SecurityManager struct {
	client    *redis.Client
	ctx       context.Context
	jwtSecret []byte
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(redisAddr string, jwtSecret string) (*SecurityManager, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0,
	})

	ctx := context.Background()

	// Test connection
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &SecurityManager{
		client:    client,
		ctx:       ctx,
		jwtSecret: []byte(jwtSecret),
	}, nil
}

// Close closes the Redis connection
func (sm *SecurityManager) Close() error {
	return sm.client.Close()
}

// HashPassword securely hashes a password using bcrypt
func (sm *SecurityManager) HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters long")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// VerifyPassword checks if a password matches a hash
func (sm *SecurityManager) VerifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// CreateAPIKey generates a new API key for a user
func (sm *SecurityManager) CreateAPIKey(userID string, permissions []string) (string, error) {
	// Generate random API key
	keyBytes := make([]byte, 32)
	_, err := rand.Read(keyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}

	apiKey := base64.URLEncoding.EncodeToString(keyBytes)

	// Store API key with user info
	keyData := map[string]interface{}{
		"user_id":     userID,
		"permissions": strings.Join(permissions, ","),
		"created_at":  time.Now().Unix(),
	}

	// Hash the API key for storage to prevent key leakage from Redis
	keyHash := sha256.Sum256([]byte(apiKey))
	keyHashStr := base64.StdEncoding.EncodeToString(keyHash[:])

	err = sm.client.HSet(sm.ctx, apiKeyPrefix+keyHashStr, keyData).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store API key: %w", err)
	}

	return apiKey, nil
}

// ValidateAPIKey validates an API key and returns the associated user ID and permissions
func (sm *SecurityManager) ValidateAPIKey(apiKey string) (string, []string, error) {
	// Hash the API key
	keyHash := sha256.Sum256([]byte(apiKey))
	keyHashStr := base64.StdEncoding.EncodeToString(keyHash[:])

	// Get key data
	keyData, err := sm.client.HGetAll(sm.ctx, apiKeyPrefix+keyHashStr).Result()
	if err != nil || len(keyData) == 0 {
		return "", nil, errors.New("invalid API key")
	}

	userID := keyData["user_id"]
	permissionsStr := keyData["permissions"]
	permissions := strings.Split(permissionsStr, ",")

	return userID, permissions, nil
}

// GenerateCSRFToken generates a new CSRF token for a session
func (sm *SecurityManager) GenerateCSRFToken(sessionID string) (string, error) {
	token := uuid.New().String()

	// Store token in Redis with expiration
	err := sm.client.Set(sm.ctx, csrfTokenPrefix+sessionID, token, csrfTokenExpiration).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store CSRF token: %w", err)
	}

	return token, nil
}

// ValidateCSRFToken validates a CSRF token for a session
func (sm *SecurityManager) ValidateCSRFToken(sessionID, token string) bool {
	storedToken, err := sm.client.Get(sm.ctx, csrfTokenPrefix+sessionID).Result()
	if err != nil || storedToken != token {
		return false
	}

	return true
}

// CheckRateLimit checks if a rate limit has been exceeded
// Returns true if the request should be allowed, false if rate limited
func (sm *SecurityManager) CheckRateLimit(key string, limit int, period time.Duration) (bool, error) {
	// Use Redis pipeline for atomic operations
	pipe := sm.client.Pipeline()

	// Increment counter
	countResult := pipe.Incr(sm.ctx, rateLimitKeyPrefix+key)

	// Set expiration if not already set
	pipe.Expire(sm.ctx, rateLimitKeyPrefix+key, period)

	// Execute pipeline
	_, err := pipe.Exec(sm.ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check rate limit: %w", err)
	}

	// Get counter value
	count, err := countResult.Result()
	if err != nil {
		return false, fmt.Errorf("failed to get rate limit counter: %w", err)
	}

	// Check if limit exceeded
	return count <= int64(limit), nil
}

// RecordFailedLogin records a failed login attempt for a user
func (sm *SecurityManager) RecordFailedLogin(userID string) error {
	key := failedLoginKeyPrefix + userID

	// Increment failed login counter
	count, err := sm.client.Incr(sm.ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to record failed login: %w", err)
	}

	// Set expiration if not already set
	if count == 1 {
		err = sm.client.Expire(sm.ctx, key, loginLockoutDuration).Err()
		if err != nil {
			return fmt.Errorf("failed to set expiration for failed login counter: %w", err)
		}
	}

	return nil
}

// CheckLoginAllowed checks if a user is allowed to login (not locked out)
func (sm *SecurityManager) CheckLoginAllowed(userID string) (bool, error) {
	key := failedLoginKeyPrefix + userID

	// Get failed login count
	count, err := sm.client.Get(sm.ctx, key).Int64()
	if err == redis.Nil {
		// No failed logins
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check login allowed: %w", err)
	}

	// Check if exceeded max attempts
	return count < maxFailedLoginAttempts, nil
}

// ResetFailedLogins resets the failed login counter for a user
func (sm *SecurityManager) ResetFailedLogins(userID string) error {
	key := failedLoginKeyPrefix + userID

	err := sm.client.Del(sm.ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to reset failed logins: %w", err)
	}

	return nil
}

// InputValidation validates and sanitizes input according to OWASP guidelines
func (sm *SecurityManager) ValidateAndSanitizeInput(input string, maxLength int, allowedChars string) (string, error) {
	// Check length
	if len(input) > maxLength {
		return "", fmt.Errorf("input exceeds maximum length of %d characters", maxLength)
	}

	// If allowed characters specified, validate against them
	if allowedChars != "" {
		allowedMap := make(map[rune]bool)
		for _, char := range allowedChars {
			allowedMap[char] = true
		}

		for _, char := range input {
			if !allowedMap[char] {
				return "", fmt.Errorf("input contains invalid character: %q", char)
			}
		}
	}

	// Additional sanitization can be implemented based on the specific needs

	return input, nil
}
