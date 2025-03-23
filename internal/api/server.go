// internal/api/server.go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/jwtauth/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/cmatc13/stathera/internal/orderbook"
	"github.com/cmatc13/stathera/internal/security"
	"github.com/cmatc13/stathera/internal/transaction"
	"github.com/cmatc13/stathera/internal/wallet"
	"github.com/cmatc13/stathera/pkg/config"
	"github.com/cmatc13/stathera/pkg/health"
	"github.com/cmatc13/stathera/pkg/logging"
	"github.com/cmatc13/stathera/pkg/metrics"
	txproc "github.com/cmatc13/stathera/pkg/transaction"
)

// Server represents the API server
type Server struct {
	config           *config.Config
	router           *chi.Mux
	txProcessor      txproc.Processor
	orderbook        *orderbook.RedisOrderBook
	tokenAuth        *jwtauth.JWTAuth
	server           *http.Server
	logger           *logging.Logger
	metricsCollector *metrics.Metrics
	healthRegistry   *health.Registry
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, txProcessor txproc.Processor, orderbook *orderbook.RedisOrderBook) *Server {
	r := chi.NewRouter()
	tokenAuth := jwtauth.New("HS256", []byte(cfg.Auth.JWTSecret), nil)

	// Set up structured logger
	logCfg := logging.Config{
		Level:       logging.LogLevel(cfg.Log.Level),
		Output:      log.Writer(),
		ServiceName: "api",
		Environment: cfg.Log.Environment,
	}
	logger := logging.New(logCfg)

	// Set up metrics
	metricsCfg := metrics.Config{
		Namespace:   cfg.Metrics.Namespace,
		Subsystem:   "api",
		ServiceName: "api",
	}
	metricsCollector := metrics.New(metricsCfg)

	// Set up health registry
	healthRegistry := health.NewRegistry(logger)

	s := &Server{
		config:           cfg,
		router:           r,
		txProcessor:      txProcessor,
		orderbook:        orderbook,
		tokenAuth:        tokenAuth,
		logger:           logger,
		metricsCollector: metricsCollector,
		healthRegistry:   healthRegistry,
		server: &http.Server{
			Addr:    ":" + cfg.API.Port,
			Handler: r,
		},
	}

	// Set up middleware and routes
	s.setupMiddleware()
	s.setupRoutes()
	s.setupHealthChecks()

	return s
}

// setupMiddleware configures middleware for the server
func (s *Server) setupMiddleware() {
	// Initialize security middleware
	securityManager, err := security.NewSecurityManager(s.config.Redis.Address, s.config.Auth.JWTSecret)
	if err != nil {
		s.logger.Error("Failed to initialize security manager", "error", err)
		return
	}

	securityMiddleware := NewSecurityMiddleware(securityManager, s.tokenAuth, s.logger)

	// Basic middleware
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)

	// Security middleware
	s.router.Use(securityMiddleware.SecureHeaders)
	s.router.Use(securityMiddleware.ContentSecurityPolicy)
	s.router.Use(securityMiddleware.ErrorHandling)
	s.router.Use(securityMiddleware.XSSProtection)
	s.router.Use(securityMiddleware.SQLInjectionProtection)

	// Custom structured logging middleware with security enhancements
	s.router.Use(securityMiddleware.RequestLogging)

	// Custom metrics middleware
	s.router.Use(MetricsMiddleware(s.metricsCollector, "api"))

	// Custom recoverer with metrics
	s.router.Use(RecovererWithMetrics(s.logger, s.metricsCollector, "api"))

	// Add CORS middleware with stricter settings
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   s.config.API.CORSAllowedOrigins, // Use configuration instead of wildcard
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-API-Key"},
		ExposedHeaders:   []string{"Link", "X-New-Token"}, // Expose token renewal header
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Add advanced rate limiting middleware (per user/IP and path)
	s.router.Use(securityMiddleware.RateLimiter(100, 1*time.Minute))
}

// setupRoutes configures the API routes
func (s *Server) setupRoutes() {
	// Initialize security middleware
	securityManager, err := security.NewSecurityManager(s.config.Redis.Address, s.config.Auth.JWTSecret)
	if err != nil {
		s.logger.Error("Failed to initialize security manager", "error", err)
		return
	}

	securityMiddleware := NewSecurityMiddleware(securityManager, s.tokenAuth, s.logger)

	// Public routes
	s.router.Group(func(r chi.Router) {
		// Apply input validation and sanitization
		r.Use(securityMiddleware.InputSanitization)
		r.Use(securityMiddleware.RequestValidation(func(r *http.Request) error {
			// Basic validation - in a real implementation, you would have more specific validation
			return nil
		}))

		r.Get("/health", s.handleHealth)
		r.Get("/metrics", promhttp.Handler().ServeHTTP)

		// Apply content type validation for endpoints that accept JSON
		r.With(securityMiddleware.ValidateContentType("application/json")).Post("/register", s.handleRegister)
		r.With(securityMiddleware.ValidateContentType("application/json")).Post("/login", s.handleLogin)
	})

	// Protected routes - require authentication (JWT or API key)
	s.router.Group(func(r chi.Router) {
		// Authentication middleware - try API key first, then JWT
		r.Use(securityMiddleware.APIKeyAuth)
		r.Use(jwtauth.Verifier(s.tokenAuth))
		r.Use(securityMiddleware.JWTWithBruteForceProtection)
		r.Use(jwtauth.Authenticator)

		// Add CSRF protection for state-changing operations
		r.Use(securityMiddleware.CSRFProtection)

		// Apply input validation and sanitization
		r.Use(securityMiddleware.InputSanitization)
		r.Use(securityMiddleware.RequestValidation(func(r *http.Request) error {
			// Basic validation - in a real implementation, you would have more specific validation
			return nil
		}))

		// Apply content type validation for endpoints that accept JSON
		r.Use(securityMiddleware.ValidateContentType("application/json"))

		// Apply response sanitization
		r.Use(securityMiddleware.ResponseSanitization)

		// User routes
		r.Get("/balance", s.handleGetBalance)
		r.Get("/transactions", s.handleGetTransactions)

		// Transaction routes
		r.Post("/transfer", s.handleTransfer)

		// Wallet routes
		r.Get("/wallet", s.handleGetWalletInfo)

		// Order book routes
		r.Get("/orderbook", s.handleGetOrderBook)
		r.Post("/orders", s.handlePlaceOrder)
		r.Delete("/orders/{id}", s.handleCancelOrder)
	})

	// Admin routes - require admin role
	s.router.Group(func(r chi.Router) {
		// Authentication middleware with enhanced security
		r.Use(securityMiddleware.APIKeyAuth)
		r.Use(jwtauth.Verifier(s.tokenAuth))
		r.Use(securityMiddleware.JWTWithBruteForceProtection)
		r.Use(jwtauth.Authenticator)
		r.Use(s.adminOnly)

		// Add CSRF protection for state-changing operations
		r.Use(securityMiddleware.CSRFProtection)

		// Apply input validation and sanitization
		r.Use(securityMiddleware.InputSanitization)
		r.Use(securityMiddleware.RequestValidation(func(r *http.Request) error {
			// Basic validation - in a real implementation, you would have more specific validation
			return nil
		}))

		// Apply content type validation for endpoints that accept JSON
		r.Use(securityMiddleware.ValidateContentType("application/json"))

		// Apply response sanitization
		r.Use(securityMiddleware.ResponseSanitization)

		// Apply object-level access control
		r.Use(securityMiddleware.AccessControl("admin", func(r *http.Request, resourceID string) bool {
			// In a real implementation, you would check if the user has access to the resource
			// For now, we'll just check if the user has the admin role
			_, claims, _ := jwtauth.FromContext(r.Context())
			role, _ := claims["role"].(string)
			return role == "admin"
		}))

		r.Get("/admin/system/supply", s.handleGetTotalSupply)
		r.Get("/admin/system/inflation", s.handleGetInflationRate)
		r.Post("/admin/system/adjust-inflation", s.handleAdjustInflation)
	})
}

// setupHealthChecks configures health checks for the server
func (s *Server) setupHealthChecks() {
	// Register API server health check
	s.healthRegistry.Register("api", health.ServiceChecker("api", func(ctx context.Context) error {
		return nil // API server is healthy if this code is running
	}))

	// Register Redis health check
	s.healthRegistry.Register("redis", health.RedisChecker(s.config.Redis.Address, func(ctx context.Context) error {
		// This is a placeholder - in a real implementation, you would ping Redis
		// For now, we'll just check if the Redis address is valid
		return nil
	}))

	// Register transaction processor health check
	s.healthRegistry.Register("transaction-processor", health.DependencyChecker("transaction-processor", func(ctx context.Context) error {
		// This is a placeholder - in a real implementation, you would check the transaction processor
		return nil
	}))

	// Register orderbook health check
	s.healthRegistry.Register("orderbook", health.DependencyChecker("orderbook", func(ctx context.Context) error {
		// This is a placeholder - in a real implementation, you would check the orderbook
		return nil
	}))
}

// Start starts the API server
func (s *Server) Start() {
	s.logger.Info("Starting API server", "port", s.config.API.Port)

	// Record the start time for metrics
	s.metricsCollector.ServiceLastStarted.Set(float64(time.Now().Unix()))

	// Start recording uptime
	uptimeDone := make(chan struct{})
	s.metricsCollector.RecordUptime(uptimeDone)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error("Error starting server", "error", err)
		close(uptimeDone)
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) {
	s.logger.Info("Shutting down API server")
	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error("Error during server shutdown", "error", err)
	}
	s.logger.Info("API server shutdown complete")
}

// Response represents a standardized API response
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Run all health checks
	checks := s.healthRegistry.RunChecks(r.Context())

	// Determine overall status
	status := health.StatusUp
	for _, check := range checks {
		if check.Status == health.StatusDown {
			status = health.StatusDown
			break
		} else if check.Status == health.StatusUnknown && status != health.StatusDown {
			status = health.StatusUnknown
		}
	}

	// Set HTTP status code based on health status
	httpStatus := http.StatusOK
	if status == health.StatusDown {
		httpStatus = http.StatusServiceUnavailable
	}

	// Build response
	resp := Response{
		Success: status == health.StatusUp,
		Message: "Service health status: " + string(status),
		Data: map[string]interface{}{
			"status":    status,
			"timestamp": time.Now().Unix(),
			"version":   s.config.API.Version,
			"checks":    checks,
			"system": map[string]interface{}{
				"go_version":    runtime.Version(),
				"go_goroutines": runtime.NumGoroutine(),
				"go_cpus":       runtime.NumCPU(),
			},
		},
	}

	s.renderJSON(w, resp, httpStatus)
}

// handleRegister handles user registration requests
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.renderError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		s.renderError(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Create a new wallet for the user
	newWallet, err := wallet.NewWallet()
	if err != nil {
		s.renderError(w, "Failed to create wallet", http.StatusInternalServerError)
		return
	}

	// In a real implementation, you would:
	// 1. Check if username/email already exists
	// 2. Hash the password
	// 3. Store user data in a database
	// 4. Assign the wallet to the user

	// For this implementation, we'll just return the wallet details
	resp := Response{
		Success: true,
		Message: "User registered successfully",
		Data: map[string]interface{}{
			"username":       req.Username,
			"wallet_address": newWallet.Address,
			// Note: In a real app, you would NOT return the private key here
			// This is just for demonstration
			"private_key": newWallet.ExportPrivateKey(),
		},
	}

	s.renderJSON(w, resp, http.StatusCreated)
}

// handleLogin handles user login requests
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.renderError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// In a real implementation, you would:
	// 1. Retrieve user from database
	// 2. Verify password
	// 3. Check account status

	// For this implementation, we'll assume authentication is successful
	// and generate a JWT token

	// Create claims with user information
	claims := map[string]interface{}{
		"user_id":        "12345", // Example user ID
		"username":       req.Username,
		"role":           "user",
		"wallet_address": "example_wallet_address",
		"exp":            time.Now().Add(time.Hour * 24).Unix(), // 24-hour expiration
	}

	// Generate JWT token
	_, tokenString, err := s.tokenAuth.Encode(claims)
	if err != nil {
		s.renderError(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	resp := Response{
		Success: true,
		Message: "Login successful",
		Data: map[string]interface{}{
			"token":      tokenString,
			"expires_at": time.Now().Add(time.Hour * 24).Unix(),
		},
	}

	s.renderJSON(w, resp, http.StatusOK)
}

// handleGetBalance handles balance check requests
func (s *Server) handleGetBalance(w http.ResponseWriter, r *http.Request) {
	// Get user from JWT token
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil {
		s.renderError(w, "Authentication error", http.StatusUnauthorized)
		return
	}

	walletAddress, ok := claims["wallet_address"].(string)
	if !ok {
		s.renderError(w, "Invalid token claims", http.StatusBadRequest)
		return
	}

	// Get balance from Redis
	// This assumes the txProcessor interface has a GetBalance method
	// If it doesn't, you'll need to modify this code
	balance, err := s.txProcessor.(interface{ GetBalance(string) (float64, error) }).GetBalance(walletAddress)
	if err != nil {
		s.renderError(w, "Failed to retrieve balance", http.StatusInternalServerError)
		return
	}

	resp := Response{
		Success: true,
		Data: map[string]interface{}{
			"address": walletAddress,
			"balance": balance,
		},
	}

	s.renderJSON(w, resp, http.StatusOK)
}

// handleGetTransactions handles transaction history requests
func (s *Server) handleGetTransactions(w http.ResponseWriter, r *http.Request) {
	// Get user from JWT token
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil {
		s.renderError(w, "Authentication error", http.StatusUnauthorized)
		return
	}

	walletAddress, ok := claims["wallet_address"].(string)
	if !ok {
		s.renderError(w, "Invalid token claims", http.StatusBadRequest)
		return
	}

	// Get pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := int64(10) // Default
	offset := int64(0) // Default

	if limitStr != "" {
		if l, err := strconv.ParseInt(limitStr, 10, 64); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.ParseInt(offsetStr, 10, 64); err == nil && o >= 0 {
			offset = o
		}
	}

	// Get transactions from Redis
	// This assumes the txProcessor interface has a GetUserTransactions method
	// If it doesn't, you'll need to modify this code
	transactions, err := s.txProcessor.(interface {
		GetUserTransactions(string, int64, int64) ([]*transaction.Transaction, error)
	}).GetUserTransactions(walletAddress, limit, offset)
	if err != nil {
		s.renderError(w, "Failed to retrieve transactions", http.StatusInternalServerError)
		return
	}

	resp := Response{
		Success: true,
		Data: map[string]interface{}{
			"transactions": transactions,
			"pagination": map[string]interface{}{
				"limit":  limit,
				"offset": offset,
				"total":  len(transactions), // In a real implementation, you'd get the total count
			},
		},
	}

	s.renderJSON(w, resp, http.StatusOK)
}

// handleTransfer handles money transfer requests
func (s *Server) handleTransfer(w http.ResponseWriter, r *http.Request) {
	// Get user from JWT token
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil {
		s.renderError(w, "Authentication error", http.StatusUnauthorized)
		return
	}

	senderAddress, ok := claims["wallet_address"].(string)
	if !ok {
		s.renderError(w, "Invalid token claims", http.StatusBadRequest)
		return
	}

	// Parse request
	var req struct {
		ReceiverAddress string  `json:"receiver_address"`
		Amount          float64 `json:"amount"`
		Description     string  `json:"description"`
		PrivateKey      string  `json:"private_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.renderError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.ReceiverAddress == "" || req.Amount <= 0 {
		s.renderError(w, "Invalid receiver address or amount", http.StatusBadRequest)
		return
	}

	// In a real implementation, the private key would not be sent in the request
	// Instead, the user would sign the transaction client-side
	// This is just for demonstration purposes

	// Import wallet from private key
	userWallet, err := wallet.ImportWallet(req.PrivateKey)
	if err != nil {
		s.renderError(w, "Invalid private key", http.StatusBadRequest)
		return
	}

	// Verify the wallet address matches the authenticated user
	if userWallet.Address != senderAddress {
		s.renderError(w, "Private key does not match authenticated user", http.StatusUnauthorized)
		return
	}

	// Generate nonce for transaction
	nonce, err := wallet.GenerateNonce()
	if err != nil {
		s.renderError(w, "Failed to generate nonce", http.StatusInternalServerError)
		return
	}

	// Create transaction
	// Calculate fee (0.1% of the amount, minimum 0.01)
	fee := req.Amount * 0.001
	if fee < 0.01 {
		fee = 0.01
	}

	tx, err := transaction.NewTransaction(
		senderAddress,
		req.ReceiverAddress,
		req.Amount,
		fee,
		transaction.Payment,
		nonce,
		req.Description,
	)
	if err != nil {
		s.renderError(w, "Failed to create transaction", http.StatusBadRequest)
		return
	}

	// Sign transaction
	signData, err := tx.SignableData()
	if err != nil {
		s.renderError(w, "Failed to generate signable data", http.StatusInternalServerError)
		return
	}

	tx.Signature, err = userWallet.SignMessage(signData)
	if err != nil {
		s.renderError(w, "Failed to sign transaction", http.StatusInternalServerError)
		return
	}

	// Submit transaction to processor
	err = s.txProcessor.SubmitTransaction(tx)
	if err != nil {
		s.renderError(w, "Failed to submit transaction", http.StatusInternalServerError)
		return
	}

	resp := Response{
		Success: true,
		Message: "Transaction submitted successfully",
		Data: map[string]interface{}{
			"transaction_id": tx.ID,
			"amount":         tx.Amount,
			"fee":            tx.Fee,
			"timestamp":      tx.Timestamp,
		},
	}

	s.renderJSON(w, resp, http.StatusOK)
}

// handleGetWalletInfo handles wallet info requests
func (s *Server) handleGetWalletInfo(w http.ResponseWriter, r *http.Request) {
	// Get user from JWT token
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil {
		s.renderError(w, "Authentication error", http.StatusUnauthorized)
		return
	}

	walletAddress, ok := claims["wallet_address"].(string)
	if !ok {
		s.renderError(w, "Invalid token claims", http.StatusBadRequest)
		return
	}

	// In a real implementation, you would retrieve detailed wallet info
	// For this implementation, we'll just return the address

	resp := Response{
		Success: true,
		Data: map[string]interface{}{
			"address": walletAddress,
		},
	}

	s.renderJSON(w, resp, http.StatusOK)
}

// handleGetOrderBook handles order book requests
func (s *Server) handleGetOrderBook(w http.ResponseWriter, r *http.Request) {
	// Get depth parameter
	depthStr := r.URL.Query().Get("depth")
	depth := int64(10) // Default

	if depthStr != "" {
		if d, err := strconv.ParseInt(depthStr, 10, 64); err == nil && d > 0 {
			depth = d
		}
	}

	// Get order book from Redis
	orderBookData, err := s.orderbook.GetOrderBook(depth)
	if err != nil {
		s.renderError(w, "Failed to retrieve order book", http.StatusInternalServerError)
		return
	}

	resp := Response{
		Success: true,
		Data:    orderBookData,
	}

	s.renderJSON(w, resp, http.StatusOK)
}

// handlePlaceOrder handles order placement requests
func (s *Server) handlePlaceOrder(w http.ResponseWriter, r *http.Request) {
	// Get user from JWT token
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil {
		s.renderError(w, "Authentication error", http.StatusUnauthorized)
		return
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		s.renderError(w, "Invalid token claims", http.StatusBadRequest)
		return
	}

	// Parse request
	var req struct {
		Type   string  `json:"type"`
		Price  float64 `json:"price"`
		Amount float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.renderError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Price <= 0 || req.Amount <= 0 {
		s.renderError(w, "Price and amount must be positive", http.StatusBadRequest)
		return
	}

	// Determine order type
	var orderType orderbook.OrderType
	if req.Type == "buy" {
		orderType = orderbook.BidOrder
	} else if req.Type == "sell" {
		orderType = orderbook.AskOrder
	} else {
		s.renderError(w, "Invalid order type", http.StatusBadRequest)
		return
	}

	// Create order
	order := orderbook.NewOrder(userID, orderType, req.Price, req.Amount)

	// Place order
	err = s.orderbook.PlaceOrder(order)
	if err != nil {
		s.renderError(w, "Failed to place order", http.StatusInternalServerError)
		return
	}

	resp := Response{
		Success: true,
		Message: "Order placed successfully",
		Data: map[string]interface{}{
			"order_id":  order.ID,
			"type":      order.Type,
			"price":     order.Price,
			"amount":    order.Amount,
			"timestamp": order.CreatedAt,
		},
	}

	s.renderJSON(w, resp, http.StatusOK)
}

// handleCancelOrder handles order cancellation requests
func (s *Server) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	// Get user from JWT token
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil {
		s.renderError(w, "Authentication error", http.StatusUnauthorized)
		return
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		s.renderError(w, "Invalid token claims", http.StatusBadRequest)
		return
	}

	// Get order ID from URL
	orderID := chi.URLParam(r, "id")
	if orderID == "" {
		s.renderError(w, "Order ID is required", http.StatusBadRequest)
		return
	}

	// Cancel order
	err = s.orderbook.CancelOrder(orderID, userID)
	if err != nil {
		s.renderError(w, fmt.Sprintf("Failed to cancel order: %v", err), http.StatusInternalServerError)
		return
	}

	resp := Response{
		Success: true,
		Message: "Order cancelled successfully",
	}

	s.renderJSON(w, resp, http.StatusOK)
}

// handleGetTotalSupply handles total supply requests (admin only)
func (s *Server) handleGetTotalSupply(w http.ResponseWriter, r *http.Request) {
	// Get total supply from Redis
	// This assumes the txProcessor interface has a GetTotalSupply method
	// If it doesn't, you'll need to modify this code
	totalSupply, err := s.txProcessor.(interface{ GetTotalSupply() (float64, error) }).GetTotalSupply()
	if err != nil {
		s.renderError(w, "Failed to retrieve total supply", http.StatusInternalServerError)
		return
	}

	resp := Response{
		Success: true,
		Data: map[string]interface{}{
			"total_supply": totalSupply,
			"timestamp":    time.Now().Unix(),
		},
	}

	s.renderJSON(w, resp, http.StatusOK)
}

// handleGetInflationRate handles inflation rate requests (admin only)
func (s *Server) handleGetInflationRate(w http.ResponseWriter, r *http.Request) {
	// Get inflation rate from Redis
	// This assumes the txProcessor interface has a GetInflationRate method
	// If it doesn't, you'll need to modify this code
	inflationRate, err := s.txProcessor.(interface{ GetInflationRate() (float64, error) }).GetInflationRate()
	if err != nil {
		s.renderError(w, "Failed to retrieve inflation rate", http.StatusInternalServerError)
		return
	}

	resp := Response{
		Success: true,
		Data: map[string]interface{}{
			"inflation_rate": inflationRate,
			"timestamp":      time.Now().Unix(),
		},
	}

	s.renderJSON(w, resp, http.StatusOK)
}

// handleAdjustInflation handles inflation rate adjustment requests (admin only)
func (s *Server) handleAdjustInflation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MinRate float64 `json:"min_rate"`
		MaxRate float64 `json:"max_rate"`
		MaxStep float64 `json:"max_step"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.renderError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.MinRate < 0 || req.MaxRate <= req.MinRate || req.MaxStep <= 0 {
		s.renderError(w, "Invalid inflation parameters", http.StatusBadRequest)
		return
	}

	// In a real implementation, you would update the inflation rate
	// For this implementation, we'll just return a success response

	resp := Response{
		Success: true,
		Message: "Inflation rate updated successfully",
		Data: map[string]interface{}{
			"min_rate":  req.MinRate,
			"max_rate":  req.MaxRate,
			"timestamp": time.Now().Unix(),
		},
	}

	s.renderJSON(w, resp, http.StatusOK)
}

// adminOnly is middleware to verify the user has admin role
func (s *Server) adminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, claims, err := jwtauth.FromContext(r.Context())
		if err != nil {
			s.renderError(w, "Authentication error", http.StatusUnauthorized)
			return
		}

		role, ok := claims["role"].(string)
		if !ok || role != "admin" {
			s.renderError(w, "Admin access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// renderJSON renders a JSON response
func (s *Server) renderJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("Error encoding JSON response", "error", err)
	}
}

// renderError renders an error response
func (s *Server) renderError(w http.ResponseWriter, message string, status int) {
	// Record error metric
	s.metricsCollector.RecordError("api", "http", strconv.Itoa(status))

	resp := Response{
		Success: false,
		Error:   message,
	}

	s.renderJSON(w, resp, status)
}
