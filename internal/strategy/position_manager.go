package strategy

import (
	"aibot/internal/types"
	"fmt"
	"math"
	"time"
)

// PositionManager handles deterministic position management for grid strategies
type PositionManager struct {
	// Configuration
	MaxPositionSize     float64 `json:"max_position_size"`     // Maximum position size per symbol
	MaxOpenPositions   int     `json:"max_open_positions"`   // Maximum total positions
	StopLossPercent     float64 `json:"stop_loss_percent"`     // Stop loss percentage
	TakeProfitPercent   float64 `json:"take_profit_percent"`   // Take profit percentage
	RiskPerPosition    float64 `json:"risk_per_position"`    // Risk per position (2%)
	PartialCloseRatio   float64 `json:"partial_close_ratio"`   // Partial close ratio (50%)

	// State tracking
	positions          map[string]*PositionState `json:"positions"`          // Current positions by symbol
	gridStrategies      map[string]*GridState    `json:"grid_strategies"`      // Active grid strategies
	breakoutPositions   map[string]*BreakoutState `json:"breakout_positions"`   // Active breakout positions
	positionHistory     []PositionEvent           `json:"position_history"`     // All position events

	// Risk management
	totalRiskExposure   float64                     `json:"total_risk_exposure"`
	dailyLossLimit      float64                     `json:"daily_loss_limit"`
	positionCounter     int64                       `json:"position_counter"`
	TimeoutHours        int                         `json:"timeout_hours"`
	TrailingStopPercent float64                     `json:"trailing_stop_percent"`

	// Performance tracking
	totalProfit         float64                     `json:"total_profit"`
	totalLoss           float64                     `json:"total_loss"`
	winRate             float64                     `json:"win_rate"`
	averageHoldTime     time.Duration              `json:"average_hold_time"`
}

// PositionState tracks the state of each position
type PositionState struct {
	Position       *types.Position            `json:"position"`
	EntryTime      time.Time                `json:"entry_time"`
	LastUpdate     time.Time                `json:"last_update"`
	StopLoss       float64                  `json:"stop_loss"`
	TakeProfit     float64                  `json:"take_profit"`
	PartialClose    bool                    `json:"partial_close"`
	CloseTriggers   []CloseTrigger           `json:"close_triggers"`
	Notes          []string                `json:"notes"`
}

// GridState tracks the state of grid positions
type GridState struct {
	Symbol        string                    `json:"symbol"`
	GridLevels    []*types.GridLevel        `json:"grid_levels"`
	ActiveLevels  map[string]*types.Order  `json:"active_levels"`
	FilledLevels  []*types.GridLevel        `json:"filled_levels"`
	PendingClose  []string                  `json:"pending_close"`
	TotalValue    float64                   `json:"total_value"`
	LastUpdated   time.Time                 `json:"last_updated"`
}

// BreakoutState tracks breakout position state
type BreakoutState struct {
	Position      *types.Position           `json:"position"`
	EntryPrice     float64                   `json:"entry_price"`
	EntryTime      time.Time                 `json:"entry_time"`
	BreakoutType   BreakoutType              `json:"breakout_type"`
	Confirmation   bool                      `json:"confirmation"`
	StopLoss       float64                   `json:"stop_loss"`
	TakeProfit     float64                   `json:"take_profit"`
	TrailingStop   float64                   `json:"trailing_stop"`
	LastUpdate     time.Time                 `json:"last_update"`
}

// CloseTrigger represents a condition for closing a position
type CloseTrigger struct {
	Type         TriggerType   `json:"type"`         // "stop_loss", "take_profit", "grid_breach", "timeout"
	Price        float64      `json:"price"`        // Trigger price
	Time         time.Time    `json:"time"`         // Trigger time
	Reason       string       `json:"reason"`       // Trigger reason
	Executed     bool         `json:"executed"`     // Whether triggered
	PositionSize float64      `json:"position_size"` // Size at trigger time
}

// TriggerType represents different types of close triggers
type TriggerType string

const (
	TriggerStopLoss    TriggerType = "stop_loss"
	TriggerTakeProfit  TriggerType = "take_profit"
	TriggerGridBreach  TriggerType = "grid_breach"
	TriggerTimeout    TriggerType = "timeout"
	TriggerFalseBreakout TriggerType = "false_breakout"
)

// PositionEvent represents a significant position event
type PositionEvent struct {
	EventType     string    `json:"event_type"`     // "open", "close", "modify", "stop_loss", "take_profit"
	Symbol       string    `json:"symbol"`
	PositionID   string    `json:"position_id"`
	PositionType string    `json:"position_type"`
	Quantity     float64   `json:"quantity"`
	Price        float64   `json:"price"`
	PnL          float64   `json:"pnl"`
	Timestamp    time.Time `json:"timestamp"`
	Reason       string    `json:"reason"`
	TriggerType  string    `json:"trigger_type"`
}

// PositionManagerConfig holds configuration for position management
type PositionManagerConfig struct {
	MaxPositionSize      float64 `json:"max_position_size"`       // Maximum size per position
	MaxOpenPositions     int     `json:"max_open_positions"`      // Maximum positions total
	DefaultStopLoss       float64 `json:"default_stop_loss"`       // Default stop loss % (2%)
	DefaultTakeProfit     float64 `json:"default_take_profit"`     // Default take profit % (3%)
	MaxRiskPerPosition    float64 `json:"max_risk_per_position"`    // Risk per position (2%)
	PartialCloseRatio     float64 `json:"partial_close_ratio"`     // Partial close ratio (50%)
	MaxDailyLoss         float64 `json:"max_daily_loss"`         // Maximum daily loss (5%)
	TimeoutHours          int     `json:"timeout_hours"`          // Position timeout (24h)
	TrailingStopPercent   float64 `json:"trailing_stop_percent"`   // Trailing stop % (1%)
}

// NewPositionManager creates a new position manager
func NewPositionManager(config PositionManagerConfig) *PositionManager {
	// Set defaults
	if config.MaxPositionSize == 0 {
		config.MaxPositionSize = 1000.0
	}
	if config.MaxOpenPositions == 0 {
		config.MaxOpenPositions = 10
	}
	if config.DefaultStopLoss == 0 {
		config.DefaultStopLoss = 0.02 // 2%
	}
	if config.DefaultTakeProfit == 0 {
		config.DefaultTakeProfit = 0.03 // 3%
	}
	if config.MaxRiskPerPosition == 0 {
		config.MaxRiskPerPosition = 0.02 // 2%
	}
	if config.PartialCloseRatio == 0 {
		config.PartialCloseRatio = 0.5 // 50%
	}
	if config.MaxDailyLoss == 0 {
		config.MaxDailyLoss = 0.05 // 5%
	}
	if config.TimeoutHours == 0 {
		config.TimeoutHours = 24
	}
	if config.TrailingStopPercent == 0 {
		config.TrailingStopPercent = 0.01 // 1%
	}

	return &PositionManager{
		MaxPositionSize:   config.MaxPositionSize,
		MaxOpenPositions:  config.MaxOpenPositions,
		StopLossPercent:   config.DefaultStopLoss,
		TakeProfitPercent: config.DefaultTakeProfit,
		RiskPerPosition:  config.MaxRiskPerPosition,
		PartialCloseRatio: config.PartialCloseRatio,
		dailyLossLimit:    config.MaxDailyLoss,
		TimeoutHours:      config.TimeoutHours,
		TrailingStopPercent: config.TrailingStopPercent,
		positions:        make(map[string]*PositionState),
		gridStrategies:    make(map[string]*GridState),
		breakoutPositions: make(map[string]*BreakoutState),
		positionHistory:   make([]PositionEvent, 0),
		positionCounter:   0,
	}
}

// OpenGridPosition opens a new grid position
func (pm *PositionManager) OpenGridPosition(symbol string, positionType types.PositionType, quantity, price float64) (*types.OrderResult, error) {
	// Validate position size
	if quantity > pm.MaxPositionSize {
		return nil, fmt.Errorf("position size %f exceeds maximum %f", quantity, pm.MaxPositionSize)
	}

	// Check position limits
	if len(pm.positions) >= pm.MaxOpenPositions {
		return nil, fmt.Errorf("maximum positions %d already open", pm.MaxOpenPositions)
	}

	// Create new position
	positionID := fmt.Sprintf("%s_%d", symbol, pm.positionCounter)
	position := types.NewPosition(positionID, symbol, positionType, quantity, price, 1.0) // Default 1x leverage

	// Calculate stop loss and take profit
	stopLossPrice := pm.calculateStopLoss(positionType, price)
	takeProfitPrice := pm.calculateTakeProfit(positionType, price)

	// Create position state
	state := &PositionState{
		Position:    position,
		EntryTime:   time.Now(),
		LastUpdate:  time.Now(),
		StopLoss:    stopLossPrice,
		TakeProfit:  takeProfitPrice,
		Notes:       []string{"Grid position opened"},
	}

	// Set up close triggers
	state.CloseTriggers = pm.setupGridTriggers(positionType, price, stopLossPrice, takeProfitPrice)

	pm.positions[symbol] = state
	pm.positionCounter++
	pm.totalRiskExposure += (quantity * price) / 100 // Convert to account units

	// Record event
	pm.recordEvent("open", symbol, positionID, string(positionType), quantity, price, 0, "Grid position opened", "")

	// Calculate position result
	result := &types.OrderResult{
		OrderID:     positionID,
		Symbol:      symbol,
		Side:        pm.getSideForPositionType(positionType),
		PositionType: string(positionType),
		Quantity:    quantity,
		Price:       price,
		FilledQty:   quantity,
		FilledPrice: price,
		Timestamp:   state.EntryTime,
		Status:      "filled",
	}

	return result, nil
}

// OpenBreakoutPosition opens a breakout position with tiered entry
func (pm *PositionManager) OpenBreakoutPosition(symbol string, breakoutType BreakoutType, quantity, price float64, confidence float64) (*types.OrderResult, error) {
	positionType := pm.getPositionTypeFromBreakout(breakoutType)

	// Tiered entry: 50% immediate, 50% after confirmation
	immediateQuantity := quantity * 0.5

	// Open immediate position
	result, err := pm.OpenGridPosition(symbol, positionType, immediateQuantity, price)
	if err != nil {
		return nil, err
	}

	// Mark as breakout position
	if state, exists := pm.positions[symbol]; exists {
		state.Notes = append(state.Notes, "Breakout position (immediate 50%)")
	}

	// Schedule remaining 50% for confirmation (handled by caller)
	return result, nil
}

// CompleteBreakoutPosition completes the second half of breakout entry
func (pm *PositionManager) CompleteBreakoutPosition(symbol string, remainingQuantity, price float64) (*types.OrderResult, error) {
	_ = pm.getPositionTypeFromBreakout(pm.getCurrentBreakoutType(symbol)) // Suppress unused variable warning

	result, err := pm.AddToPosition(symbol, remainingQuantity, price)
	if err != nil {
		return nil, err
	}

	// Update notes
	if state, exists := pm.positions[symbol]; exists {
		state.Notes = append(state.Notes, "Breakout position completed (final 50%)")
	}

	return result, nil
}

// AddToPosition adds to an existing position
func (pm *PositionManager) AddToPosition(symbol string, quantity, price float64) (*types.OrderResult, error) {
	state, exists := pm.positions[symbol]
	if !exists {
		return nil, fmt.Errorf("no position found for symbol %s", symbol)
	}

	// Validate addition
	if state.Position.Size+quantity > pm.MaxPositionSize {
		return nil, fmt.Errorf("position size would exceed maximum")
	}

	// Average entry price calculation
	currentValue := state.Position.Size * state.Position.EntryPrice
	additionalValue := quantity * price
	newSize := state.Position.Size + quantity
	newEntryPrice := (currentValue + additionalValue) / newSize

	// Update position
	state.Position.Size = newSize
	state.Position.EntryPrice = newEntryPrice
	state.Position.UpdateMarkPrice(price)
	state.LastUpdate = time.Now()

	// Recalculate stops based on new entry price
	state.StopLoss = pm.calculateStopLoss(state.Position.Type, newEntryPrice)
	state.TakeProfit = pm.calculateTakeProfit(state.Position.Type, newEntryPrice)

	// Update close triggers based on new levels
	pm.updateTriggers(state, newEntryPrice)

	// Record event
	pm.recordEvent("modify", symbol, state.Position.ID, string(state.Position.Type), quantity, price, state.Position.UnrealizedPnL, "Position size increased", "")

	return &types.OrderResult{
		OrderID:     state.Position.ID,
		Symbol:      symbol,
		Side:        pm.getSideForPositionType(state.Position.Type),
		PositionType: string(state.Position.Type),
		Quantity:    quantity,
		Price:       price,
		FilledQty:   quantity,
		FilledPrice: price,
		Timestamp:   state.LastUpdate,
		Status:      "filled",
	}, nil
}

// ClosePosition closes a position with specified parameters
func (pm *PositionManager) ClosePosition(symbol string, quantity float64, price float64, reason string, triggerType TriggerType) (*types.OrderResult, error) {
	state, exists := pm.positions[symbol]
	if !exists {
		return nil, fmt.Errorf("no position found for symbol %s", symbol)
	}

	// Validate quantity
	if quantity > state.Position.Size {
		quantity = state.Position.Size
	}

	// Calculate PnL for this close
	entryValue := quantity * state.Position.EntryPrice
	exitValue := quantity * price
	pnl := exitValue - entryValue // Simplified PnL calculation

	// Update position
	state.Position.Size -= quantity
	state.Position.UpdateMarkPrice(price)
	state.Position.RealizedPnL += pnl
	state.LastUpdate = time.Now()

	// Update triggers
	state.CloseTriggers = append(state.CloseTriggers, CloseTrigger{
		Type:         triggerType,
	Price:        price,
		Time:         time.Now(),
	Reason:       reason,
		Executed:     true,
		PositionSize: quantity,
	})

	// Close position if fully closed
	if state.Position.Size <= 0.001 { // Threshold for rounding
		state.Position.Status = "closed"
		now := time.Now()
		state.Position.ExitTime = &now

		// Remove from positions
		delete(pm.positions, symbol)
		pm.totalRiskExposure -= entryValue / 100

		// Update performance stats
		if pnl > 0 {
			pm.totalProfit += pnl
		} else {
			pm.totalLoss += math.Abs(pnl)
		}
		pm.updatePerformanceStats()
	}

	// Record event
	eventType := "partial_close"
	if state.Position.Size <= 0.001 {
		eventType = "close"
	}
	pm.recordEvent(eventType, symbol, state.Position.ID, string(state.Position.Type), quantity, price, pnl, reason, string(triggerType))

	// Calculate position result
	result := &types.OrderResult{
		OrderID:     state.Position.ID,
		Symbol:      symbol,
		Side:        pm.getClosingSide(state.Position.Type),
		PositionType: string(state.Position.Type),
		Quantity:    quantity,
		Price:       price,
		FilledQty:   quantity,
		FilledPrice: price,
		Fee:         pm.calculateFees(exitValue),
		Timestamp:   state.LastUpdate,
		Status:      "filled",
	}

	return result, nil
}

// ProcessCloseTriggers checks and processes position close triggers
func (pm *PositionManager) ProcessCloseTriggers(symbol string, currentPrice float64) ([]*types.OrderResult, error) {
	state, exists := pm.positions[symbol]
	if !exists {
		return nil, nil
	}

	var results []*types.OrderResult

	// Update position price
	state.Position.UpdateMarkPrice(currentPrice)
	state.LastUpdate = time.Now()

	// Update trailing stop if configured
	pm.updateTrailingStop(state, currentPrice)

	// Check each trigger
	for i := range state.CloseTriggers {
		trigger := &state.CloseTriggers[i]
		if trigger.Executed {
			continue
		}

		shouldTrigger := false

		switch trigger.Type {
		case TriggerStopLoss:
			shouldTrigger = pm.shouldTriggerStopLoss(state, currentPrice, trigger.Price)
		case TriggerTakeProfit:
			shouldTrigger = pm.shouldTriggerTakeProfit(state, currentPrice, trigger.Price)
		case TriggerGridBreach:
			shouldTrigger = pm.shouldTriggerGridBreach(state, currentPrice)
		case TriggerTimeout:
			shouldTrigger = time.Since(state.EntryTime) > time.Duration(pm.TimeoutHours)*time.Hour
		case TriggerFalseBreakout:
			shouldTrigger = pm.shouldTriggerFalseBreakout(state)
		}

		if shouldTrigger {
			// Execute the trigger
			quantity := trigger.PositionSize
			if quantity == 0 {
				quantity = state.Position.Size
			}

			result, err := pm.ClosePosition(symbol, quantity, currentPrice, trigger.Reason, trigger.Type)
			if err != nil {
				return nil, err
			}

			trigger.Executed = true
			results = append(results, result)

			// Mark other similar triggers as executed to prevent duplicate orders
			for j := range state.CloseTriggers {
				if j != i && state.CloseTriggers[j].Type == trigger.Type {
					state.CloseTriggers[j].Executed = true
				}
			}

			// If position was closed, break early
			if state.Position.Size <= 0.001 {
				break
			}
		}
	}

	return results, nil
}

// GetPosition returns current position state
func (pm *PositionManager) GetPosition(symbol string) (*PositionState, bool) {
	state, exists := pm.positions[symbol]
	return state, exists
}

// GetAllPositions returns all active positions
func (pm *PositionManager) GetAllPositions() map[string]*PositionState {
	positions := make(map[string]*PositionState)
	for symbol, state := range pm.positions {
		positions[symbol] = state
	}
	return positions
}

// GetPositionStats returns position management statistics
func (pm *PositionManager) GetPositionStats() map[string]interface{} {
	activePositions := len(pm.positions)
	totalPositions := len(pm.positionHistory)

	wins := 0
	losses := 0
	for _, event := range pm.positionHistory {
		if event.EventType == "close" {
			if event.PnL > 0 {
				wins++
			} else {
				losses++
			}
		}
	}

	winRate := float64(0)
	if totalPositions > 0 {
		winRate = float64(wins) / float64(totalPositions) * 100
	}

	return map[string]interface{}{
		"active_positions":   activePositions,
		"total_positions":    totalPositions,
		"total_profit":       pm.totalProfit,
		"total_loss":         pm.totalLoss,
		"win_rate":           winRate,
	"average_hold_time":    pm.averageHoldTime,
		"total_risk_exposure": pm.totalRiskExposure,
	"daily_loss_limit":   pm.dailyLossLimit,
		"position_counter":   pm.positionCounter,
	}
}

// Record closing trigger execution
func (pm *PositionManager) recordEvent(eventType, symbol, positionID, positionType string, size, price, pnl float64, reason, triggerType string) {
	event := PositionEvent{
	EventType:    eventType,
		Symbol:      symbol,
		PositionID:  positionID,
	PositionType: positionType,
	Quantity:    size,
	Price:       price,
		PnL:          pnl,
		Timestamp:   time.Now(),
		Reason:      reason,
		TriggerType: triggerType,
	}

	pm.positionHistory = append(pm.positionHistory, event)

	// Keep only last 1000 events
	if len(pm.positionHistory) > 1000 {
		pm.positionHistory = pm.positionHistory[1:]
	}
}

// Helper functions
func (pm *PositionManager) calculateStopLoss(positionType types.PositionType, price float64) float64 {
	switch positionType {
	case types.PositionTypeLong:
		return price * (1 - pm.StopLossPercent)
	case types.PositionTypeShort:
		return price * (1 + pm.StopLossPercent)
	default:
		return price
	}
}

func (pm *PositionManager) calculateTakeProfit(positionType types.PositionType, price float64) float64 {
	switch positionType {
	case types.PositionTypeLong:
		return price * (1 + pm.TakeProfitPercent)
	case types.PositionTypeShort:
		return price * (1 - pm.TakeProfitPercent)
	default:
		return price
	}
}

func (pm *PositionManager) getSideForPositionType(positionType types.PositionType) string {
	switch positionType {
	case types.PositionTypeLong:
		return "buy"
	case types.PositionTypeShort:
		return "sell"
	default:
		return "buy"
	}
}

func (pm *PositionManager) getClosingSide(positionType types.PositionType) string {
	switch positionType {
	case types.PositionTypeLong:
		return "sell"
	case types.PositionTypeShort:
		return "buy"
	default:
		return "sell"
	}
}

func (pm *PositionManager) getPositionTypeFromBreakout(breakoutType BreakoutType) types.PositionType {
	switch breakoutType {
	case BreakoutTypeUp:
		return types.PositionTypeShort // Short on upward breakout
	case BreakoutTypeDown:
		return types.PositionTypeLong  // Long on downward breakout
	default:
		return types.PositionTypeLong
	}
}

func (pm *PositionManager) getCurrentBreakoutType(symbol string) BreakoutType {
	// This would be determined by the breakout detector
	// For now, return None as placeholder
	return BreakoutTypeNone
}

func (pm *PositionManager) setupGridTriggers(positionType types.PositionType, price, stopLoss, takeProfit float64) []CloseTrigger {
	triggers := []CloseTrigger{
		{Type: TriggerStopLoss, Price: stopLoss, Time: time.Now(), Reason: "Initial stop loss", Executed: false},
		{Type: TriggerTakeProfit, Price: takeProfit, Time: time.Now(), Reason: "Initial take profit", Executed: false},
		{Type: TriggerTimeout, Price: 0, Time: time.Now().Add(time.Duration(pm.TimeoutHours) * time.Hour), Reason: "Position timeout", Executed: false},
	}

	return triggers
}

func (pm *PositionManager) updateTriggers(state *PositionState, newEntryPrice float64) {
	// Update stop loss and take profit based on new entry price
	newStopLoss := pm.calculateStopLoss(state.Position.Type, newEntryPrice)
	newTakeProfit := pm.calculateTakeProfit(state.Position.Type, newEntryPrice)

	state.StopLoss = newStopLoss
	state.TakeProfit = newTakeProfit

	// Update existing triggers
	for i := range state.CloseTriggers {
		switch state.CloseTriggers[i].Type {
		case TriggerStopLoss:
			state.CloseTriggers[i].Price = newStopLoss
		case TriggerTakeProfit:
			state.CloseTriggers[i].Price = newTakeProfit
		}
	}
}

func (pm *PositionManager) updateTrailingStop(state *PositionState, currentPrice float64) {
	if state.Position.Type == types.PositionTypeLong {
		// Trailing stop moves up as price increases
		newStopLoss := currentPrice * (1 - pm.TrailingStopPercent)
		if newStopLoss > state.StopLoss {
			state.StopLoss = newStopLoss
			state.StopLoss = math.Max(state.StopLoss, state.Position.EntryPrice*0.99) // Never move below entry
		}
	} else {
		// Trailing stop moves down as price decreases
		newStopLoss := currentPrice * (1 + pm.TrailingStopPercent)
		if newStopLoss < state.StopLoss {
			state.StopLoss = newStopLoss
			state.StopLoss = math.Min(state.StopLoss, state.Position.EntryPrice*1.01) // Never move above entry
		}
	}
}

func (pm *PositionManager) updatePerformanceStats() {
	totalClosed := 0
	totalHoldTime := time.Duration(0)

	for _, event := range pm.positionHistory {
		if event.EventType == "close" {
			totalClosed++
			// Calculate hold time (would need entry timestamp)
			holdTime := event.Timestamp.Sub(event.Timestamp) // Placeholder
			totalHoldTime += holdTime
		}
	}

	if totalClosed > 0 {
		pm.averageHoldTime = totalHoldTime / time.Duration(totalClosed)
	}
}

func (pm *PositionManager) shouldTriggerStopLoss(state *PositionState, currentPrice, triggerPrice float64) bool {
	return (state.Position.Type == types.PositionTypeLong && currentPrice <= triggerPrice) ||
		(state.Position.Type == types.PositionTypeShort && currentPrice >= triggerPrice)
}

func (pm *PositionManager) shouldTriggerTakeProfit(state *PositionState, currentPrice, triggerPrice float64) bool {
	return (state.Position.Type == types.PositionTypeLong && currentPrice >= triggerPrice) ||
		(state.Position.Type == types.PositionTypeShort && currentPrice <= triggerPrice)
}

func (pm *PositionManager) shouldTriggerGridBreach(state *PositionState, currentPrice float64) bool {
	// This would check if price has moved outside grid bounds
	// Implementation depends on grid strategy state
	return false
}

func (pm *PositionManager) shouldTriggerFalseBreakout(state *PositionState) bool {
	// This would check false breakout detector signals
	return false
}

func (pm *PositionManager) calculateFees(notionalValue float64) float64 {
	// Simplified fee calculation (0.04% maker fee average)
	return notionalValue * 0.0004
}