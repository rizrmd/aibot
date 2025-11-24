package strategy

import (
	"aibot/internal/data"
	"aibot/internal/indicators"
	"aibot/internal/types"
	"math"
	"time"
)

// BreakoutDetector detects and confirms price breakouts from grid bounds
type BreakoutDetector struct {
	// Configuration
	ConfirmationCandles  int     `json:"confirmation_candles"`   // Number of candles for confirmation
	MinBreakoutStrength  float64 `json:"min_breakout_strength"`  // Minimum breakout strength (0.3%)
	VolumeMultiplier     float64 `json:"volume_multiplier"`     // Volume requirement (1.5x average)
	MomentumThreshold    float64 `json:"momentum_threshold"`    // RSI overbought/oversold (70/30)
	FalseBreakoutPenalty float64 `json:"false_breakout_penalty"` // Penalty for false breakouts

	// State tracking
	breakoutHistory      []BreakoutEvent `json:"breakout_history"`
	recentPrices         []float64       `json:"recent_prices"`
	recentVolumes        []float64       `json:"recent_volumes"`
	candleAggregator     *data.CandleAggregator
	technicalAnalyzer    *indicators.TechnicalAnalyzer
	signalGenerator      *indicators.SignalGenerator

	// Performance tracking
	falseBreakoutCount   int `json:"false_breakout_count"`
	trueBreakoutCount    int `json:"true_breakout_count"`
	consecutiveFailures  int `json:"consecutive_failures"`
}

// BreakoutType represents the type of breakout
type BreakoutType string

const (
	BreakoutTypeUp     BreakoutType = "up"
	BreakoutTypeDown   BreakoutType = "down"
	BreakoutTypeNone   BreakoutType = "none"
)

// BreakoutSignal represents a detected breakout
type BreakoutSignal struct {
	Type         BreakoutType `json:"type"`
	Confidence   float64      `json:"confidence"`   // 0-1
	Strength     float64      `json:"strength"`     // How far beyond bounds (%)
	VolumeRatio  float64      `json:"volume_ratio"`  // Current vs average volume
	ConfirmCandles int         `json:"confirm_candles"` // Candles confirmed
	Timestamp    time.Time    `json:"timestamp"`
	Symbol       string       `json:"symbol"`
	Price        float64      `json:"price"`
	GridBounds   GridBounds   `json:"grid_bounds"`
	Reasons      []string     `json:"reasons"`      // Why this breakout was detected
}

// BreakoutEvent tracks a breakout from start to completion
type BreakoutEvent struct {
	ID              string        `json:"id"`
	Type            BreakoutType  `json:"type"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         *time.Time    `json:"end_time,omitempty"`
	EntryPrice      float64       `json:"entry_price"`
	MaxPrice        float64       `json:"max_price"`      // For upward breakouts
	MinPrice        float64       `json:"min_price"`      // For downward breakouts
	ExitPrice       float64       `json:"exit_price,omitempty"`
	Profit          float64       `json:"profit,omitempty"`
	Confirmation    bool          `json:"confirmation"`
	WasReal         bool          `json:"was_real"`       // True if breakout was sustained
	Duration        time.Duration `json:"duration"`
	Symbol          string        `json:"symbol"`
}

// GridBounds represents the current grid boundaries
type GridBounds struct {
	UpperBound float64 `json:"upper_bound"`
	LowerBound float64 `json:"lower_bound"`
	Center     float64 `json:"center"`
	Range      float64 `json:"range"`
}

// BreakoutConfig holds configuration for breakout detection
type BreakoutConfig struct {
	ConfirmationPeriod  int     `json:"confirmation_period"`  // 3 candles (900ms at 300ms each)
	MinBreakoutStrength float64 `json:"min_breakout_strength"` // 0.3% (0.003)
	VolumeMultiplier    float64 `json:"volume_multiplier"`    // 1.5x average volume
	RSIOverbought       float64 `json:"rsi_overbought"`       // 70
	RSIOversold         float64 `json:"rsi_oversold"`         // 30
	ATRMultiple         float64 `json:"atr_multiple"`         // 1.5x ATR
	MomentumThreshold   float64 `json:"momentum_threshold"`   // 0.5% momentum requirement
}

// NewBreakoutDetector creates a new breakout detector
func NewBreakoutDetector(config BreakoutConfig, aggregator *data.CandleAggregator, analyzer *indicators.TechnicalAnalyzer) *BreakoutDetector {
	// Set defaults
	if config.ConfirmationPeriod == 0 {
		config.ConfirmationPeriod = 3
	}
	if config.MinBreakoutStrength == 0 {
		config.MinBreakoutStrength = 0.003 // 0.3%
	}
	if config.VolumeMultiplier == 0 {
		config.VolumeMultiplier = 1.5
	}
	if config.RSIOverbought == 0 {
		config.RSIOverbought = 70
	}
	if config.RSIOversold == 0 {
		config.RSIOversold = 30
	}
	if config.ATRMultiple == 0 {
		config.ATRMultiple = 1.5
	}
	if config.MomentumThreshold == 0 {
		config.MomentumThreshold = 0.005 // 0.5%
	}

	return &BreakoutDetector{
		ConfirmationCandles:  config.ConfirmationPeriod,
		MinBreakoutStrength:  config.MinBreakoutStrength,
		VolumeMultiplier:     config.VolumeMultiplier,
		MomentumThreshold:    config.MomentumThreshold,
		candleAggregator:     aggregator,
		technicalAnalyzer:    analyzer,
		signalGenerator:      indicators.NewSignalGenerator(indicators.SignalThresholds{
			RSIOverbought: config.RSIOverbought,
			RSIOversold:   config.RSIOversold,
		}),
		breakoutHistory: make([]BreakoutEvent, 0),
		recentPrices:    make([]float64, 0),
		recentVolumes:   make([]float64, 0),
	}
}

// DetectBreakout analyzes current price and detects potential breakouts
func (bd *BreakoutDetector) DetectBreakout(symbol string, gridBounds GridBounds, currentPrice float64) *BreakoutSignal {
	// Update price history
	bd.updatePriceHistory(currentPrice)

	// Check for breakout conditions
	breakoutType := bd.checkBreakoutCondition(gridBounds, currentPrice)
	if breakoutType == BreakoutTypeNone {
		return nil
	}

	// Get indicator values for confirmation
	indicatorValues := bd.technicalAnalyzer.GetIndicatorValues(symbol)
	if indicatorValues == nil {
		return nil
	}

	// Calculate breakout strength
	strength := bd.calculateBreakoutStrength(gridBounds, currentPrice, breakoutType)

	// Check volume confirmation
	volumeRatio := bd.checkVolumeConfirmation(symbol, indicatorValues)

	// Get current timeframe for momentum analysis
	currentCandle := bd.candleAggregator.GetCurrentCandle(symbol, data.Timeframe3s)
	if currentCandle == nil {
		return nil
	}

	// Analyze momentum and other technical factors
	momentum := bd.calculateMomentum(currentCandle, indicatorValues)
	rsiCondition := bd.checkRSICondition(indicatorValues, breakoutType)
	atrCondition := bd.checkATRCondition(indicatorValues, currentPrice)

	// Calculate confidence score
	confidence := bd.calculateConfidence(strength, volumeRatio, momentum, rsiCondition, atrCondition)

	// Generate reasons
	reasons := bd.generateBreakoutReasons(breakoutType, strength, volumeRatio, momentum, rsiCondition, atrCondition)

	// Create breakout signal
	signal := &BreakoutSignal{
		Type:           breakoutType,
		Confidence:     confidence,
		Strength:       strength,
		VolumeRatio:    volumeRatio,
		ConfirmCandles: 0,
		Timestamp:      time.Now(),
		Symbol:         symbol,
		Price:          currentPrice,
		GridBounds:     gridBounds,
		Reasons:        reasons,
	}

	// Start tracking this breakout if confidence is high enough
	if confidence >= 0.6 {
		bd.startBreakoutTracking(signal)
	}

	return signal
}

// ConfirmBreakout confirms or rejects a breakout after the confirmation period
func (bd *BreakoutDetector) ConfirmBreakout(breakoutID string, symbol string, currentPrice float64) bool {
	// Find the breakout event
	var breakout *BreakoutEvent
	for i := range bd.breakoutHistory {
		if bd.breakoutHistory[i].ID == breakoutID {
			breakout = &bd.breakoutHistory[i]
			break
		}
	}

	if breakout == nil {
		return false
	}

	// Check if breakout is still valid
	isValid := bd.validateBreakout(breakout, currentPrice)

	if isValid {
		breakout.Confirmation = true
		bd.trueBreakoutCount++
		bd.consecutiveFailures = 0
	} else {
		bd.falseBreakoutCount++
		bd.consecutiveFailures++
	}

	// End the breakout
	now := time.Now()
	breakout.EndTime = &now
	breakout.Duration = now.Sub(breakout.StartTime)
	breakout.WasReal = isValid

	return isValid
}

// updatePriceHistory updates the recent price and volume history
func (bd *BreakoutDetector) updatePriceHistory(currentPrice float64) {
	// Add to recent prices (keep last 20)
	bd.recentPrices = append(bd.recentPrices, currentPrice)
	if len(bd.recentPrices) > 20 {
		bd.recentPrices = bd.recentPrices[1:]
	}

	// Add to recent volumes (using a simple estimation)
	estimatedVolume := 1000.0 // This would come from actual ticker data
	bd.recentVolumes = append(bd.recentVolumes, estimatedVolume)
	if len(bd.recentVolumes) > 20 {
		bd.recentVolumes = bd.recentVolumes[1:]
	}
}

// checkBreakoutCondition checks if price has broken out of grid bounds
func (bd *BreakoutDetector) checkBreakoutCondition(bounds GridBounds, price float64) BreakoutType {
	breakoutThreshold := bounds.Range * bd.MinBreakoutStrength

	if price > bounds.UpperBound+breakoutThreshold {
		return BreakoutTypeUp
	} else if price < bounds.LowerBound-breakoutThreshold {
		return BreakoutTypeDown
	}

	return BreakoutTypeNone
}

// calculateBreakoutStrength calculates how strong the breakout is
func (bd *BreakoutDetector) calculateBreakoutStrength(bounds GridBounds, price float64, breakoutType BreakoutType) float64 {
	switch breakoutType {
	case BreakoutTypeUp:
		return ((price - bounds.UpperBound) / bounds.UpperBound) * 100
	case BreakoutTypeDown:
		return ((bounds.LowerBound - price) / bounds.LowerBound) * 100
	default:
		return 0
	}
}

// checkVolumeConfirmation checks if volume supports the breakout
func (bd *BreakoutDetector) checkVolumeConfirmation(symbol string, indicatorValues *indicators.IndicatorValues) float64 {
	if indicatorValues.VolumeSMA == 0 {
		return 1.0 // Default if no volume data
	}

	// Get recent candle for volume
	candles := bd.candleAggregator.GetCandles(symbol, data.Timeframe3s, 1)
	if len(candles) == 0 {
		return 1.0
	}

	currentVolume := candles[len(candles)-1].Volume
	volumeRatio := currentVolume / indicatorValues.VolumeSMA

	return volumeRatio
}

// calculateMomentum calculates price momentum
func (bd *BreakoutDetector) calculateMomentum(candle *types.OHLCV, indicatorValues *indicators.IndicatorValues) float64 {
	if len(bd.recentPrices) < 2 {
		return 0
	}

	// Calculate price change percentage
	priceChange := (candle.Close - candle.Open) / candle.Open
	return priceChange
}

// checkRSICondition checks if RSI supports the breakout
func (bd *BreakoutDetector) checkRSICondition(indicatorValues *indicators.IndicatorValues, breakoutType BreakoutType) bool {
	if indicatorValues.RSI == 0 {
		return true // No RSI data
	}

	switch breakoutType {
	case BreakoutTypeUp:
		return indicatorValues.RSI > 50 // Bullish momentum
	case BreakoutTypeDown:
		return indicatorValues.RSI < 50 // Bearish momentum
	default:
		return true
	}
}

// checkATRCondition checks if ATR supports the breakout
func (bd *BreakoutDetector) checkATRCondition(indicatorValues *indicators.IndicatorValues, price float64) bool {
	if indicatorValues.ATR == 0 {
		return true // No ATR data
	}

	// Check if breakout is at least ATR multiple
	atrPercentage := (indicatorValues.ATR / price) * 100
	return atrPercentage >= bd.MomentumThreshold*100
}

// calculateConfidence calculates overall confidence in the breakout
func (bd *BreakoutDetector) calculateConfidence(strength, volumeRatio, momentum float64, rsiCondition, atrCondition bool) float64 {
	// Base confidence from strength
	strengthScore := min(1.0, strength/5.0) // 5% strength = full confidence

	// Volume confirmation
	volumeScore := min(1.0, volumeRatio/2.0) // 2x volume = full confidence

	// Momentum score
	momentumScore := min(1.0, math.Abs(momentum)/0.01) // 1% move = full confidence

	// Technical conditions
	technicalScore := 0.5 // Base score
	if rsiCondition {
		technicalScore += 0.25
	}
	if atrCondition {
		technicalScore += 0.25
	}

	// Adjust for false breakout history
	penalty := float64(bd.consecutiveFailures) * 0.1
	penalty = min(0.5, penalty) // Max 50% penalty

	// Weighted average
	confidence := (strengthScore*0.3 + volumeScore*0.25 + momentumScore*0.25 + technicalScore*0.2)
	confidence -= penalty

	return max(0, min(1.0, confidence))
}

// generateBreakoutReasons creates human-readable reasons for the breakout
func (bd *BreakoutDetector) generateBreakoutReasons(breakoutType BreakoutType, strength, volumeRatio, momentum float64, rsiCondition, atrCondition bool) []string {
	var reasons []string

	switch breakoutType {
	case BreakoutTypeUp:
		reasons = append(reasons, "Price broke above upper grid bound")
	case BreakoutTypeDown:
		reasons = append(reasons, "Price broke below lower grid bound")
	}

	if strength > 0.5 {
		reasons = append(reasons, "Strong breakout detected")
	}

	if volumeRatio > bd.VolumeMultiplier {
		reasons = append(reasons, "High volume confirmation")
	} else if volumeRatio > 1.0 {
		reasons = append(reasons, "Moderate volume support")
	}

	if math.Abs(momentum) > bd.MomentumThreshold {
		reasons = append(reasons, "Strong momentum indicator")
	}

	if rsiCondition {
		reasons = append(reasons, "RSI supports breakout direction")
	}

	if atrCondition {
		reasons = append(reasons, "ATR indicates significant move")
	}

	if bd.consecutiveFailures > 0 {
		reasons = append(reasons, "Recent false breakouts detected")
	}

	return reasons
}

// startBreakoutTracking begins tracking a new breakout event
func (bd *BreakoutDetector) startBreakoutTracking(signal *BreakoutSignal) {
	breakout := BreakoutEvent{
		ID:         generateBreakoutID(),
		Type:       signal.Type,
		StartTime:  signal.Timestamp,
		EntryPrice: signal.Price,
		MaxPrice:   signal.Price,
		MinPrice:   signal.Price,
		Symbol:     signal.Symbol,
		Confirmation: false,
		WasReal:    false,
	}

	if signal.Type == BreakoutTypeUp {
		breakout.MaxPrice = signal.Price
	} else {
		breakout.MinPrice = signal.Price
	}

	bd.breakoutHistory = append(bd.breakoutHistory, breakout)

	// Keep only last 50 events
	if len(bd.breakoutHistory) > 50 {
		bd.breakoutHistory = bd.breakoutHistory[1:]
	}
}

// validateBreakout checks if a breakout is still valid
func (bd *BreakoutDetector) validateBreakout(breakout *BreakoutEvent, currentPrice float64) bool {
	timeSinceStart := time.Since(breakout.StartTime)
	minDuration := time.Duration(bd.ConfirmationCandles) * 300 * time.Millisecond

	if timeSinceStart < minDuration {
		// Not enough time passed yet
		return false
	}

	// Check if breakout is still in the right direction
	switch breakout.Type {
	case BreakoutTypeUp:
		if currentPrice <= breakout.EntryPrice {
			return false // Failed upward breakout
		}
	case BreakoutTypeDown:
		if currentPrice >= breakout.EntryPrice {
			return false // Failed downward breakout
		}
	}

	// Update max/min prices
	if currentPrice > breakout.MaxPrice {
		breakout.MaxPrice = currentPrice
	}
	if currentPrice < breakout.MinPrice {
		breakout.MinPrice = currentPrice
	}

	return true
}

// GetBreakoutStats returns breakout detection statistics
func (bd *BreakoutDetector) GetBreakoutStats() map[string]interface{} {
	totalBreakouts := bd.trueBreakoutCount + bd.falseBreakoutCount
	successRate := float64(0)
	if totalBreakouts > 0 {
		successRate = float64(bd.trueBreakoutCount) / float64(totalBreakouts) * 100
	}

	return map[string]interface{}{
		"total_breakouts":      totalBreakouts,
		"true_breakouts":       bd.trueBreakoutCount,
		"false_breakouts":      bd.falseBreakoutCount,
		"success_rate":         successRate,
		"consecutive_failures": bd.consecutiveFailures,
		"recent_events":        len(bd.breakoutHistory),
	}
}

// GetActiveBreakouts returns currently active (unconfirmed) breakouts
func (bd *BreakoutDetector) GetActiveBreakouts() []BreakoutEvent {
	var active []BreakoutEvent
	for _, breakout := range bd.breakoutHistory {
		if !breakout.Confirmation && breakout.EndTime == nil {
			active = append(active, breakout)
		}
	}
	return active
}

// Helper functions
func generateBreakoutID() string {
	return time.Now().Format("20060102150405") + "-" + string(rune(time.Now().UnixNano()%1000))
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}