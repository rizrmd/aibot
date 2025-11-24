package strategy

import (
	"math"
	"time"
)

// FalseBreakoutDetector specializes in detecting false breakouts and managing recovery strategies
type FalseBreakoutDetector struct {
	// Configuration
	FakeoutThreshold     float64 `json:"fakeout_threshold"`      // Price reversion threshold (0.5%)
	ReversalThreshold    float64 `json:"reversal_threshold"`     // Strong reversal threshold (1.0%)
	ConfirmationWindow   int     `json:"confirmation_window"`    // Candles to confirm fakeout (3 candles)
	MinVolumeDecline     float64 `json:"min_volume_decline"`     // Minimum volume drop (50%)
	MomentumReversalTime int     `json:"momentum_reversal_time"` // Time for momentum reversal (900ms)
	MaxFakeoutFrequency   float64 `json:"max_fakeout_frequency"`  // Max fakeout frequency per hour

	// State tracking
	recentPriceChanges   []float64 `json:"recent_price_changes"`
	volumeHistory        []float64 `json:"volume_history"`
	fakeoutCount        int       `json:"fakeout_count"`
	lastBreakoutTime    time.Time `json:"last_breakout_time"`
	recentFakeouts      []time.Time `json:"recent_fakeouts"`

	// Detection thresholds
	atrMultiplier        float64 `json:"atr_multiplier"`        // ATR for reversal detection
	standardDevMultiplier float64 `json:"std_dev_multiplier"`   // Standard deviation threshold
}

// FalseBreakoutSignal represents a detected false breakout
type FalseBreakoutSignal struct {
	SignalType     FalseBreakoutType `json:"signal_type"`
	Confidence     float64           `json:"confidence"`
	ReversalType   ReversalType      `json:"reversal_type"`
	ReversalTarget float64           `json:"reversal_target"`
	Timestamp      time.Time         `json:"timestamp"`
	Symbol         string            `json:"symbol"`
	EntryPrice     float64           `json:"entry_price"`
	CurrentPrice   float64           `json:"current_price"`
	Reasons        []string          `json:"reasons"`
	RecoveryAction string            `json:"recovery_action"`
}

// FalseBreakoutType represents different types of false breakouts
type FalseBreakoutType string

const (
	FalseBreakoutQuickReversal FalseBreakoutType = "quick_reversal"    // Immediate reversion
	FalseBreakoutVolumeDrop    FalseBreakoutType = "volume_drop"       // Volume dries up
	FalseBreakoutMomentumShift FalseBreakoutType = "momentum_shift"    // Momentum reverses
	FalseBreakoutConsolidation FalseBreakoutType = "consolidation"     // Price consolidates
)

// ReversalType represents the type of reversal detected
type ReversalType string

const (
	ReversalTypeTakeProfit ReversalType = "take_profit"    // Exit with profit
	ReversalTypeStopLoss   ReversalType = "stop_loss"      // Cut losses
	ReversalTypeReverse    ReversalType = "reverse"         // Take opposite position
)

// FalseBreakoutConfig holds configuration for false breakout detection
type FalseBreakoutConfig struct {
	PriceReversionThreshold float64 `json:"price_reversion_threshold"` // 0.5%
	StrongReversalThreshold float64 `json:"strong_reversal_threshold"` // 1.0%
	ConfirmationCandles      int     `json:"confirmation_candles"`       // 3 candles
	VolumeDeclineThreshold   float64 `json:"volume_decline_threshold"`   // 0.5 (50% drop)
	MomentumReversalMs      int     `json:"momentum_reversal_ms"`      // 900ms
	ATRMultiplier           float64 `json:"atr_multiplier"`            // 1.5x
	StdDevMultiplier        float64 `json:"std_dev_multiplier"`         // 2.0x
}

// NewFalseBreakoutDetector creates a new false breakout detector
func NewFalseBreakoutDetector(config FalseBreakoutConfig) *FalseBreakoutDetector {
	// Set defaults
	if config.PriceReversionThreshold == 0 {
		config.PriceReversionThreshold = 0.005 // 0.5%
	}
	if config.StrongReversalThreshold == 0 {
		config.StrongReversalThreshold = 0.01 // 1.0%
	}
	if config.ConfirmationCandles == 0 {
		config.ConfirmationCandles = 3
	}
	if config.VolumeDeclineThreshold == 0 {
		config.VolumeDeclineThreshold = 0.5 // 50%
	}
	if config.MomentumReversalMs == 0 {
		config.MomentumReversalMs = 900 // 900ms
	}
		// Use a default value for ATRMultiple since it's not in the config struct
	atrMultiple := 1.5
	if config.StdDevMultiplier == 0 {
		config.StdDevMultiplier = 2.0
	}

	return &FalseBreakoutDetector{
		FakeoutThreshold:     config.PriceReversionThreshold,
		ReversalThreshold:    config.StrongReversalThreshold,
		ConfirmationWindow:   config.ConfirmationCandles,
		MinVolumeDecline:     config.VolumeDeclineThreshold,
		MomentumReversalTime: config.MomentumReversalMs,
		atrMultiplier:        atrMultiple,
		standardDevMultiplier: config.StdDevMultiplier,
		recentPriceChanges:   make([]float64, 0),
		volumeHistory:        make([]float64, 0),
		recentFakeouts:      make([]time.Time, 0),
	}
}

// DetectFalseBreakout analyzes price action to detect false breakouts
func (fb *FalseBreakoutDetector) DetectFalseBreakout(
	symbol string,
	entryPrice, currentPrice float64,
	breakoutType BreakoutType,
	atr, averageVolume float64,
	timeSinceBreakout time.Duration,
) *FalseBreakoutSignal {

	// Update tracking data
	fb.updateTrackingData(currentPrice, averageVolume, timeSinceBreakout)

	// Check different false breakout patterns
	signals := make([]*FalseBreakoutSignal, 0)

	// Quick reversion detection
	if signal := fb.detectQuickReversal(symbol, entryPrice, currentPrice, breakoutType, atr); signal != nil {
		signals = append(signals, signal)
	}

	// Volume drop detection
	if signal := fb.detectVolumeDrop(symbol, averageVolume, timeSinceBreakout); signal != nil {
		signals = append(signals, signal)
	}

	// Momentum shift detection
	if signal := fb.detectMomentumShift(symbol, breakoutType, timeSinceBreakout); signal != nil {
		signals = append(signals, signal)
	}

	// Price consolidation detection
	if signal := fb.detectConsolidation(symbol, timeSinceBreakout); signal != nil {
		signals = append(signals, signal)
	}

	// Combine signals if multiple patterns detected
	if len(signals) > 0 {
		return fb.combineSignals(signals, symbol, entryPrice, currentPrice, breakoutType)
	}

	return nil
}

// detectQuickReversal detects immediate price reversal patterns
func (fb *FalseBreakoutDetector) detectQuickReversal(
	symbol string,
	entryPrice, currentPrice float64,
	breakoutType BreakoutType,
	atr float64,
) *FalseBreakoutSignal {

	priceChange := (currentPrice - entryPrice) / entryPrice
	atrChange := atr / entryPrice

	// Check if price moved back toward the grid
	var isReversal bool
	var reversalStrength float64

	switch breakoutType {
	case BreakoutTypeUp:
		isReversal = priceChange < -fb.FakeoutThreshold && priceChange > -fb.ReversalThreshold
		reversalStrength = math.Abs(priceChange)
	case BreakoutTypeDown:
		isReversal = priceChange > fb.FakeoutThreshold && priceChange < fb.ReversalThreshold
		reversalStrength = math.Abs(priceChange)
	default:
		return nil
	}

	if !isReversal {
		return nil
	}

	// Check if reversal is significant relative to ATR
	if atrChange > 0 && math.Abs(priceChange) < atrChange*0.5 {
		return nil // Reversal too small relative to volatility
	}

	confidence := fb.calculateReversalConfidence(reversalStrength, atrChange)
	reversalType := fb.determineReversalType(reversalStrength, priceChange)

	return &FalseBreakoutSignal{
		SignalType:     FalseBreakoutQuickReversal,
		Confidence:     confidence,
		ReversalType:   reversalType,
		ReversalTarget: entryPrice, // Target is back to entry
		Timestamp:      time.Now(),
		Symbol:         symbol,
		EntryPrice:     entryPrice,
		CurrentPrice:   currentPrice,
		Reasons:        []string{"Quick price reversal detected"},
		RecoveryAction: fb.getRecoveryAction(reversalType),
	}
}

// detectVolumeDrop detects declining volume patterns
func (fb *FalseBreakoutDetector) detectVolumeDrop(
	symbol string,
	averageVolume float64,
	timeSinceBreakout time.Duration,
) *FalseBreakoutSignal {

	if len(fb.volumeHistory) < 5 {
		return nil
	}

	// Calculate recent volume average
	recentVolumeAvg := fb.calculateRecentAverageVolume()
	volumeDropRatio := (averageVolume - recentVolumeAvg) / averageVolume

	// Check if volume has dropped significantly
	if volumeDropRatio < fb.MinVolumeDecline {
		// Volume dropping suggests weakening breakout
		confidence := min(1.0, volumeDropRatio*2) // Scale confidence

		return &FalseBreakoutSignal{
			SignalType:     FalseBreakoutVolumeDrop,
			Confidence:     confidence,
			ReversalType:   ReversalTypeTakeProfit,
			ReversalTarget: 0, // No specific target
			Timestamp:      time.Now(),
			Symbol:         symbol,
			CurrentPrice:   0,
			Reasons:        []string{"Significant volume decline detected"},
			RecoveryAction: "Consider taking profits",
		}
	}

	return nil
}

// detectMomentumShift detects momentum reversals
func (fb *FalseBreakoutDetector) detectMomentumShift(
	symbol string,
	breakoutType BreakoutType,
	timeSinceBreakout time.Duration,
) *FalseBreakoutSignal {

	if len(fb.recentPriceChanges) < 3 {
		return nil
	}

	// Calculate recent momentum (last few changes)
	recentMomentum := fb.calculateRecentMomentum()

	// Check if momentum has reversed direction
	var momentumReversed bool
	var reversalStrength float64

	switch breakoutType {
	case BreakoutTypeUp:
		momentumReversed = recentMomentum < -fb.FakeoutThreshold
		reversalStrength = math.Abs(recentMomentum)
	case BreakoutTypeDown:
		momentumReversed = recentMomentum > fb.FakeoutThreshold
		reversalStrength = math.Abs(recentMomentum)
	default:
		return nil
	}

	if !momentumReversed {
		return nil
	}

	// Check if enough time has passed for momentum shift
	if timeSinceBreakout < time.Duration(fb.MomentumReversalTime)*time.Millisecond {
		return nil
	}

	confidence := min(1.0, reversalStrength*100)
	reversalType := fb.determineReversalType(reversalStrength, recentMomentum)

	return &FalseBreakoutSignal{
		SignalType:     FalseBreakoutMomentumShift,
		Confidence:     confidence,
		ReversalType:   reversalType,
		ReversalTarget: 0, // Determined by position management
		Timestamp:      time.Now(),
		Symbol:         symbol,
		CurrentPrice:   0,
		Reasons:        []string{"Momentum shift detected"},
		RecoveryAction: fb.getRecoveryAction(reversalType),
	}
}

// detectConsolidation detects price consolidation patterns
func (fb *FalseBreakoutDetector) detectConsolidation(
	symbol string,
	timeSinceBreakout time.Duration,
) *FalseBreakoutSignal {

	if len(fb.recentPriceChanges) < fb.ConfirmationWindow {
		return nil
	}

	// Calculate price range and volatility in recent period
	recentPrices := fb.recentPriceChanges[len(fb.recentPriceChanges)-fb.ConfirmationWindow:]
	if len(recentPrices) < 2 {
		return nil
	}

	priceRange := calculateStandardDeviation(recentPrices)
	avgPrice := calculateAverage(recentPrices)

	// Check if price is consolidating (low volatility)
	volatilityRatio := priceRange / avgPrice
	if volatilityRatio < 0.002 { // 0.2% threshold for consolidation
		confidence := min(1.0, (0.002/volatilityRatio))

		return &FalseBreakoutSignal{
			SignalType:     FalseBreakoutConsolidation,
			Confidence:     confidence,
			ReversalType:   ReversalTypeStopLoss,
			ReversalTarget: 0,
			Timestamp:      time.Now(),
			Symbol:         symbol,
			CurrentPrice:   avgPrice,
			Reasons:        []string{"Price consolidation detected"},
			RecoveryAction: "Consider stopping losses",
		}
	}

	return nil
}

// updateTrackingData updates internal tracking data
func (fb *FalseBreakoutDetector) updateTrackingData(currentPrice, averageVolume float64, timeSinceBreakout time.Duration) {
	// Update price changes (keep last 20)
	fb.recentPriceChanges = append(fb.recentPriceChanges, currentPrice)
	if len(fb.recentPriceChanges) > 20 {
		fb.recentPriceChanges = fb.recentPriceChanges[1:]
	}

	// Update volume history
	fb.volumeHistory = append(fb.volumeHistory, averageVolume)
	if len(fb.volumeHistory) > 20 {
		fb.volumeHistory = fb.volumeHistory[1:]
	}

	// Update fakeout count
	fb.lastBreakoutTime = time.Now()
}

// combineSignals combines multiple false breakout signals
func (fb *FalseBreakoutDetector) combineSignals(
	signals []*FalseBreakoutSignal,
	symbol string,
	entryPrice, currentPrice float64,
	breakoutType BreakoutType,
) *FalseBreakoutSignal {

	if len(signals) == 0 {
		return nil
	}

	// Find the highest confidence signal
	var bestSignal *FalseBreakoutSignal
	maxConfidence := 0.0

	for _, signal := range signals {
		if signal.Confidence > maxConfidence {
			maxConfidence = signal.Confidence
			bestSignal = signal
		}
	}

	// Enhance confidence if multiple signals agree
	confidenceBoost := float64(len(signals)-1) * 0.1 // 10% boost per additional signal
	bestSignal.Confidence = min(1.0, bestSignal.Confidence + confidenceBoost)

	// Update combined reasons
	allReasons := []string{"Multiple false breakout patterns detected"}
	for _, signal := range signals {
		allReasons = append(allReasons, signal.Reasons...)
	}
	bestSignal.Reasons = allReasons

	// Update current price if not set
	if bestSignal.CurrentPrice == 0 {
		bestSignal.CurrentPrice = currentPrice
	}
	bestSignal.EntryPrice = entryPrice

	return bestSignal
}

// calculateReversalConfidence calculates confidence based on reversal strength
func (fb *FalseBreakoutDetector) calculateReversalConfidence(strength, atrChange float64) float64 {
	baseConfidence := min(1.0, strength*100)

	// Increase confidence if reversal aligns with volatility
	if atrChange > 0 {
		volatilityScore := min(1.0, (strength/atrChange)*2)
		baseConfidence = (baseConfidence + volatilityScore) / 2
	}

	return baseConfidence
}

// determineReversalType determines the appropriate action type
func (fb *FalseBreakoutDetector) determineReversalType(strength, priceChange float64) ReversalType {
	if math.Abs(priceChange) > fb.ReversalThreshold {
		// Strong reversal - could be good for reversal trade
		return ReversalTypeReverse
	} else if math.Abs(priceChange) > fb.FakeoutThreshold*1.5 {
		// Moderate reversal - take profit
		return ReversalTypeTakeProfit
	} else {
		// Weak reversal - cut losses
		return ReversalTypeStopLoss
	}
}

// getRecoveryAction returns the recommended recovery action
func (fb *FalseBreakoutDetector) getRecoveryAction(reversalType ReversalType) string {
	switch reversalType {
	case ReversalTypeTakeProfit:
		return "Close position and take profit"
	case ReversalTypeStopLoss:
		return "Close position to minimize loss"
	case ReversalTypeReverse:
		return "Consider taking opposite position"
	default:
		return "Monitor closely"
	}
}

// calculateRecentAverageVolume calculates average volume from recent history
func (fb *FalseBreakoutDetector) calculateRecentAverageVolume() float64 {
	if len(fb.volumeHistory) == 0 {
		return 0
	}

	sum := 0.0
	for _, volume := range fb.volumeHistory {
		sum += volume
	}
	return sum / float64(len(fb.volumeHistory))
}

// calculateRecentMomentum calculates momentum from recent price changes
func (fb *FalseBreakoutDetector) calculateRecentMomentum() float64 {
	if len(fb.recentPriceChanges) < 2 {
		return 0
	}

	// Calculate average rate of change over recent period
	totalChange := fb.recentPriceChanges[len(fb.recentPriceChanges)-1] - fb.recentPriceChanges[0]
	avgChange := totalChange / float64(len(fb.recentPriceChanges)-1)

	return avgChange / fb.recentPriceChanges[0]
}

// GetFalseBreakoutStats returns statistics about false breakout detection
func (fb *FalseBreakoutDetector) GetFalseBreakoutStats() map[string]interface{} {
	return map[string]interface{}{
		"fakeout_count":       fb.fakeoutCount,
	"recent_fakeouts":     len(fb.recentFakeouts),
	"last_breakout_time":  fb.lastBreakoutTime,
		"price_history_size":  len(fb.recentPriceChanges),
		"volume_history_size": len(fb.volumeHistory),
	}
}

// RegisterFakeout registers a confirmed false breakout
func (fb *FalseBreakoutDetector) RegisterFakeout(symbol string) {
	fb.fakeoutCount++
	fb.recentFakeouts = append(fb.recentFakeouts, time.Now())

	// Keep only last 10 fakeouts
	if len(fb.recentFakeouts) > 10 {
		fb.recentFakeouts = fb.recentFakeouts[1:]
	}
}

// Helper functions
func calculateStandardDeviation(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	mean := calculateAverage(values)
	sum := 0.0
	for _, value := range values {
		diff := value - mean
		sum += diff * diff
	}
	variance := sum / float64(len(values)-1)
	return math.Sqrt(variance)
}

func calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}