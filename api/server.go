// Package api provides a HTTP API server for the Stathera system.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cmatc13/stathera/ledger"
	"github.com/cmatc13/stathera/settlement"
	"github.com/cmatc13/stathera/timeoracle"
	"github.com/cmatc13/stathera/transaction"
	"github.com/gorilla/mux"
)

// Server represents the API server
type Server struct {
	router           *mux.Router
	httpServer       *http.Server
	txEngine         *transaction.TransactionEngine
	canonicalLedger  *ledger.Ledger
	settlementEngine *settlement.SettlementEngine
	timeOracle       timeoracle.TimeOracle
}

// NewServer creates a new API server
func NewServer(
	txEngine *transaction.TransactionEngine,
	canonicalLedger *ledger.Ledger,
	settlementEngine *settlement.SettlementEngine,
	timeOracle timeoracle.TimeOracle,
	port int,
) *Server {
	s := &Server{
		router:           mux.NewRouter(),
		txEngine:         txEngine,
		canonicalLedger:  canonicalLedger,
		settlementEngine: settlementEngine,
		timeOracle:       timeOracle,
	}

	// Initialize HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      s.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Set up routes
	s.setupRoutes()

	return s
}

// setupRoutes configures the API routes
func (s *Server) setupRoutes() {
	// API version prefix
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Health check
	api.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Account endpoints
	api.HandleFunc("/accounts", s.handleCreateAccount).Methods("POST")
	api.HandleFunc("/accounts/{address}", s.handleGetAccount).Methods("GET")
	api.HandleFunc("/accounts/{address}/balance", s.handleGetBalance).Methods("GET")

	// Transaction endpoints
	api.HandleFunc("/transactions", s.handleSubmitTransaction).Methods("POST")
	api.HandleFunc("/transactions/{id}", s.handleGetTransaction).Methods("GET")
	api.HandleFunc("/transactions", s.handleListTransactions).Methods("GET")

	// System endpoints
	api.HandleFunc("/system/supply", s.handleGetSupply).Methods("GET")
	api.HandleFunc("/system/inflation", s.handleGetInflation).Methods("GET")
	api.HandleFunc("/system/time", s.handleGetTime).Methods("GET")
}

// Start starts the API server
func (s *Server) Start() error {
	log.Printf("API server starting on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Stop stops the API server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("API server shutting down")
	return s.httpServer.Shutdown(ctx)
}

// handleHealth handles the health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	}
	respondWithJSON(w, http.StatusOK, response)
}

// handleCreateAccount handles account creation
func (s *Server) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Address string `json:"address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Generate a dummy public key for now
	// In a real implementation, the client would generate the key pair
	// and send the public key
	pubKey := make([]byte, 32)

	if err := s.txEngine.CreateAccount(req.Address, pubKey); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]string{
		"address": req.Address,
		"status":  "created",
	}
	respondWithJSON(w, http.StatusCreated, response)
}

// handleGetAccount handles getting account details
func (s *Server) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	account, err := s.txEngine.GetAccount(address)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Account not found")
		return
	}

	respondWithJSON(w, http.StatusOK, account)
}

// handleGetBalance handles getting account balance
func (s *Server) handleGetBalance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	balance, err := s.txEngine.GetBalance(address)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Account not found")
		return
	}

	response := map[string]interface{}{
		"address": address,
		"balance": balance,
	}
	respondWithJSON(w, http.StatusOK, response)
}

// handleSubmitTransaction handles transaction submission
func (s *Server) handleSubmitTransaction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Sender      string  `json:"sender"`
		Receiver    string  `json:"receiver"`
		Amount      float64 `json:"amount"`
		Fee         float64 `json:"fee"`
		Type        string  `json:"type"`
		Nonce       string  `json:"nonce"`
		Description string  `json:"description"`
		Signature   []byte  `json:"signature"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Create transaction
	tx, err := transaction.NewTransaction(
		req.Sender,
		req.Receiver,
		req.Amount,
		req.Fee,
		transaction.TransactionType(req.Type),
		req.Nonce,
		req.Description,
	)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Set signature (in a real implementation, this would be verified)
	tx.Signature = req.Signature

	// Process transaction
	if err := s.txEngine.ProcessTransaction(tx); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]string{
		"id":     tx.ID,
		"status": string(tx.Status),
	}
	respondWithJSON(w, http.StatusCreated, response)
}

// handleGetTransaction handles getting transaction details
func (s *Server) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	tx, err := s.txEngine.GetTransaction(id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Transaction not found")
		return
	}

	respondWithJSON(w, http.StatusOK, tx)
}

// handleListTransactions handles listing transactions
func (s *Server) handleListTransactions(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	address := r.URL.Query().Get("address")

	var transactions []*transaction.Transaction

	if address != "" {
		// If address is provided, get transactions for that address
		// This is a placeholder - the actual implementation would depend on your storage layer
		respondWithError(w, http.StatusNotImplemented, "Address filtering not implemented")
		return
	} else {
		// Otherwise, get all transactions
		transactions = s.txEngine.GetTransactions()
	}

	respondWithJSON(w, http.StatusOK, transactions)
}

// handleGetSupply handles getting the total supply
func (s *Server) handleGetSupply(w http.ResponseWriter, r *http.Request) {
	supply := s.canonicalLedger.GetTotalSupply()

	response := map[string]interface{}{
		"total_supply": supply,
	}
	respondWithJSON(w, http.StatusOK, response)
}

// handleGetInflation handles getting the inflation rate
// This is a placeholder - the actual implementation would depend on your system
func (s *Server) handleGetInflation(w http.ResponseWriter, r *http.Request) {
	// Placeholder - in a real implementation, you would get this from the ledger
	response := map[string]interface{}{
		"min_inflation":     1.5,
		"max_inflation":     3.0,
		"current_inflation": 2.0,
	}
	respondWithJSON(w, http.StatusOK, response)
}

// handleGetTime handles getting the current time with proof
func (s *Server) handleGetTime(w http.ResponseWriter, r *http.Request) {
	timestamp, proof, err := s.timeOracle.GetTimeWithProof()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"timestamp": timestamp,
		"proof":     proof,
	}
	respondWithJSON(w, http.StatusOK, response)
}

// respondWithError returns an error response
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// respondWithJSON returns a JSON response
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
