package types

import (
	"time"
)

// Ticker represents real-time price and volume data
type Ticker struct {
	Symbol    string    `json:"symbol"`
	Timestamp time.Time `json:"timestamp"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
	Bid       float64   `json:"bid"`       // Best bid price
	Ask       float64   `json:"ask"`       // Best ask price
	BidSize   float64   `json:"bid_size"`  // Size at best bid
	AskSize   float64   `json:"ask_size"`  // Size at best ask
	High24h   float64   `json:"high_24h"`  // 24-hour high
	Low24h    float64   `json:"low_24h"`   // 24-hour low
	Change24h float64   `json:"change_24h"` // 24-hour price change
	ChangePct float64   `json:"change_pct"` // 24-hour percentage change
}

// NewTicker creates a new ticker instance
func NewTicker(symbol string, timestamp time.Time, price, volume float64) *Ticker {
	return &Ticker{
		Symbol:    symbol,
		Timestamp: timestamp,
		Price:     price,
		Volume:    volume,
	}
}

// GetSpread returns the bid-ask spread
func (t *Ticker) GetSpread() float64 {
	return t.Ask - t.Bid
}

// GetSpreadPercentage returns spread as percentage of mid price
func (t *Ticker) GetSpreadPercentage() float64 {
	if t.Bid == 0 || t.Ask == 0 {
		return 0
	}
	midPrice := (t.Bid + t.Ask) / 2
	spread := t.GetSpread()
	return (spread / midPrice) * 100
}

// GetMidPrice returns the midpoint price
func (t *Ticker) GetMidPrice() float64 {
	if t.Bid == 0 || t.Ask == 0 {
		return t.Price
	}
	return (t.Bid + t.Ask) / 2
}

// GetVolumeWeightedPrice returns volume-weighted average price
func (t *Ticker) GetVolumeWeightedPrice() float64 {
	if t.BidSize == 0 || t.AskSize == 0 {
		return t.GetMidPrice()
	}
	totalSize := t.BidSize + t.AskSize
	return (t.Bid*t.BidSize + t.Ask*t.AskSize) / totalSize
}

// IsBullish24h returns true if 24h change is positive
func (t *Ticker) IsBullish24h() bool {
	return t.Change24h > 0
}

// IsBearish24h returns true if 24h change is negative
func (t *Ticker) IsBearish24h() bool {
	return t.Change24h < 0
}

// GetRange24h returns the 24-hour price range
func (t *Ticker) GetRange24h() float64 {
	return t.High24h - t.Low24h
}

// IsNearHigh returns true if price is near 24h high
func (t *Ticker) IsNearHigh(thresholdPercent float64) bool {
	if t.High24h == 0 {
		return false
	}
	distanceFromHigh := (t.High24h - t.Price) / t.High24h
	return distanceFromHigh <= (thresholdPercent / 100)
}

// IsNearLow returns true if price is near 24h low
func (t *Ticker) IsNearLow(thresholdPercent float64) bool {
	if t.Low24h == 0 {
		return false
	}
	distanceFromLow := (t.Price - t.Low24h) / t.Low24h
	return distanceFromLow <= (thresholdPercent / 100)
}

// Update24hStats updates 24-hour statistics with new data
func (t *Ticker) Update24hStats(high, low, change, changePct float64) {
	t.High24h = high
	t.Low24h = low
	t.Change24h = change
	t.ChangePct = changePct
}

// UpdateOrderBook updates bid/ask data
func (t *Ticker) UpdateOrderBook(bid, ask, bidSize, askSize float64) {
	t.Bid = bid
	t.Ask = ask
	t.BidSize = bidSize
	t.AskSize = askSize
}

// Copy creates a deep copy of the ticker
func (t *Ticker) Copy() *Ticker {
	copy := *t
	return &copy
}

// GetLiquidityScore returns a score based on bid/ask sizes and spread
func (t *Ticker) GetLiquidityScore() float64 {
	if t.BidSize == 0 || t.AskSize == 0 || t.GetSpreadPercentage() == 0 {
		return 0
	}

	// Higher sizes and lower spread = higher liquidity score
	avgSize := (t.BidSize + t.AskSize) / 2
	spreadPenalty := 100 / (t.GetSpreadPercentage() + 1) // Inverse spread penalty

	return avgSize * spreadPenalty / 1000 // Normalize
}

// GetVolatility24h estimates volatility from 24h range
func (t *Ticker) GetVolatility24h() float64 {
	if t.Low24h == 0 || t.High24h == 0 {
		return 0
	}
	return ((t.High24h - t.Low24h) / t.Low24h) * 100
}