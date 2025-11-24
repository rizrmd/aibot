package types

import (
	"time"
)

// PositionType represents the type of position
type PositionType string

const (
	PositionTypeLong  PositionType = "long"
	PositionTypeShort PositionType = "short"
)

// Position represents a trading position
type Position struct {
	ID           string        `json:"id"`
	Symbol       string        `json:"symbol"`
	Type         PositionType  `json:"type"`
	Size         float64       `json:"size"`
	EntryPrice   float64       `json:"entry_price"`
	MarkPrice    float64       `json:"mark_price"`
	UnrealizedPnL float64      `json:"unrealized_pnl"`
	RealizedPnL  float64       `json:"realized_pnl"`
	Leverage     float64       `json:"leverage"`
	Margin       float64       `json:"margin"`
	FeePaid      float64       `json:"fee_paid"`
	EntryTime    time.Time     `json:"entry_time"`
	ExitTime     *time.Time    `json:"exit_time,omitempty"`
	Status       string        `json:"status"` // "open", "closed", "partial"
}

// NewPosition creates a new position
func NewPosition(id, symbol string, posType PositionType, size, entryPrice, leverage float64) *Position {
	now := time.Now()
	margin := size * entryPrice / leverage

	return &Position{
		ID:         id,
		Symbol:     symbol,
		Type:       posType,
		Size:       size,
		EntryPrice: entryPrice,
		Leverage:   leverage,
		Margin:     margin,
		EntryTime:  now,
		Status:     "open",
	}
}

// UpdateMarkPrice updates the mark price and recalculates unrealized PnL
func (p *Position) UpdateMarkPrice(markPrice float64) {
	p.MarkPrice = markPrice
	p.calculateUnrealizedPnL()
}

// calculateUnrealizedPnL calculates the unrealized profit/loss
func (p *Position) calculateUnrealizedPnL() {
	priceDiff := p.MarkPrice - p.EntryPrice

	if p.Type == PositionTypeLong {
		p.UnrealizedPnL = priceDiff * p.Size
	} else {
		p.UnrealizedPnL = -priceDiff * p.Size
	}
}

// GetUnrealizedPnLPercentage returns unrealized PnL as percentage of margin
func (p *Position) GetUnrealizedPnLPercentage() float64 {
	if p.Margin == 0 {
		return 0
	}
	return (p.UnrealizedPnL / p.Margin) * 100
}

// IsProfitable returns true if position is profitable
func (p *Position) IsProfitable() bool {
	return p.UnrealizedPnL > 0
}

// Close closes the position with given exit price
func (p *Position) Close(exitPrice float64, exitFee float64) {
	now := time.Now()
	p.ExitTime = &now
	p.MarkPrice = exitPrice
	p.calculateUnrealizedPnL()
	p.RealizedPnL = p.UnrealizedPnL
	p.FeePaid += exitFee
	p.Status = "closed"
}

// PartialClose partially closes the position
func (p *Position) PartialClose(closeSize float64, exitPrice float64, exitFee float64) {
	if closeSize > p.Size {
		closeSize = p.Size
	}

	// Calculate PnL for the closed portion
	priceDiff := exitPrice - p.EntryPrice
	var closedPnL float64
	if p.Type == PositionTypeLong {
		closedPnL = priceDiff * closeSize
	} else {
		closedPnL = -priceDiff * closeSize
	}

	// Update position
	p.RealizedPnL += closedPnL
	p.FeePaid += exitFee
	p.Size -= closeSize

	if p.Size <= 0.001 { // Threshold for rounding errors
		p.Status = "closed"
		p.ExitTime = &[]time.Time{time.Now()}[0]
	}
}

// GetDuration returns how long the position has been open
func (p *Position) GetDuration() time.Duration {
	if p.ExitTime != nil {
		return p.ExitTime.Sub(p.EntryTime)
	}
	return time.Since(p.EntryTime)
}

// GetTotalCost returns total cost including margin and fees
func (p *Position) GetTotalCost() float64 {
	return p.Margin + p.FeePaid
}

// GetROI returns return on investment percentage
func (p *Position) GetROI() float64 {
	if p.Margin == 0 {
		return 0
	}
	return (p.RealizedPnL / p.Margin) * 100
}