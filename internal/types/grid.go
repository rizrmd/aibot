package types

import (
	"time"
)

// GridLevel represents a single grid level
type GridLevel struct {
	ID       string     `json:"id"`
	Symbol   string     `json:"symbol"`
	Price    float64    `json:"price"`
	Quantity float64    `json:"quantity"`
	Side     OrderSide  `json:"side"` // "buy" or "sell"
	OrderID  string     `json:"order_id,omitempty"` // The order ID if placed
	Active   bool       `json:"active"` // Whether this level is currently active
	Filled   bool       `json:"filled"` // Whether this level has been filled
	FillTime *time.Time `json:"fill_time,omitempty"` // When this level was filled
}

// NewGridLevel creates a new grid level
func NewGridLevel(id, symbol string, price, quantity float64, side OrderSide) *GridLevel {
	return &GridLevel{
		ID:       id,
		Symbol:   symbol,
		Price:    price,
		Quantity: quantity,
		Side:     side,
		Active:   true,
		Filled:   false,
	}
}

// MarkFilled marks the grid level as filled
func (gl *GridLevel) MarkFilled(orderID string) {
	now := time.Now()
	gl.Filled = true
	gl.Active = false
	gl.OrderID = orderID
	gl.FillTime = &now
}

// GetExpectedProfit calculates expected profit for this grid level
func (gl *GridLevel) GetExpectedProfit(gridSpread float64, makerFee, takerFee float64) float64 {
	// Grid strategy: maker fee for entry, taker fee for exit
	totalFees := makerFee + takerFee
	profitBeforeFees := gridSpread * gl.Quantity * gl.Price
	return profitBeforeFees - (totalFees * gl.Quantity * gl.Price)
}

// GetExpectedProfitPercentage returns profit as percentage of investment
func (gl *GridLevel) GetExpectedProfitPercentage(gridSpread float64, makerFee, takerFee float64) float64 {
	investment := gl.Quantity * gl.Price
	if investment == 0 {
		return 0
	}
	return (gl.GetExpectedProfit(gridSpread, makerFee, takerFee) / investment) * 100
}

// GridStrategy represents the entire grid strategy configuration
type GridStrategy struct {
	ID             string       `json:"id"`
	Symbol         string       `json:"symbol"`
	UpperBound     float64      `json:"upper_bound"`
	LowerBound     float64      `json:"lower_bound"`
	GridLevels     []*GridLevel `json:"grid_levels"`
	GridSpacing    float64      `json:"grid_spacing"`
	LevelCount     int          `json:"level_count"`
	TotalQuantity  float64      `json:"total_quantity"`
	QuantityPerLevel float64    `json:"quantity_per_level"`
	Active         bool         `json:"active"`
	Mode           string       `json:"mode"` // "grid" or "breakout"
	CreateTime     time.Time    `json:"create_time"`
	UpdateTime     time.Time    `json:"update_time"`
	LastFillTime   *time.Time   `json:"last_fill_time,omitempty"`
}

// GridConfig represents grid configuration parameters
type GridConfig struct {
	Symbol               string  `json:"symbol"`
	InitialBalance       float64 `json:"initial_balance"`
	RiskPerTrade         float64 `json:"risk_per_trade"` // Percentage (e.g., 0.02 for 2%)
	GridSpreadPercentage float64 `json:"grid_spread_percentage"` // Minimum spread per level
	MakerFee             float64 `json:"maker_fee"` // 0.0002 for 0.02%
	TakerFee             float64 `json:"taker_fee"` // 0.0006 for 0.06%
	MinSpread            float64 `json:"min_spread"` // Minimum spread after fees
	MaxLevels            int     `json:"max_levels"` // Maximum grid levels
	VolatilityMultiplier float64 `json:"volatility_multiplier"` // ATR multiplier for bounds
}

// NewGridStrategy creates a new grid strategy
func NewGridStrategy(id, symbol string, upperBound, lowerBound float64, levelCount int, totalQuantity float64) *GridStrategy {
	now := time.Now()
	gridSpacing := (upperBound - lowerBound) / float64(levelCount)
	quantityPerLevel := totalQuantity / float64(levelCount)

	levels := make([]*GridLevel, levelCount)
	for i := 0; i < levelCount; i++ {
		price := lowerBound + gridSpacing*float64(i+1)
		side := OrderSideBuy // Default: start with buy orders at lower levels
		if i >= levelCount/2 {
			side = OrderSideSell // Upper half: sell orders
		}

		levelID := id + "_level_" + string(rune(i))
		levels[i] = NewGridLevel(levelID, symbol, price, quantityPerLevel, side)
	}

	return &GridStrategy{
		ID:               id,
		Symbol:           symbol,
		UpperBound:       upperBound,
		LowerBound:       lowerBound,
		GridLevels:       levels,
		GridSpacing:      gridSpacing,
		LevelCount:       levelCount,
		TotalQuantity:    totalQuantity,
		QuantityPerLevel: quantityPerLevel,
		Active:           true,
		Mode:             "grid",
		CreateTime:       now,
		UpdateTime:       now,
	}
}

// GetActiveLevels returns active (not filled) grid levels
func (gs *GridStrategy) GetActiveLevels() []*GridLevel {
	var activeLevels []*GridLevel
	for _, level := range gs.GridLevels {
		if level.Active && !level.Filled {
			activeLevels = append(activeLevels, level)
		}
	}
	return activeLevels
}

// GetFilledLevels returns filled grid levels
func (gs *GridStrategy) GetFilledLevels() []*GridLevel {
	var filledLevels []*GridLevel
	for _, level := range gs.GridLevels {
		if level.Filled {
			filledLevels = append(filledLevels, level)
		}
	}
	return filledLevels
}

// GetLevelsBySide returns grid levels filtered by side
func (gs *GridStrategy) GetLevelsBySide(side OrderSide) []*GridLevel {
	var levels []*GridLevel
	for _, level := range gs.GridLevels {
		if level.Side == side {
			levels = append(levels, level)
		}
	}
	return levels
}

// UpdateBounds updates the grid bounds and recalculates levels
func (gs *GridStrategy) UpdateBounds(newUpper, newLower float64) {
	gs.UpperBound = newUpper
	gs.LowerBound = newLower
	gs.GridSpacing = (newUpper - newLower) / float64(gs.LevelCount)
	gs.UpdateTime = time.Now()

	// Recalculate level prices
	for i, level := range gs.GridLevels {
		level.Price = newLower + gs.GridSpacing*float64(i+1)
	}
}

// GetCenterPrice returns the center price of the grid
func (gs *GridStrategy) GetCenterPrice() float64 {
	return (gs.UpperBound + gs.LowerBound) / 2
}

// IsPriceInGrid checks if price is within grid bounds
func (gs *GridStrategy) IsPriceInGrid(price float64) bool {
	return price >= gs.LowerBound && price <= gs.UpperBound
}

// GetNearestGridLevel returns the nearest grid level to the given price
func (gs *GridStrategy) GetNearestGridLevel(price float64) *GridLevel {
	var nearest *GridLevel
	minDistance := float64(1e9) // Large initial value

	for _, level := range gs.GridLevels {
		distance := absFloat(price - level.Price)
		if distance < minDistance {
			minDistance = distance
			nearest = level
		}
	}

	return nearest
}

// ExpandBounds expands the grid bounds by the given percentage
func (gs *GridStrategy) ExpandBounds(percentage float64) {
	rangeSize := gs.UpperBound - gs.LowerBound
	expansion := rangeSize * (percentage / 100)

	newUpper := gs.UpperBound + expansion
	newLower := gs.LowerBound - expansion

	// Ensure lower bound doesn't go negative
	if newLower < 0 {
		newLower = 0
	}

	gs.UpdateBounds(newUpper, newLower)
}

// GetExpectedTotalProfit calculates expected profit for all levels
func (gs *GridStrategy) GetExpectedTotalProfit(makerFee, takerFee float64) float64 {
	var totalProfit float64
	for _, level := range gs.GetActiveLevels() {
		totalProfit += level.GetExpectedProfit(gs.GridSpacing, makerFee, takerFee)
	}
	return totalProfit
}

// Helper function
func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}