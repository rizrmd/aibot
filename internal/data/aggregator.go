package data

import (
	"aibot/internal/types"
	"sync"
	"time"
)

// CandleTimeframe represents different timeframes for candlesticks
type CandleTimeframe string

const (
	Timeframe1s  CandleTimeframe = "1s"
	Timeframe3s  CandleTimeframe = "3s"
	Timeframe15s CandleTimeframe = "15s"
	Timeframe30s CandleTimeframe = "30s"
	Timeframe1m  CandleTimeframe = "1m"
)

// CandleAggregator aggregates high-frequency data into different timeframes
type CandleAggregator struct {
	// Data storage per symbol and timeframe
	data map[string]map[CandleTimeframe]*TimeframeData
	mu   sync.RWMutex

	// Configuration
	baseInterval time.Duration // Base interval (300ms in your case)
	maxHistory  int           // Maximum candles to keep per timeframe
}

// TimeframeData stores candle data for a specific timeframe
type TimeframeData struct {
	Timeframe CandleTimeframe
	Interval  time.Duration
	Candles   []types.OHLCV
	// Current incomplete candle being built
	CurrentCandle *types.OHLCV
	LastUpdateTime time.Time
}

// AggregatorConfig holds configuration for the candle aggregator
type AggregatorConfig struct {
	BaseInterval time.Duration            `json:"base_interval"`    // 300ms
	MaxHistory   int                      `json:"max_history"`     // Candles per timeframe
	Timeframes   []CandleTimeframe        `json:"timeframes"`      // Which timeframes to generate
	Symbols      []string                 `json:"symbols"`         // Symbols to track
}

// NewCandleAggregator creates a new candle aggregator
func NewCandleAggregator(config AggregatorConfig) *CandleAggregator {
	// Set defaults
	if config.BaseInterval == 0 {
		config.BaseInterval = 300 * time.Millisecond
	}
	if config.MaxHistory == 0 {
		config.MaxHistory = 200
	}
	if len(config.Timeframes) == 0 {
		config.Timeframes = []CandleTimeframe{Timeframe1s, Timeframe3s, Timeframe15s}
	}

	aggregator := &CandleAggregator{
		data:        make(map[string]map[CandleTimeframe]*TimeframeData),
		baseInterval: config.BaseInterval,
		maxHistory:  config.MaxHistory,
	}

	// Initialize data structures for all symbols and timeframes
	for _, symbol := range config.Symbols {
		aggregator.data[symbol] = make(map[CandleTimeframe]*TimeframeData)
		for _, tf := range config.Timeframes {
			interval := aggregator.getTimeframeInterval(tf)
			aggregator.data[symbol][tf] = &TimeframeData{
				Timeframe: tf,
				Interval:  interval,
				Candles:   make([]types.OHLCV, 0),
			}
		}
	}

	return aggregator
}

// AddTick adds a single tick/price update and updates all timeframes
func (ca *CandleAggregator) AddTick(ticker types.Ticker) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	symbolData, exists := ca.data[ticker.Symbol]
	if !exists {
		// Initialize symbol if not exists
		ca.data[ticker.Symbol] = make(map[CandleTimeframe]*TimeframeData)
		symbolData = ca.data[ticker.Symbol]
	}

	for _, tfData := range symbolData {
		ca.updateTimeframe(tfData, ticker)
	}
}

// AddCandle adds a complete OHLCV candle and updates all timeframes
func (ca *CandleAggregator) AddCandle(candle types.OHLCV) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	symbolData, exists := ca.data[candle.Symbol]
	if !exists {
		// Initialize symbol if not exists
		ca.data[candle.Symbol] = make(map[CandleTimeframe]*TimeframeData)
		symbolData = ca.data[candle.Symbol]
	}

	for _, tfData := range symbolData {
		ca.updateTimeframeWithCandle(tfData, candle)
	}
}

// updateTimeframe updates a specific timeframe with ticker data
func (ca *CandleAggregator) updateTimeframe(tfData *TimeframeData, ticker types.Ticker) {
	now := ticker.Timestamp

	// Initialize current candle if needed
	if tfData.CurrentCandle == nil {
		tfData.CurrentCandle = &types.OHLCV{
			Symbol:    ticker.Symbol,
			Timestamp: ca.alignTimeToTimeframe(now, tfData.Interval),
			Open:      ticker.Price,
			High:      ticker.Price,
			Low:       ticker.Price,
			Close:     ticker.Price,
			Volume:    ticker.Volume,
		}
		tfData.LastUpdateTime = now
		return
	}

	// Check if we need to close the current candle and start a new one
	candleEndTime := tfData.CurrentCandle.Timestamp.Add(tfData.Interval)
	if now.After(candleEndTime) || now.Equal(candleEndTime) {
		// Close current candle
		ca.closeCurrentCandle(tfData)

		// Start new candle
		tfData.CurrentCandle = &types.OHLCV{
			Symbol:    ticker.Symbol,
			Timestamp: ca.alignTimeToTimeframe(now, tfData.Interval),
			Open:      ticker.Price,
			High:      ticker.Price,
			Low:       ticker.Price,
			Close:     ticker.Price,
			Volume:    ticker.Volume,
		}
	} else {
		// Update current candle
		ca.updateCurrentCandle(tfData.CurrentCandle, ticker)
	}

	tfData.LastUpdateTime = now
}

// updateTimeframeWithCandle updates a specific timeframe with OHLCV data
func (ca *CandleAggregator) updateTimeframeWithCandle(tfData *TimeframeData, candle types.OHLCV) {
	// For simplicity, add the candle as-is (could be improved with proper aggregation)
	ca.addCandleToHistory(tfData, candle)
}

// updateCurrentCandle updates the current candle with new ticker data
func (ca *CandleAggregator) updateCurrentCandle(currentCandle *types.OHLCV, ticker types.Ticker) {
	currentCandle.High = max(currentCandle.High, ticker.Price)
	currentCandle.Low = min(currentCandle.Low, ticker.Price)
	currentCandle.Close = ticker.Price
	currentCandle.Volume += ticker.Volume
}

// closeCurrentCandle closes the current candle and adds it to history
func (ca *CandleAggregator) closeCurrentCandle(tfData *TimeframeData) {
	if tfData.CurrentCandle != nil {
		ca.addCandleToHistory(tfData, *tfData.CurrentCandle)
		tfData.CurrentCandle = nil
	}
}

// addCandleToHistory adds a candle to the history, maintaining max size
func (ca *CandleAggregator) addCandleToHistory(tfData *TimeframeData, candle types.OHLCV) {
	tfData.Candles = append(tfData.Candles, candle)

	// Limit history size
	if len(tfData.Candles) > ca.maxHistory {
		tfData.Candles = tfData.Candles[1:]
	}
}

// GetCandles returns candles for a specific symbol and timeframe
func (ca *CandleAggregator) GetCandles(symbol string, timeframe CandleTimeframe, limit int) []types.OHLCV {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	symbolData, exists := ca.data[symbol]
	if !exists {
		return nil
	}

	tfData, exists := symbolData[timeframe]
	if !exists {
		return nil
	}

	if limit <= 0 || limit > len(tfData.Candles) {
		limit = len(tfData.Candles)
	}

	// Return the most recent candles
	start := len(tfData.Candles) - limit
	result := make([]types.OHLCV, limit)
	copy(result, tfData.Candles[start:])

	return result
}

// GetCurrentCandle returns the current (incomplete) candle for a symbol and timeframe
func (ca *CandleAggregator) GetCurrentCandle(symbol string, timeframe CandleTimeframe) *types.OHLCV {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	symbolData, exists := ca.data[symbol]
	if !exists {
		return nil
	}

	tfData, exists := symbolData[timeframe]
	if !exists {
		return nil
	}

	if tfData.CurrentCandle != nil {
		candle := *tfData.CurrentCandle
		return &candle
	}

	return nil
}

// GetAllTimeframes returns all available timeframes for a symbol
func (ca *CandleAggregator) GetAllTimeframes(symbol string) map[CandleTimeframe][]types.OHLCV {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	result := make(map[CandleTimeframe][]types.OHLCV)

	symbolData, exists := ca.data[symbol]
	if !exists {
		return result
	}

	for timeframe, tfData := range symbolData {
		candles := make([]types.OHLCV, len(tfData.Candles))
		copy(candles, tfData.Candles)
		result[timeframe] = candles
	}

	return result
}

// GetLatestPrice returns the latest price for a symbol (from any timeframe)
func (ca *CandleAggregator) GetLatestPrice(symbol string) float64 {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	symbolData, exists := ca.data[symbol]
	if !exists {
		return 0
	}

	// Try to get from current candle first, then from completed candles
	for _, tfData := range symbolData {
		if tfData.CurrentCandle != nil {
			return tfData.CurrentCandle.Close
		}
		if len(tfData.Candles) > 0 {
			return tfData.Candles[len(tfData.Candles)-1].Close
		}
	}

	return 0
}

// GetSymbols returns all tracked symbols
func (ca *CandleAggregator) GetSymbols() []string {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	symbols := make([]string, 0, len(ca.data))
	for symbol := range ca.data {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// AddSymbol adds a new symbol to track
func (ca *CandleAggregator) AddSymbol(symbol string, timeframes []CandleTimeframe) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	if _, exists := ca.data[symbol]; exists {
		return // Symbol already exists
	}

	if len(timeframes) == 0 {
		timeframes = []CandleTimeframe{Timeframe1s, Timeframe3s, Timeframe15s}
	}

	ca.data[symbol] = make(map[CandleTimeframe]*TimeframeData)
	for _, tf := range timeframes {
		interval := ca.getTimeframeInterval(tf)
		ca.data[symbol][tf] = &TimeframeData{
			Timeframe: tf,
			Interval:  interval,
			Candles:   make([]types.OHLCV, 0),
		}
	}
}

// RemoveSymbol removes a symbol from tracking
func (ca *CandleAggregator) RemoveSymbol(symbol string) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	delete(ca.data, symbol)
}

// Clear removes all data
func (ca *CandleAggregator) Clear() {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	ca.data = make(map[string]map[CandleTimeframe]*TimeframeData)
}

// GetStats returns statistics about the aggregator
func (ca *CandleAggregator) GetStats() map[string]interface{} {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_symbols"] = len(ca.data)

	symbolStats := make(map[string]interface{})
	for symbol, symbolData := range ca.data {
		timeframeStats := make(map[string]interface{})
		for timeframe, tfData := range symbolData {
			timeframeStats[string(timeframe)] = map[string]interface{}{
				"candles_count": len(tfData.Candles),
				"has_current":   tfData.CurrentCandle != nil,
				"interval":       tfData.Interval.String(),
			}
		}
		symbolStats[symbol] = timeframeStats
	}
	stats["symbols"] = symbolStats

	return stats
}

// getTimeframeInterval returns the duration for a given timeframe
func (ca *CandleAggregator) getTimeframeInterval(timeframe CandleTimeframe) time.Duration {
	switch timeframe {
	case Timeframe1s:
		return 1 * time.Second
	case Timeframe3s:
		return 3 * time.Second
	case Timeframe15s:
		return 15 * time.Second
	case Timeframe30s:
		return 30 * time.Second
	case Timeframe1m:
		return 1 * time.Minute
	default:
		return 1 * time.Second // Default to 1 second
	}
}

// alignTimeToTimeframe aligns a timestamp to the start of a timeframe interval
func (ca *CandleAggregator) alignTimeToTimeframe(t time.Time, interval time.Duration) time.Time {
	// Truncate to the nearest interval
	return t.Truncate(interval)
}

// Helper functions
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}