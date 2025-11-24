package trading

import (
	"aibot/internal/types"
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// SimulationExecutor provides simulated trading execution
type SimulationExecutor struct {
	config           SimulationConfig
	balance          float64
	availableBalance float64
	positions        map[string]*types.Position
	openOrders       map[string]*types.Order
	orderHistory     []*types.Order
	stats            ExecutionStats
	mu               sync.RWMutex
	connected        bool
	priceFeed        <-chan types.Ticker
	rng              *rand.Rand
	orderCounter     int64
	positionCounter  int64
	tradeCounter     int64
}

// NewSimulationExecutor creates a new simulation executor
func NewSimulationExecutor(config SimulationConfig) *SimulationExecutor {
	// Set defaults
	if config.Balance <= 0 {
		config.Balance = 10000.0 // Default $10,000
	}
	if config.Slippage <= 0 {
		config.Slippage = 0.0005 // 0.05% default slippage
	}
	if config.Latency <= 0 {
		config.Latency = 50 * time.Millisecond // 50ms default latency
	}
	if config.FillProbability <= 0 {
		config.FillProbability = 0.8 // 80% fill probability
	}
	if config.PartialFillRate <= 0 {
		config.PartialFillRate = 0.2 // 20% partial fill rate
	}
	if config.RejectionRate <= 0 {
		config.RejectionRate = 0.01 // 1% rejection rate
	}
	if config.Commission <= 0 {
		config.Commission = 0.0004 // 0.04% default commission
	}

	return &SimulationExecutor{
		config:           config,
		balance:          config.Balance,
		availableBalance: config.Balance,
		positions:        make(map[string]*types.Position),
		openOrders:       make(map[string]*types.Order),
		orderHistory:     make([]*types.Order, 0),
		rng:              rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Connect establishes connection to the simulated exchange
func (se *SimulationExecutor) Connect(ctx context.Context) error {
	se.mu.Lock()
	defer se.mu.Unlock()

	se.connected = true
	se.stats = ExecutionStats{
		TotalOrders:     0,
		SuccessfulOrders: 0,
		FailedOrders:    0,
	}

	return nil
}

// Disconnect closes the connection
func (se *SimulationExecutor) Disconnect() error {
	se.mu.Lock()
	defer se.mu.Unlock()

	se.connected = false
	return nil
}

// IsConnected returns connection status
func (se *SimulationExecutor) IsConnected() bool {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.connected
}

// OpenLong opens a long position
func (se *SimulationExecutor) OpenLong(symbol string, quantity float64, price float64) (*types.OrderResult, error) {
	order := types.NewMarketOrder(
		se.generateOrderID(),
		symbol,
		types.OrderSideBuy,
		quantity,
		types.PositionTypeLong,
	)

	order.Price = price
	return se.executeOrder(order)
}

// OpenShort opens a short position
func (se *SimulationExecutor) OpenShort(symbol string, quantity float64, price float64) (*types.OrderResult, error) {
	order := types.NewMarketOrder(
		se.generateOrderID(),
		symbol,
		types.OrderSideSell,
		quantity,
		types.PositionTypeShort,
	)

	order.Price = price
	return se.executeOrder(order)
}

// CloseLong closes a long position
func (se *SimulationExecutor) CloseLong(symbol string, quantity float64, price float64) (*types.OrderResult, error) {
	order := types.NewMarketOrder(
		se.generateOrderID(),
		symbol,
		types.OrderSideSell,
		quantity,
		types.PositionTypeLong,
	)

	order.Price = price
	order.ReduceOnly = true
	return se.executeOrder(order)
}

// CloseShort closes a short position
func (se *SimulationExecutor) CloseShort(symbol string, quantity float64, price float64) (*types.OrderResult, error) {
	order := types.NewMarketOrder(
		se.generateOrderID(),
		symbol,
		types.OrderSideBuy,
		quantity,
		types.PositionTypeShort,
	)

	order.Price = price
	order.ReduceOnly = true
	return se.executeOrder(order)
}

// PlaceOrder places a generic order
func (se *SimulationExecutor) PlaceOrder(order *types.Order) (*types.OrderResult, error) {
	return se.executeOrder(order)
}

// CancelOrder cancels an order
func (se *SimulationExecutor) CancelOrder(orderID string) error {
	se.mu.Lock()
	defer se.mu.Unlock()

	order, exists := se.openOrders[orderID]
	if !exists {
		return fmt.Errorf("order not found: %s", orderID)
	}

	order.Cancel()
	delete(se.openOrders, orderID)
	se.stats.FailedOrders++

	return nil
}

// GetOrder retrieves an order by ID
func (se *SimulationExecutor) GetOrder(orderID string) (*types.Order, error) {
	se.mu.RLock()
	defer se.mu.RUnlock()

	// Check open orders first
	if order, exists := se.openOrders[orderID]; exists {
		return order, nil
	}

	// Check order history
	for _, order := range se.orderHistory {
		if order.ID == orderID {
			return order, nil
		}
	}

	return nil, fmt.Errorf("order not found: %s", orderID)
}

// GetOpenOrders retrieves all open orders
func (se *SimulationExecutor) GetOpenOrders(symbol string) ([]*types.Order, error) {
	se.mu.RLock()
	defer se.mu.RUnlock()

	var openOrders []*types.Order
	for _, order := range se.openOrders {
		if symbol == "" || order.Symbol == symbol {
			openOrders = append(openOrders, order)
		}
	}

	return openOrders, nil
}

// GetOrderHistory retrieves order history
func (se *SimulationExecutor) GetOrderHistory(symbol string, limit int) ([]*types.Order, error) {
	se.mu.RLock()
	defer se.mu.RUnlock()

	var history []*types.Order
	for _, order := range se.orderHistory {
		if symbol == "" || order.Symbol == symbol {
			history = append(history, order)
		}
	}

	// Apply limit
	if limit > 0 && len(history) > limit {
		history = history[len(history)-limit:]
	}

	return history, nil
}

// GetPosition retrieves a position by symbol
func (se *SimulationExecutor) GetPosition(symbol string) (*types.Position, error) {
	se.mu.RLock()
	defer se.mu.RUnlock()

	if position, exists := se.positions[symbol]; exists {
		// Return a copy to avoid external modification
		positionCopy := *position
		return &positionCopy, nil
	}

	return nil, nil // No position
}

// GetAllPositions retrieves all positions
func (se *SimulationExecutor) GetAllPositions() ([]*types.Position, error) {
	se.mu.RLock()
	defer se.mu.RUnlock()

	var allPositions []*types.Position
	for _, position := range se.positions {
		if position.Size > 0 {
			positionCopy := *position
			allPositions = append(allPositions, &positionCopy)
		}
	}

	return allPositions, nil
}

// GetBalance returns total balance
func (se *SimulationExecutor) GetBalance() (float64, error) {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.balance, nil
}

// GetAvailableBalance returns available balance
func (se *SimulationExecutor) GetAvailableBalance() (float64, error) {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.availableBalance, nil
}

// GetMarginInfo returns margin information
func (se *SimulationExecutor) GetMarginInfo() (*MarginInfo, error) {
	se.mu.RLock()
	defer se.mu.RUnlock()

	var usedMargin float64
	var unrealizedPnL float64

	for _, position := range se.positions {
		usedMargin += position.Margin
		unrealizedPnL += position.UnrealizedPnL
	}

	return &MarginInfo{
		TotalBalance:    se.balance,
		AvailableBalance: se.availableBalance,
		UsedMargin:      usedMargin,
		FreeMargin:      se.availableBalance - usedMargin,
		Leverage:        se.config.DefaultLeverage,
		Currency:        "USD",
	}, nil
}

// GetTicker returns current ticker (would need price feed)
func (se *SimulationExecutor) GetTicker(symbol string) (*types.Ticker, error) {
	// In simulation, this would need access to price feed
	return nil, fmt.Errorf("ticker not available in simulation without price feed")
}

// GetOrderBook returns order book (simulated)
func (se *SimulationExecutor) GetOrderBook(symbol string, depth int) (*OrderBook, error) {
	// Simulate a simple order book
	bids := make([]PriceLevel, depth)
	asks := make([]PriceLevel, depth)

	basePrice := 100.0 // Would get from price feed in real implementation
	for i := 0; i < depth; i++ {
		bids[i] = PriceLevel{
			Price:    basePrice - float64(i+1)*0.01,
			Quantity: 100.0 + float64(i)*10,
			Orders:   int(10 + i),
		}
		asks[i] = PriceLevel{
			Price:    basePrice + float64(i+1)*0.01,
			Quantity: 100.0 + float64(i)*10,
			Orders:   int(10 + i),
		}
	}

	return &OrderBook{
		Symbol:    symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now(),
	}, nil
}

// GetFeeRates returns current fee rates
func (se *SimulationExecutor) GetFeeRates() (*FeeRates, error) {
	return &FeeRates{
		MakerFee:      se.config.Commission,
		TakerFee:      se.config.Commission,
		SettlementFee: 0.0,
	}, nil
}

// GetLeverage returns leverage for a symbol
func (se *SimulationExecutor) GetLeverage(symbol string) (float64, error) {
	return se.config.DefaultLeverage, nil
}

// SetLeverage sets leverage for a symbol
func (se *SimulationExecutor) SetLeverage(symbol string, leverage float64) error {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.config.DefaultLeverage = leverage
	return nil
}

// executeOrder handles the order execution logic
func (se *SimulationExecutor) executeOrder(order *types.Order) (*types.OrderResult, error) {
	se.mu.Lock()
	defer se.mu.Unlock()

	if !se.connected {
		return nil, fmt.Errorf("not connected")
	}

	// Simulate rejection
	if se.rng.Float64() < se.config.RejectionRate {
		order.Status = types.OrderStatusRejected
		se.stats.FailedOrders++
		return &types.OrderResult{
			OrderID: order.ID,
			Status:  "rejected",
		}, nil
	}

	// Add to open orders for limit orders
	if order.Type == types.OrderTypeLimit {
		se.openOrders[order.ID] = order
		// Simulate limit order fill after delay
		go se.simulateLimitOrderFill(order)
		return &types.OrderResult{
			OrderID: order.ID,
			Status:  "pending",
		}, nil
	}

	// Execute market order immediately
	return se.executeMarketOrder(order)
}

// executeMarketOrder executes a market order
func (se *SimulationExecutor) executeMarketOrder(order *types.Order) (*types.OrderResult, error) {
	// Simulate slippage
	slippage := 0.0
	if order.Type == types.OrderTypeMarket {
		slippage = se.rng.NormFloat64() * se.config.Slippage
		if order.IsBuy() {
			slippage = math.Abs(slippage) // Positive slippage for buys
		} else {
			slippage = -math.Abs(slippage) // Negative slippage for sells
		}
	}

	executionPrice := order.Price * (1 + slippage)

	// Calculate fees
	feeRate := se.config.Commission
	if order.Type == types.OrderTypeMarket {
		feeRate = se.config.Commission * 1.5 // Taker fees are higher
	}
	fee := order.Quantity * executionPrice * feeRate

	// Check if reduce-only order
	if order.ReduceOnly {
		position, exists := se.positions[order.Symbol]
		if !exists || position.Size == 0 {
			return nil, fmt.Errorf("no position to reduce for %s", order.Symbol)
		}

		// Adjust quantity to not exceed position size
		if order.Quantity > position.Size {
			order.Quantity = position.Size
		}
	}

	// Simulate partial fill
	var fillQty float64
	if se.rng.Float64() < se.config.PartialFillRate {
		fillQty = order.Quantity * (0.5 + se.rng.Float64()*0.5) // 50-100% fill
	} else {
		fillQty = order.Quantity
	}

	// Fill the order
	order.Fill(fillQty, executionPrice, fee)
	se.orderHistory = append(se.orderHistory, order)
	se.stats.TotalOrders++
	se.stats.SuccessfulOrders++
	se.stats.LastOrderTime = time.Now()

	// Update position
	se.updatePosition(order, executionPrice, fillQty, fee)

	return &types.OrderResult{
		OrderID:     order.ID,
		Symbol:      order.Symbol,
		Side:        string(order.Side),
		PositionType: string(order.PositionType),
		Quantity:    order.Quantity,
		Price:       order.Price,
		FilledQty:   order.FilledQty,
		FilledPrice: order.FilledPrice,
		Fee:         order.Fee,
		Timestamp:   order.UpdateTime,
		Status:      string(order.Status),
	}, nil
}

// simulateLimitOrderFill simulates limit order fills
func (se *SimulationExecutor) simulateLimitOrderFill(order *types.Order) {
	time.Sleep(se.config.Latency)

	se.mu.Lock()
	defer se.mu.Unlock()

	if !se.connected {
		return
	}

	// Check if order still exists and is active
	if existingOrder, exists := se.openOrders[order.ID]; !exists || !existingOrder.IsActive() {
		return
	}

	// Simulate fill probability
	if se.rng.Float64() > se.config.FillProbability {
		return // Don't fill
	}

	// Fill the order at limit price
	fee := order.Quantity * order.Price * se.config.Commission
	order.Fill(order.Quantity, order.Price, fee)

	delete(se.openOrders, order.ID)
	se.orderHistory = append(se.orderHistory, order)
	se.stats.SuccessfulOrders++

	// Update position
	se.updatePosition(order, order.Price, order.FilledQty, fee)
}

// updatePosition updates positions based on order execution
func (se *SimulationExecutor) updatePosition(order *types.Order, executionPrice, fillQty, fee float64) {
	symbol := order.Symbol
	var position *types.Position
	var exists bool

	if position, exists = se.positions[symbol]; !exists {
		// Create new position
		position = types.NewPosition(
			se.generatePositionID(),
			symbol,
			order.PositionType,
			fillQty,
			executionPrice,
			se.config.DefaultLeverage,
		)
		se.positions[symbol] = position
	} else {
		// Update existing position
		if order.ReduceOnly {
			// Closing position
			position.PartialClose(fillQty, executionPrice, fee)
		} else {
			// Adding to position (not typically used in grid strategy)
			// For simplicity, we'll close existing and open new
			position.Close(executionPrice, fee)
			position = types.NewPosition(
				se.generatePositionID(),
				symbol,
				order.PositionType,
				fillQty,
				executionPrice,
				se.config.DefaultLeverage,
			)
			se.positions[symbol] = position
		}
	}

	// Update balance
	se.balance -= fee
	se.availableBalance -= fee

	// Remove closed positions
	if position.Status == "closed" {
		delete(se.positions, symbol)
	}
}

// generateOrderID generates a unique order ID
func (se *SimulationExecutor) generateOrderID() string {
	se.orderCounter++
	return fmt.Sprintf("sim_order_%d_%d", time.Now().UnixNano(), se.orderCounter)
}

// generatePositionID generates a unique position ID
func (se *SimulationExecutor) generatePositionID() string {
	se.positionCounter++
	return fmt.Sprintf("sim_pos_%d_%d", time.Now().UnixNano(), se.positionCounter)
}

// UpdateWithTicker updates positions with new ticker data
func (se *SimulationExecutor) UpdateWithTicker(ticker types.Ticker) {
	se.mu.Lock()
	defer se.mu.Unlock()

	if position, exists := se.positions[ticker.Symbol]; exists {
		position.UpdateMarkPrice(ticker.Price)
	}
}

// GetStats returns execution statistics
func (se *SimulationExecutor) GetStats() ExecutionStats {
	se.mu.RLock()
	defer se.mu.RUnlock()

	// Calculate additional stats
	stats := se.stats
	stats.OpenPositions = len(se.positions)

	var totalUnrealizedPnL float64
	var totalRealizedPnL float64
	for _, position := range se.positions {
		totalUnrealizedPnL += position.UnrealizedPnL
		totalRealizedPnL += position.RealizedPnL
	}
	stats.UnrealizedPnL = totalUnrealizedPnL
	stats.RealizedPnL = totalRealizedPnL

	return stats
}