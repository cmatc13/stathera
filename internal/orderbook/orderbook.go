// internal/orderbook/orderbook.go
package orderbook

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// OrderType represents the type of order
type OrderType string

const (
	// BidOrder is an order to buy the coin
	BidOrder OrderType = "BID"
	// AskOrder is an order to sell the coin
	AskOrder OrderType = "ASK"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	// Open orders are active and available for matching
	Open OrderStatus = "OPEN"
	// Filled orders have been fully matched
	Filled OrderStatus = "FILLED"
	// PartiallyFilled orders have been partially matched
	PartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	// Cancelled orders have been cancelled by the user
	Cancelled OrderStatus = "CANCELLED"
)

// Order represents a market order
type Order struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id"`
	Type      OrderType   `json:"type"`
	Price     float64     `json:"price"`
	Amount    float64     `json:"amount"`
	Filled    float64     `json:"filled"`
	Status    OrderStatus `json:"status"`
	CreatedAt int64       `json:"created_at"`
	UpdatedAt int64       `json:"updated_at"`
}

// NewOrder creates a new order
func NewOrder(userID string, orderType OrderType, price, amount float64) *Order {
	now := time.Now().Unix()

	return &Order{
		ID:        uuid.New().String(),
		UserID:    userID,
		Type:      orderType,
		Price:     price,
		Amount:    amount,
		Filled:    0,
		Status:    Open,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Match represents a matched order
type Match struct {
	ID         string  `json:"id"`
	BidOrderID string  `json:"bid_order_id"`
	AskOrderID string  `json:"ask_order_id"`
	Price      float64 `json:"price"`
	Amount     float64 `json:"amount"`
	BidUserID  string  `json:"bid_user_id"`
	AskUserID  string  `json:"ask_user_id"`
	ExecutedAt int64   `json:"executed_at"`
}

// RedisOrderBook implements an order book using Redis
type RedisOrderBook struct {
	client *redis.Client
	ctx    context.Context
	mu     sync.Mutex // For thread safety
}

const (
	// Redis key prefixes
	bidOrdersKey     = "orderbook:bids"          // Sorted set of bid orders
	askOrdersKey     = "orderbook:asks"          // Sorted set of ask orders
	orderPrefix      = "order:"                  // Prefix for individual orders
	matchPrefix      = "match:"                  // Prefix for matches
	userOrdersPrefix = "user:orders:"            // Prefix for user orders
	recentTradesKey  = "orderbook:recent_trades" // Sorted set of recent trades
)

// NewRedisOrderBook creates a new Redis-backed order book
func NewRedisOrderBook(redisAddr string) (*RedisOrderBook, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0,
	})

	ctx := context.Background()

	// Test connection
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisOrderBook{
		client: client,
		ctx:    ctx,
	}, nil
}

// Close closes the Redis connection
func (rob *RedisOrderBook) Close() error {
	return rob.client.Close()
}

// PlaceOrder adds a new order to the order book
func (rob *RedisOrderBook) PlaceOrder(order *Order) error {
	rob.mu.Lock()
	defer rob.mu.Unlock()

	// Store the order
	orderJSON, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	// Store the order details
	err = rob.client.Set(rob.ctx, orderPrefix+order.ID, orderJSON, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to store order: %w", err)
	}

	// Add to the appropriate sorted set
	var key string
	var score float64

	if order.Type == BidOrder {
		key = bidOrdersKey
		score = -order.Price // Negative for descending order (highest bids first)
	} else {
		key = askOrdersKey
		score = order.Price // Ascending order (lowest asks first)
	}

	err = rob.client.ZAdd(rob.ctx, key, &redis.Z{
		Score:  score,
		Member: order.ID,
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to add order to sorted set: %w", err)
	}

	// Add to user's orders
	err = rob.client.ZAdd(rob.ctx, userOrdersPrefix+order.UserID, &redis.Z{
		Score:  float64(order.CreatedAt),
		Member: order.ID,
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to add order to user's orders: %w", err)
	}

	// Try to match the order
	matches, err := rob.matchOrder(order)
	if err != nil {
		return fmt.Errorf("failed to match order: %w", err)
	}

	// Process matches
	for _, match := range matches {
		err = rob.processMatch(match)
		if err != nil {
			return fmt.Errorf("failed to process match: %w", err)
		}
	}

	return nil
}

// matchOrder attempts to match an order with existing orders
func (rob *RedisOrderBook) matchOrder(order *Order) ([]*Match, error) {
	var matches []*Match

	// Determine which set to search based on order type
	// The counterparty key will be used to find matching orders

	// Get potential matching orders
	var counterpartyOrders []string
	var err error

	if order.Type == BidOrder {
		// For bid orders, get asks with price <= bid price (lowest first)
		counterpartyOrders, err = rob.client.ZRangeByScore(rob.ctx, askOrdersKey, &redis.ZRangeBy{
			Min:   "0",
			Max:   fmt.Sprintf("%f", order.Price),
			Count: 100,
		}).Result()
	} else {
		// For ask orders, get bids with price >= ask price (highest first)
		counterpartyOrders, err = rob.client.ZRangeByScore(rob.ctx, bidOrdersKey, &redis.ZRangeBy{
			Min:   fmt.Sprintf("%f", -order.Price),
			Max:   "0",
			Count: 100,
		}).Result()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get counterparty orders: %w", err)
	}

	// Process potential matches
	remainingAmount := order.Amount

	for _, counterpartyID := range counterpartyOrders {
		if remainingAmount <= 0 {
			break
		}

		// Get counterparty order details
		counterpartyJSON, err := rob.client.Get(rob.ctx, orderPrefix+counterpartyID).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get counterparty order: %w", err)
		}

		var counterparty Order
		if err := json.Unmarshal([]byte(counterpartyJSON), &counterparty); err != nil {
			return nil, fmt.Errorf("failed to unmarshal counterparty order: %w", err)
		}

		// Skip filled or cancelled orders
		if counterparty.Status == Filled || counterparty.Status == Cancelled {
			continue
		}

		// Calculate match amount
		availableAmount := counterparty.Amount - counterparty.Filled
		matchAmount := min(remainingAmount, availableAmount)

		if matchAmount <= 0 {
			continue
		}

		// Determine match price (typically the older order's price)
		matchPrice := counterparty.Price

		// Create match record
		var match *Match
		if order.Type == BidOrder {
			match = &Match{
				ID:         uuid.New().String(),
				BidOrderID: order.ID,
				AskOrderID: counterparty.ID,
				Price:      matchPrice,
				Amount:     matchAmount,
				BidUserID:  order.UserID,
				AskUserID:  counterparty.UserID,
				ExecutedAt: time.Now().Unix(),
			}
		} else {
			match = &Match{
				ID:         uuid.New().String(),
				BidOrderID: counterparty.ID,
				AskOrderID: order.ID,
				Price:      matchPrice,
				Amount:     matchAmount,
				BidUserID:  counterparty.UserID,
				AskUserID:  order.UserID,
				ExecutedAt: time.Now().Unix(),
			}
		}

		matches = append(matches, match)

		// Update remainingAmount
		remainingAmount -= matchAmount
	}

	return matches, nil
}

// processMatch processes a match between two orders
func (rob *RedisOrderBook) processMatch(match *Match) error {
	// Get bid order
	bidOrderJSON, err := rob.client.Get(rob.ctx, orderPrefix+match.BidOrderID).Result()
	if err != nil {
		return fmt.Errorf("failed to get bid order: %w", err)
	}

	var bidOrder Order
	if err := json.Unmarshal([]byte(bidOrderJSON), &bidOrder); err != nil {
		return fmt.Errorf("failed to unmarshal bid order: %w", err)
	}

	// Get ask order
	askOrderJSON, err := rob.client.Get(rob.ctx, orderPrefix+match.AskOrderID).Result()
	if err != nil {
		return fmt.Errorf("failed to get ask order: %w", err)
	}

	var askOrder Order
	if err := json.Unmarshal([]byte(askOrderJSON), &askOrder); err != nil {
		return fmt.Errorf("failed to unmarshal ask order: %w", err)
	}

	// Update orders with match
	bidOrder.Filled += match.Amount
	bidOrder.UpdatedAt = time.Now().Unix()

	askOrder.Filled += match.Amount
	askOrder.UpdatedAt = time.Now().Unix()

	// Update order status
	if bidOrder.Filled >= bidOrder.Amount {
		bidOrder.Status = Filled
	} else {
		bidOrder.Status = PartiallyFilled
	}

	if askOrder.Filled >= askOrder.Amount {
		askOrder.Status = Filled
	} else {
		askOrder.Status = PartiallyFilled
	}

	// Store updated orders
	updatedBidOrderJSON, err := json.Marshal(bidOrder)
	if err != nil {
		return fmt.Errorf("failed to marshal updated bid order: %w", err)
	}

	err = rob.client.Set(rob.ctx, orderPrefix+bidOrder.ID, updatedBidOrderJSON, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to store updated bid order: %w", err)
	}

	updatedAskOrderJSON, err := json.Marshal(askOrder)
	if err != nil {
		return fmt.Errorf("failed to marshal updated ask order: %w", err)
	}

	err = rob.client.Set(rob.ctx, orderPrefix+askOrder.ID, updatedAskOrderJSON, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to store updated ask order: %w", err)
	}

	// Remove filled orders from order book
	if bidOrder.Status == Filled {
		err = rob.client.ZRem(rob.ctx, bidOrdersKey, bidOrder.ID).Err()
		if err != nil {
			return fmt.Errorf("failed to remove filled bid order: %w", err)
		}
	}

	if askOrder.Status == Filled {
		err = rob.client.ZRem(rob.ctx, askOrdersKey, askOrder.ID).Err()
		if err != nil {
			return fmt.Errorf("failed to remove filled ask order: %w", err)
		}
	}

	// Store match
	matchJSON, err := json.Marshal(match)
	if err != nil {
		return fmt.Errorf("failed to marshal match: %w", err)
	}

	err = rob.client.Set(rob.ctx, matchPrefix+match.ID, matchJSON, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to store match: %w", err)
	}

	// Add to recent trades
	err = rob.client.ZAdd(rob.ctx, recentTradesKey, &redis.Z{
		Score:  float64(match.ExecutedAt),
		Member: match.ID,
	}).Err()

	// Trim recent trades to last 1000
	err = rob.client.ZRemRangeByRank(rob.ctx, recentTradesKey, 0, -1001).Err()
	if err != nil {
		return fmt.Errorf("failed to trim recent trades: %w", err)
	}

	// TODO: Trigger settlement process to transfer funds between users

	return nil
}

// Helper function min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// CancelOrder cancels an open order
func (rob *RedisOrderBook) CancelOrder(orderID, userID string) error {
	rob.mu.Lock()
	defer rob.mu.Unlock()

	// Get order details
	orderJSON, err := rob.client.Get(rob.ctx, orderPrefix+orderID).Result()
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	var order Order
	if err := json.Unmarshal([]byte(orderJSON), &order); err != nil {
		return fmt.Errorf("failed to unmarshal order: %w", err)
	}

	// Verify ownership
	if order.UserID != userID {
		return fmt.Errorf("unauthorized: order does not belong to user")
	}

	// Check if order can be cancelled
	if order.Status != Open && order.Status != PartiallyFilled {
		return fmt.Errorf("cannot cancel order with status %s", order.Status)
	}

	// Update order status
	order.Status = Cancelled
	order.UpdatedAt = time.Now().Unix()

	// Store updated order
	updatedOrderJSON, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal updated order: %w", err)
	}

	err = rob.client.Set(rob.ctx, orderPrefix+order.ID, updatedOrderJSON, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to store updated order: %w", err)
	}

	// Remove order from order book
	var key string
	if order.Type == BidOrder {
		key = bidOrdersKey
	} else {
		key = askOrdersKey
	}

	err = rob.client.ZRem(rob.ctx, key, order.ID).Err()
	if err != nil {
		return fmt.Errorf("failed to remove order from order book: %w", err)
	}

	return nil
}

// GetOrderBook returns the current order book state
func (rob *RedisOrderBook) GetOrderBook(depth int64) (map[string]interface{}, error) {
	rob.mu.Lock()
	defer rob.mu.Unlock()

	if depth <= 0 {
		depth = 10 // Default depth
	}

	// Get top bids (highest prices first)
	bidIDs, err := rob.client.ZRange(rob.ctx, bidOrdersKey, 0, depth-1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get bids: %w", err)
	}

	// Get top asks (lowest prices first)
	askIDs, err := rob.client.ZRange(rob.ctx, askOrdersKey, 0, depth-1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get asks: %w", err)
	}

	// Get bid details
	bids := make([]map[string]interface{}, 0, len(bidIDs))
	for _, id := range bidIDs {
		orderJSON, err := rob.client.Get(rob.ctx, orderPrefix+id).Result()
		if err != nil {
			continue
		}

		var order Order
		if err := json.Unmarshal([]byte(orderJSON), &order); err != nil {
			continue
		}

		bids = append(bids, map[string]interface{}{
			"price":  order.Price,
			"amount": order.Amount - order.Filled,
		})
	}

	// Get ask details
	asks := make([]map[string]interface{}, 0, len(askIDs))
	for _, id := range askIDs {
		orderJSON, err := rob.client.Get(rob.ctx, orderPrefix+id).Result()
		if err != nil {
			continue
		}

		var order Order
		if err := json.Unmarshal([]byte(orderJSON), &order); err != nil {
			continue
		}

		asks = append(asks, map[string]interface{}{
			"price":  order.Price,
			"amount": order.Amount - order.Filled,
		})
	}

	// Get recent trades
	tradeIDs, err := rob.client.ZRevRange(rob.ctx, recentTradesKey, 0, 9).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get recent trades: %w", err)
	}

	// Get trade details
	trades := make([]map[string]interface{}, 0, len(tradeIDs))
	for _, id := range tradeIDs {
		tradeJSON, err := rob.client.Get(rob.ctx, matchPrefix+id).Result()
		if err != nil {
			continue
		}

		var match Match
		if err := json.Unmarshal([]byte(tradeJSON), &match); err != nil {
			continue
		}

		trades = append(trades, map[string]interface{}{
			"price":     match.Price,
			"amount":    match.Amount,
			"timestamp": match.ExecutedAt,
		})
	}

	// Calculate market price from recent trades
	var marketPrice float64
	if len(trades) > 0 {
		var totalVolume, volumeWeightedPrice float64
		for _, trade := range trades {
			price := trade["price"].(float64)
			amount := trade["amount"].(float64)
			totalVolume += amount
			volumeWeightedPrice += price * amount
		}
		if totalVolume > 0 {
			marketPrice = volumeWeightedPrice / totalVolume
		}
	} else if len(bids) > 0 && len(asks) > 0 {
		// If no trades, use spread midpoint
		marketPrice = (bids[0]["price"].(float64) + asks[0]["price"].(float64)) / 2
	}

	return map[string]interface{}{
		"bids":          bids,
		"asks":          asks,
		"market_price":  marketPrice,
		"recent_trades": trades,
		"timestamp":     time.Now().Unix(),
	}, nil
}
