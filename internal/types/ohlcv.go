package types

import (
	"time"
)

// OHLCV represents Open, High, Low, Close, Volume data
type OHLCV struct {
	Symbol    string    `json:"symbol"`
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
}

// NewOHLCV creates a new OHLCV instance
func NewOHLCV(symbol string, timestamp time.Time, open, high, low, close, volume float64) OHLCV {
	return OHLCV{
		Symbol:    symbol,
		Timestamp: timestamp,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
	}
}

// GetPrice returns the closing price (commonly used price)
func (o OHLCV) GetPrice() float64 {
	return o.Close
}

// GetTypicalPrice returns (high + low + close) / 3
func (o OHLCV) GetTypicalPrice() float64 {
	return (o.High + o.Low + o.Close) / 3
}

// GetHL2 returns (high + low) / 2
func (o OHLCV) GetHL2() float64 {
	return (o.High + o.Low) / 2
}

// GetHLC3 returns (high + low + close) / 3 (same as typical price)
func (o OHLCV) GetHLC3() float64 {
	return o.GetTypicalPrice()
}

// GetOHLC4 returns (open + high + low + close) / 4
func (o OHLCV) GetOHLC4() float64 {
	return (o.Open + o.High + o.Low + o.Close) / 4
}

// GetRange returns the price range (high - low)
func (o OHLCV) GetRange() float64 {
	return o.High - o.Low
}

// GetBody returns the absolute difference between open and close
func (o OHLCV) GetBody() float64 {
	return abs(o.Close - o.Open)
}

// IsBullish returns true if close > open
func (o OHLCV) IsBullish() bool {
	return o.Close > o.Open
}

// IsBearish returns true if close < open
func (o OHLCV) IsBearish() bool {
	return o.Close < o.Open
}

// GetUpperWick returns the upper wick size
func (o OHLCV) GetUpperWick() float64 {
	if o.IsBullish() {
		return o.High - o.Close
	}
	return o.High - o.Open
}

// GetLowerWick returns the lower wick size
func (o OHLCV) GetLowerWick() float64 {
	if o.IsBullish() {
		return o.Open - o.Low
	}
	return o.Close - o.Low
}

// Helper function
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}