package stream

import (
	"aibot/internal/types"
	"context"
	"math"
	"math/rand"
	"sync"
	"time"
)

// SimulationProvider provides simulated streaming data for testing
type SimulationProvider struct {
	config           SimulationConfig
	ohlcvChan        chan types.OHLCV
	tickerChan       chan types.Ticker
	subscribedSymbols map[string]bool
	mu               sync.RWMutex
	running          bool
	ctx              context.Context
	cancel           context.CancelFunc
	stats            StreamStats
	lastStatsUpdate  time.Time
	eventChan        chan StreamEvent

	// Simulation state variables
	currentPrices    map[string]float64
	ohlcvAccumulator map[string]*types.OHLCV
	rng              *rand.Rand
}

// NewSimulationProvider creates a new simulation provider
func NewSimulationProvider(config SimulationConfig) *SimulationProvider {
	// Set defaults
	if config.SpeedMultiplier <= 0 {
		config.SpeedMultiplier = 1.0
	}
	if config.RandomVolatility < 0 {
		config.RandomVolatility = 0.1
	}
	if config.CandleInterval <= 0 {
		config.CandleInterval = 300 * time.Millisecond // Match your 300ms update rate
	}
	if config.BufferSize <= 0 {
		config.BufferSize = 1000
	}
	if config.MaxHistoryCandles <= 0 {
		config.MaxHistoryCandles = 200
	}

	// Initialize random number generator
	if config.Seed == 0 {
		config.Seed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(config.Seed))

	// Set default initial prices if not provided
	initialPrices := make(map[string]float64)
	for symbol := range config.VolatilityFactors {
		if price, exists := config.InitialPrices[symbol]; exists {
			initialPrices[symbol] = price
		} else {
			// Default starting prices around $100-1000
			initialPrices[symbol] = 100 + rng.Float64()*900
		}
	}

	return &SimulationProvider{
		config:           config,
		ohlcvChan:        make(chan types.OHLCV, config.BufferSize),
		tickerChan:       make(chan types.Ticker, config.BufferSize),
		subscribedSymbols: make(map[string]bool),
		eventChan:        make(chan StreamEvent, 100),
		currentPrices:    initialPrices,
		ohlcvAccumulator: make(map[string]*types.OHLCV),
		rng:              rng,
	}
}

// Start begins the simulation
func (sp *SimulationProvider) Start(ctx context.Context, symbols []string) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.running {
		return nil // Already running
	}

	sp.ctx, sp.cancel = context.WithCancel(ctx)
	sp.running = true

	// Subscribe to symbols
	for _, symbol := range symbols {
		sp.subscribedSymbols[symbol] = true
	}

	// Start simulation goroutines
	go sp.simulationLoop()
	go sp.statsLoop()

	return nil
}

// Stop stops the simulation
func (sp *SimulationProvider) Stop() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if !sp.running {
		return nil
	}

	sp.cancel()
	sp.running = false

	close(sp.ohlcvChan)
	close(sp.tickerChan)
	close(sp.eventChan)

	return nil
}

// Subscribe adds symbols to the subscription list
func (sp *SimulationProvider) Subscribe(symbols []string) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	for _, symbol := range symbols {
		sp.subscribedSymbols[symbol] = true
		sp.currentPrices[symbol] = sp.getDefaultInitialPrice(symbol)
	}

	return nil
}

// Unsubscribe removes symbols from the subscription list
func (sp *SimulationProvider) Unsubscribe(symbols []string) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	for _, symbol := range symbols {
		delete(sp.subscribedSymbols, symbol)
		delete(sp.currentPrices, symbol)
		delete(sp.ohlcvAccumulator, symbol)
	}

	return nil
}

// GetOHLCVChannel returns the OHLCV channel
func (sp *SimulationProvider) GetOHLCVChannel() <-chan types.OHLCV {
	return sp.ohlcvChan
}

// GetTickerChannel returns the ticker channel
func (sp *SimulationProvider) GetTickerChannel() <-chan types.Ticker {
	return sp.tickerChan
}

// IsConnected returns true if the simulation is running
func (sp *SimulationProvider) IsConnected() bool {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.running
}

// GetSubscribedSymbols returns the list of subscribed symbols
func (sp *SimulationProvider) GetSubscribedSymbols() []string {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	symbols := make([]string, 0, len(sp.subscribedSymbols))
	for symbol := range sp.subscribedSymbols {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// GetLastError returns the last error (always nil for simulation)
func (sp *SimulationProvider) GetLastError() error {
	return nil // Simulation doesn't have connection errors
}

// simulationLoop generates simulated market data
func (sp *SimulationProvider) simulationLoop() {
	ticker := time.NewTicker(sp.config.CandleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sp.ctx.Done():
			return
		case now := <-ticker.C:
			sp.generateTickData(now)
		}
	}
}

// generateTickData generates tick data for all subscribed symbols
func (sp *SimulationProvider) generateTickData(timestamp time.Time) {
	sp.mu.RLock()
	symbols := sp.getSubscribedSymbolsCopy()
	sp.mu.RUnlock()

	for _, symbol := range symbols {
		sp.generateSymbolTick(symbol, timestamp)
	}
}

// generateSymbolTick generates data for a single symbol
func (sp *SimulationProvider) generateSymbolTick(symbol string, timestamp time.Time) {
	currentPrice := sp.currentPrices[symbol]
	volatilityFactor := sp.getVolatilityFactor(symbol)
	trendFactor := sp.getTrendFactor(symbol)

	// Generate price change based on volatility and trend
	priceChange := sp.generatePriceChange(currentPrice, volatilityFactor, trendFactor)
	newPrice := currentPrice + priceChange

	// Ensure price doesn't go negative
	if newPrice < 0.01 {
		newPrice = 0.01
	}

	// Generate volume based on price change magnitude
	volume := sp.generateVolume(symbol, math.Abs(priceChange))

	// Update current price
	sp.currentPrices[symbol] = newPrice

	// Generate OHLCV data
	sp.generateOHLCV(symbol, newPrice, volume, timestamp)

	// Generate ticker data
	sp.generateTicker(symbol, newPrice, volume, timestamp)

	// Update statistics
	sp.stats.MessagesReceived++
	sp.stats.LastMessageTime = timestamp
}

// generatePriceChange generates a realistic price change
func (sp *SimulationProvider) generatePriceChange(currentPrice, volatility, trend float64) float64 {
	// Base random walk with volatility
	randomWalk := (sp.rng.Float64() - 0.5) * 2 * volatility * currentPrice

	// Add trend component
	trendComponent := trend * currentPrice * 0.001 // Small trend influence

	// Add some mean reversion
	meanReversion := -0.1 * (currentPrice - sp.config.InitialPrices[sp.getCurrentSymbol()]) * 0.001

	// Combine all factors with random noise
	noise := sp.rng.NormFloat64() * sp.config.RandomVolatility * currentPrice * 0.01

	priceChange := randomWalk + trendComponent + meanReversion + noise

	// Limit extreme moves
	maxChange := currentPrice * 0.05 // Max 5% change per tick
	if priceChange > maxChange {
		priceChange = maxChange
	} else if priceChange < -maxChange {
		priceChange = -maxChange
	}

	return priceChange
}

// generateVolume generates realistic volume data
func (sp *SimulationProvider) generateVolume(symbol string, priceChange float64) float64 {
	baseVolume := 1000.0 + sp.rng.Float64()*5000.0

	// Higher volume with larger price changes
	volatilityMultiplier := 1.0 + (priceChange/100.0)*10.0

	return baseVolume * volatilityMultiplier * (1.0 + sp.config.RandomVolatility*sp.rng.Float64())
}

// generateOHLCV generates OHLCV candle data
func (sp *SimulationProvider) generateOHLCV(symbol string, price, volume float64, timestamp time.Time) {
	accumulator, exists := sp.ohlcvAccumulator[symbol]
	if !exists {
		accumulator = &types.OHLCV{
			Symbol:    symbol,
			Timestamp: timestamp,
			Open:      price,
			High:      price,
			Low:       price,
			Close:     price,
			Volume:    0,
		}
		sp.ohlcvAccumulator[symbol] = accumulator
	}

	// Update OHLCV
	accumulator.High = math.Max(accumulator.High, price)
	accumulator.Low = math.Min(accumulator.Low, price)
	accumulator.Close = price
	accumulator.Volume += volume

	// Send completed candle if interval is complete
	if time.Since(accumulator.Timestamp) >= sp.config.CandleInterval {
		select {
		case sp.ohlcvChan <- *accumulator:
		case <-sp.ctx.Done():
			return
		}

		// Start new candle
		sp.ohlcvAccumulator[symbol] = &types.OHLCV{
			Symbol:    symbol,
			Timestamp: timestamp,
			Open:      price,
			High:      price,
			Low:       price,
			Close:     price,
			Volume:    0,
		}
	}
}

// generateTicker generates ticker data
func (sp *SimulationProvider) generateTicker(symbol string, price, volume float64, timestamp time.Time) {
	// Generate realistic bid/ask spread
	spread := price * 0.0001 * (1.0 + sp.rng.Float64()*0.5) // 0.01% to 0.015% spread
	bid := price - spread/2
	ask := price + spread/2

	// Generate order book sizes
	bidSize := volume * sp.rng.Float64()
	askSize := volume * sp.rng.Float64()

	ticker := types.Ticker{
		Symbol:    symbol,
		Timestamp: timestamp,
		Price:     price,
		Volume:    volume,
		Bid:       bid,
		Ask:       ask,
		BidSize:   bidSize,
		AskSize:   askSize,
	}

	select {
	case sp.tickerChan <- ticker:
	case <-sp.ctx.Done():
		return
	}
}

// getVolatilityFactor returns volatility factor for a symbol
func (sp *SimulationProvider) getVolatilityFactor(symbol string) float64 {
	if factor, exists := sp.config.VolatilityFactors[symbol]; exists {
		return factor
	}
	return 0.02 // Default 2% volatility
}

// getTrendFactor returns trend factor for a symbol
func (sp *SimulationProvider) getTrendFactor(symbol string) float64 {
	if factor, exists := sp.config.TrendFactors[symbol]; exists {
		return factor
	}
	return 0.0 // No trend by default
}

// getCurrentSymbol returns a random current symbol for mean reversion
func (sp *SimulationProvider) getCurrentSymbol() string {
	symbols := sp.GetSubscribedSymbols()
	if len(symbols) == 0 {
		return "BTCUSDT" // Default
	}
	return symbols[sp.rng.Intn(len(symbols))]
}

// getDefaultInitialPrice returns default initial price for a symbol
func (sp *SimulationProvider) getDefaultInitialPrice(symbol string) float64 {
	if price, exists := sp.config.InitialPrices[symbol]; exists {
		return price
	}
	return 100 + sp.rng.Float64()*900
}

// getSubscribedSymbolsCopy returns a copy of subscribed symbols
func (sp *SimulationProvider) getSubscribedSymbolsCopy() []string {
	symbols := make([]string, 0, len(sp.subscribedSymbols))
	for symbol := range sp.subscribedSymbols {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// statsLoop updates streaming statistics
func (sp *SimulationProvider) statsLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sp.ctx.Done():
			return
		case now := <-ticker.C:
			if !sp.lastStatsUpdate.IsZero() {
				elapsed := now.Sub(sp.lastStatsUpdate).Seconds()
				sp.stats.MessagesPerSecond = float64(sp.stats.MessagesReceived) / elapsed
			}
			sp.lastStatsUpdate = now
		}
	}
}

// GetStats returns current streaming statistics
func (sp *SimulationProvider) GetStats() StreamStats {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.stats
}