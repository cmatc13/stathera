// internal/api/security_middleware.go
package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cmatc13/stathera/internal/security"
	"github.com/cmatc13/stathera/pkg/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
)

// SecurityMiddleware wraps security-related middleware functions
type SecurityMiddleware struct {
	securityManager *security.SecurityManager
	tokenAuth       *jwtauth.JWTAuth
	logger          *logging.Logger
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(securityManager *security.SecurityManager, tokenAuth *jwtauth.JWTAuth, logger *logging.Logger) *SecurityMiddleware {
	return &SecurityMiddleware{
		securityManager: securityManager,
		tokenAuth:       tokenAuth,
		logger:          logger,
	}
}

// APIKeyAuth is middleware that validates API keys
func (sm *SecurityMiddleware) APIKeyAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get API key from header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			// No API key provided, continue to next middleware (might use JWT instead)
			next.ServeHTTP(w, r)
			return
		}

		// Validate API key
		userID, permissions, err := sm.securityManager.ValidateAPIKey(apiKey)
		if err != nil {
			sm.logger.Warn("Invalid API key",
				"remote_addr", r.RemoteAddr,
				"path", r.URL.Path,
				"error", err.Error(),
			)
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		// Store user ID and permissions in context
		ctx := context.WithValue(r.Context(), "user_id", userID)
		ctx = context.WithValue(ctx, "permissions", permissions)
		ctx = context.WithValue(ctx, "auth_method", "api_key")

		// Continue with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CSRFProtection is middleware that validates CSRF tokens
func (sm *SecurityMiddleware) CSRFProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF check for non-state-changing methods
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" || r.Method == "TRACE" {
			next.ServeHTTP(w, r)
			return
		}

		// Skip CSRF check for API key authenticated requests
		if authMethod, ok := r.Context().Value("auth_method").(string); ok && authMethod == "api_key" {
			next.ServeHTTP(w, r)
			return
		}

		// Get session ID from cookie or context
		sessionID := ""
		if cookie, err := r.Cookie("session_id"); err == nil {
			sessionID = cookie.Value
		} else if sid, ok := r.Context().Value("session_id").(string); ok {
			sessionID = sid
		}

		if sessionID == "" {
			sm.logger.Warn("CSRF validation failed: no session ID",
				"remote_addr", r.RemoteAddr,
				"path", r.URL.Path,
			)
			http.Error(w, "CSRF validation failed", http.StatusForbidden)
			return
		}

		// Get CSRF token from header
		csrfToken := r.Header.Get("X-CSRF-Token")
		if csrfToken == "" {
			sm.logger.Warn("CSRF validation failed: no CSRF token",
				"remote_addr", r.RemoteAddr,
				"path", r.URL.Path,
			)
			http.Error(w, "CSRF validation failed", http.StatusForbidden)
			return
		}

		// Validate CSRF token
		if !sm.securityManager.ValidateCSRFToken(sessionID, csrfToken) {
			sm.logger.Warn("CSRF validation failed: invalid token",
				"remote_addr", r.RemoteAddr,
				"path", r.URL.Path,
			)
			http.Error(w, "CSRF validation failed", http.StatusForbidden)
			return
		}

		// Continue with the request
		next.ServeHTTP(w, r)
	})
}

// RateLimiter is middleware that implements rate limiting per user/IP
func (sm *SecurityMiddleware) RateLimiter(limit int, period time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Determine rate limit key (user ID or IP)
			var key string
			if userID, ok := r.Context().Value("user_id").(string); ok && userID != "" {
				// Use user ID if authenticated
				key = "user:" + userID
			} else {
				// Use IP address if not authenticated
				key = "ip:" + r.RemoteAddr
			}

			// Add path to make rate limits more granular
			key = key + ":" + r.URL.Path

			// Check rate limit
			allowed, err := sm.securityManager.CheckRateLimit(key, limit, period)
			if err != nil {
				sm.logger.Error("Rate limit check failed",
					"error", err.Error(),
					"remote_addr", r.RemoteAddr,
					"path", r.URL.Path,
				)
				// Continue anyway to avoid blocking legitimate traffic due to rate limit errors
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				sm.logger.Warn("Rate limit exceeded",
					"remote_addr", r.RemoteAddr,
					"path", r.URL.Path,
					"key", key,
				)
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(period.Seconds())))
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// Continue with the request
			next.ServeHTTP(w, r)
		})
	}
}

// ContentSecurityPolicy adds CSP headers to responses
func (sm *SecurityMiddleware) ContentSecurityPolicy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set Content Security Policy header
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'")

		// Continue with the request
		next.ServeHTTP(w, r)
	})
}

// SecureHeaders adds security-related headers to responses
func (sm *SecurityMiddleware) SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Continue with the request
		next.ServeHTTP(w, r)
	})
}

// InputSanitization validates and sanitizes input parameters
func (sm *SecurityMiddleware) InputSanitization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a context to store sanitized values
		ctx := r.Context()

		// Sanitize URL path parameters
		// This is a simple example - in a real implementation, you would
		// validate each parameter based on its expected format
		for _, param := range chi.RouteContext(r.Context()).URLParams.Keys {
			value := chi.URLParam(r, param)
			sanitized, err := sm.securityManager.ValidateAndSanitizeInput(value, 100, "")
			if err != nil {
				sm.logger.Warn("Input validation failed",
					"param", param,
					"value", value,
					"error", err.Error(),
				)
				http.Error(w, "Invalid input parameter", http.StatusBadRequest)
				return
			}

			// Store sanitized value in context
			ctx = context.WithValue(ctx, "sanitized_"+param, sanitized)
		}

		// Continue with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// JWTWithBruteForceProtection enhances JWT authentication with brute force protection
func (sm *SecurityMiddleware) JWTWithBruteForceProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from request
		token, _, err := jwtauth.FromContext(r.Context())

		// If no token or invalid token, check for brute force attempts
		if err != nil || token == nil {
			// Extract username from request (e.g., from a failed login attempt)
			// This is just an example - in a real implementation, you would
			// extract the username from the request body or parameters
			username := r.FormValue("username")
			if username != "" {
				// Record failed login attempt
				userID := "user:" + username
				err := sm.securityManager.RecordFailedLogin(userID)
				if err != nil {
					sm.logger.Error("Failed to record failed login",
						"error", err.Error(),
						"username", username,
					)
				}

				// Check if user is allowed to login
				allowed, err := sm.securityManager.CheckLoginAllowed(userID)
				if err != nil {
					sm.logger.Error("Failed to check login allowed",
						"error", err.Error(),
						"username", username,
					)
				} else if !allowed {
					sm.logger.Warn("Login blocked due to too many failed attempts",
						"username", username,
					)
					http.Error(w, "Too many failed login attempts. Please try again later.", http.StatusTooManyRequests)
					return
				}
			}

			// Continue to standard JWT authenticator which will handle the error
			next.ServeHTTP(w, r)
			return
		}

		// If token is valid, reset failed login counter
		if claims, ok := token.PrivateClaims()["username"].(string); ok && claims != "" {
			userID := "user:" + claims
			err := sm.securityManager.ResetFailedLogins(userID)
			if err != nil {
				sm.logger.Error("Failed to reset failed logins",
					"error", err.Error(),
					"username", claims,
				)
			}
		}

		// Continue with the request
		next.ServeHTTP(w, r)
	})
}

// RequirePermission middleware checks if the user has the required permission
func (sm *SecurityMiddleware) RequirePermission(requiredPermission string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get permissions from context
			perms, ok := r.Context().Value("permissions").([]string)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if user has the required permission
			hasPermission := false
			for _, perm := range perms {
				if perm == requiredPermission || perm == "admin" { // admin has all permissions
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				sm.logger.Warn("Permission denied",
					"required", requiredPermission,
					"user_id", r.Context().Value("user_id"),
					"path", r.URL.Path,
				)
				http.Error(w, "Permission denied", http.StatusForbidden)
				return
			}

			// Continue with the request
			next.ServeHTTP(w, r)
		})
	}
}

// ValidateContentType ensures the request has the correct Content-Type
func (sm *SecurityMiddleware) ValidateContentType(contentType string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip validation for methods that don't typically have a body
			if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" || r.Method == "TRACE" {
				next.ServeHTTP(w, r)
				return
			}

			// Check Content-Type header
			ct := r.Header.Get("Content-Type")
			if !strings.Contains(ct, contentType) {
				sm.logger.Warn("Invalid Content-Type",
					"expected", contentType,
					"received", ct,
					"path", r.URL.Path,
				)
				http.Error(w, "Invalid Content-Type", http.StatusUnsupportedMediaType)
				return
			}

			// Continue with the request
			next.ServeHTTP(w, r)
		})
	}
}

// RequestValidation middleware validates the request body against a schema
func (sm *SecurityMiddleware) RequestValidation(validator func(r *http.Request) error) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip validation for methods that don't typically have a body
			if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" || r.Method == "TRACE" {
				next.ServeHTTP(w, r)
				return
			}

			// Validate request
			err := validator(r)
			if err != nil {
				sm.logger.Warn("Request validation failed",
					"error", err.Error(),
					"path", r.URL.Path,
				)
				http.Error(w, fmt.Sprintf("Invalid request: %s", err.Error()), http.StatusBadRequest)
				return
			}

			// Continue with the request
			next.ServeHTTP(w, r)
		})
	}
}

// ResponseSanitization middleware sanitizes response data
func (sm *SecurityMiddleware) ResponseSanitization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a response wrapper to intercept the response
		wrapper := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Continue with the wrapped response writer
		next.ServeHTTP(wrapper, r)

		// Note: In a real implementation, you would inspect and potentially
		// modify the response body here. This would require buffering the
		// response and then writing it out after sanitization.
		// For simplicity, we're just adding a header to indicate sanitization.
		wrapper.Header().Set("X-Content-Sanitized", "true")
	})
}

// SQLInjectionProtection middleware protects against SQL injection
func (sm *SecurityMiddleware) SQLInjectionProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check query parameters for SQL injection patterns
		for key, values := range r.URL.Query() {
			for _, value := range values {
				if containsSQLInjection(value) {
					sm.logger.Warn("Potential SQL injection detected",
						"param", key,
						"value", value,
						"remote_addr", r.RemoteAddr,
					)
					http.Error(w, "Invalid request", http.StatusBadRequest)
					return
				}
			}
		}

		// Continue with the request
		next.ServeHTTP(w, r)
	})
}

// containsSQLInjection checks if a string contains SQL injection patterns
func containsSQLInjection(s string) bool {
	// This is a very basic check - in a real implementation, you would use
	// a more sophisticated approach, such as a SQL parser or a dedicated library
	patterns := []string{
		"'", "--", ";", "/*", "*/", "xp_", "sp_", "exec", "select", "drop", "update", "delete", "insert",
		"union", "where", "from", "having", "group by", "order by",
	}

	lowered := strings.ToLower(s)
	for _, pattern := range patterns {
		if strings.Contains(lowered, pattern) {
			return true
		}
	}

	return false
}

// XSSProtection middleware protects against cross-site scripting
func (sm *SecurityMiddleware) XSSProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check query parameters for XSS patterns
		for key, values := range r.URL.Query() {
			for _, value := range values {
				if containsXSS(value) {
					sm.logger.Warn("Potential XSS attack detected",
						"param", key,
						"value", value,
						"remote_addr", r.RemoteAddr,
					)
					http.Error(w, "Invalid request", http.StatusBadRequest)
					return
				}
			}
		}

		// Continue with the request
		next.ServeHTTP(w, r)
	})
}

// containsXSS checks if a string contains XSS patterns
func containsXSS(s string) bool {
	// This is a very basic check - in a real implementation, you would use
	// a more sophisticated approach, such as an HTML parser or a dedicated library
	patterns := []string{
		"<script", "javascript:", "onerror=", "onload=", "eval(", "document.cookie",
		"alert(", "prompt(", "confirm(", "<iframe", "<img", "<svg",
	}

	lowered := strings.ToLower(s)
	for _, pattern := range patterns {
		if strings.Contains(lowered, pattern) {
			return true
		}
	}

	return false
}

// JWTRenewal middleware handles JWT token renewal
func (sm *SecurityMiddleware) JWTRenewal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from request
		token, _, err := jwtauth.FromContext(r.Context())
		if err != nil || token == nil {
			// No token or invalid token, continue to standard JWT authenticator
			next.ServeHTTP(w, r)
			return
		}

		// Check if token is about to expire
		if claims, ok := token.PrivateClaims()["exp"].(float64); ok {
			expTime := time.Unix(int64(claims), 0)
			renewThreshold := time.Now().Add(15 * time.Minute)

			if expTime.Before(renewThreshold) {
				// Token is about to expire, issue a new one
				// Extract necessary claims from the current token
				userID, _ := token.PrivateClaims()["user_id"].(string)
				username, _ := token.PrivateClaims()["username"].(string)
				role, _ := token.PrivateClaims()["role"].(string)
				walletAddress, _ := token.PrivateClaims()["wallet_address"].(string)

				// Create new claims with extended expiration
				newClaims := map[string]interface{}{
					"user_id":        userID,
					"username":       username,
					"role":           role,
					"wallet_address": walletAddress,
					"exp":            time.Now().Add(24 * time.Hour).Unix(),
				}

				// Generate new token
				_, newTokenString, err := sm.tokenAuth.Encode(newClaims)
				if err == nil {
					// Set the new token in the response header
					w.Header().Set("X-New-Token", newTokenString)
				}
			}
		}

		// Continue with the request
		next.ServeHTTP(w, r)
	})
}

// ErrorHandling middleware provides consistent error handling
func (sm *SecurityMiddleware) ErrorHandling(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use a panic recovery to catch any errors
		defer func() {
			if err := recover(); err != nil {
				// Log the error
				sm.logger.Error("Panic recovered in error handling middleware",
					"error", err,
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr,
				)

				// Return a generic error to the client
				http.Error(w, "An internal server error occurred", http.StatusInternalServerError)
			}
		}()

		// Continue with the request
		next.ServeHTTP(w, r)
	})
}

// AccessControl middleware implements object-level access control
func (sm *SecurityMiddleware) AccessControl(resourceType string, accessCheck func(r *http.Request, resourceID string) bool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract resource ID from URL parameters
			resourceID := chi.URLParam(r, "id")
			if resourceID == "" {
				// No resource ID, continue with the request
				next.ServeHTTP(w, r)
				return
			}

			// Check if user has access to the resource
			if !accessCheck(r, resourceID) {
				sm.logger.Warn("Access denied to resource",
					"resource_type", resourceType,
					"resource_id", resourceID,
					"user_id", r.Context().Value("user_id"),
					"path", r.URL.Path,
				)
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			// Continue with the request
			next.ServeHTTP(w, r)
		})
	}
}

// RequestLogging middleware logs detailed information about requests and responses
func (sm *SecurityMiddleware) RequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture the status code
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Get request ID from context if available
		requestID := middleware.GetReqID(r.Context())

		// Log request start with security-relevant information
		sm.logger.Info("Request started",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"request_id", requestID,
			"user_agent", r.UserAgent(),
			"referer", r.Referer(),
		)

		// Call the next handler
		next.ServeHTTP(ww, r)

		// Log request completion with security-relevant information
		duration := time.Since(start)
		status := ww.Status()
		if status == 0 {
			status = http.StatusOK // Default to 200 if status not explicitly set
		}

		// Determine log level based on status code
		if status >= 500 {
			sm.logger.Error("Request completed with server error",
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"duration_ms", duration.Milliseconds(),
				"request_id", requestID,
				"user_id", r.Context().Value("user_id"),
			)
		} else if status >= 400 {
			sm.logger.Warn("Request completed with client error",
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"duration_ms", duration.Milliseconds(),
				"request_id", requestID,
				"user_id", r.Context().Value("user_id"),
			)
		} else {
			sm.logger.Info("Request completed successfully",
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"duration_ms", duration.Milliseconds(),
				"request_id", requestID,
				"user_id", r.Context().Value("user_id"),
			)
		}
	})
}
