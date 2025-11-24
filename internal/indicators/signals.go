package indicators

import (
	"math"
)

// SignalType represents different types of trading signals
type SignalType string

const (
	SignalBuy         SignalType = "buy"
	SignalSell        SignalType = "sell"
	SignalHold        SignalType = "hold"
	SignalStrongBuy   SignalType = "strong_buy"
	SignalStrongSell  SignalType = "strong_sell"
	SignalBreakoutUp  SignalType = "breakout_up"
	SignalBreakoutDown SignalType = "breakout_down"
)

// SignalStrength represents the strength of a signal (0-100)
type SignalStrength int

const (
	StrengthVeryWeak SignalStrength = 20
	StrengthWeak     SignalStrength = 40
	StrengthModerate SignalStrength = 60
	StrengthStrong   SignalStrength = 80
	StrengthVeryStrong SignalStrength = 100
)

// TradingSignal represents a complete trading signal
type TradingSignal struct {
	Type         SignalType     `json:"type"`
	Strength     SignalStrength `json:"strength"`
	Price        float64        `json:"price"`
	Symbol       string         `json:"symbol"`
	Timestamp    int64          `json:"timestamp"`
	Reason       string         `json:"reason"`
	Confidence   float64        `json:"confidence"` // 0-1
	StopLoss     float64        `json:"stop_loss,omitempty"`
	TakeProfit   float64        `json:"take_profit,omitempty"`
}

// SignalGenerator generates trading signals based on indicator values
type SignalGenerator struct {
	thresholds SignalThresholds
}

// SignalThresholds defines thresholds for signal generation
type SignalThresholds struct {
	RSIOverbought     float64 `json:"rsi_overbought"`      // Default: 70
	RSIOversold       float64 `json:"rsi_oversold"`        // Default: 30
	RSINeutral         float64 `json:"rsi_neutral"`         // Default: 50
	MACDPositive      float64 `json:"macd_positive"`       // Default: 0
	ATRMultiplier     float64 `json:"atr_multiplier"`      // Default: 2.0
	VolatilityThreshold float64 `json:"volatility_threshold"` // Default: 0.02 (2%)
	BreakoutThreshold float64 `json:"breakout_threshold"`  // Default: 0.003 (0.3%)
	VolumeMultiplier  float64 `json:"volume_multiplier"`   // Default: 1.5
}

// NewSignalGenerator creates a new signal generator
func NewSignalGenerator(thresholds SignalThresholds) *SignalGenerator {
	// Set defaults
	if thresholds.RSIOverbought == 0 {
		thresholds.RSIOverbought = 70
	}
	if thresholds.RSIOversold == 0 {
		thresholds.RSIOversold = 30
	}
	if thresholds.RSINeutral == 0 {
		thresholds.RSINeutral = 50
	}
	if thresholds.MACDPositive == 0 {
		thresholds.MACDPositive = 0
	}
	if thresholds.ATRMultiplier == 0 {
		thresholds.ATRMultiplier = 2.0
	}
	if thresholds.VolatilityThreshold == 0 {
		thresholds.VolatilityThreshold = 0.02 // 2%
	}
	if thresholds.BreakoutThreshold == 0 {
		thresholds.BreakoutThreshold = 0.003 // 0.3%
	}
	if thresholds.VolumeMultiplier == 0 {
		thresholds.VolumeMultiplier = 1.5
	}

	return &SignalGenerator{
		thresholds: thresholds,
	}
}

// GenerateSignal generates a trading signal based on indicator values
func (sg *SignalGenerator) GenerateSignal(values *IndicatorValues, currentVolume float64) *TradingSignal {
	if values == nil {
		return &TradingSignal{
			Type:   SignalHold,
			Reason: "insufficient_data",
		}
	}

	signals := make([]*TradingSignal, 0)

	// RSI signals
	rsiSignal := sg.generateRSISignal(values)
	if rsiSignal != nil {
		signals = append(signals, rsiSignal)
	}

	// MACD signals
	macdSignal := sg.generateMACDSignal(values)
	if macdSignal != nil {
		signals = append(signals, macdSignal)
	}

	// Bollinger Bands signals
	bbSignal := sg.generateBollingerSignal(values)
	if bbSignal != nil {
		signals = append(signals, bbSignal)
	}

	// Moving Average signals
	maSignal := sg.generateMovingAverageSignal(values)
	if maSignal != nil {
		signals = append(signals, maSignal)
	}

	// Volume confirmation
	volumeConfirmed := sg.confirmWithVolume(currentVolume, values.VolumeSMA)

	// Combine signals
	combinedSignal := sg.combineSignals(signals, volumeConfirmed, values)
	combinedSignal.Symbol = values.Symbol
	combinedSignal.Price = values.CurrentPrice

	return combinedSignal
}

// generateRSISignal generates signal based on RSI
func (sg *SignalGenerator) generateRSISignal(values *IndicatorValues) *TradingSignal {
	if values.RSI == 0 {
		return nil
	}

	if values.RSI >= sg.thresholds.RSIOverbought {
		return &TradingSignal{
			Type:     SignalSell,
			Strength: StrengthStrong,
			Reason:   "rsi_overbought",
		}
	}

	if values.RSI <= sg.thresholds.RSIOversold {
		return &TradingSignal{
			Type:     SignalBuy,
			Strength: StrengthStrong,
			Reason:   "rsi_oversold",
		}
	}

	if values.RSI > sg.thresholds.RSINeutral {
		return &TradingSignal{
			Type:     SignalHold,
			Strength: StrengthWeak,
			Reason:   "rsi_bullish",
		}
	}

	return &TradingSignal{
		Type:     SignalHold,
		Strength: StrengthWeak,
		Reason:   "rsi_bearish",
	}
}

// generateMACDSignal generates signal based on MACD
func (sg *SignalGenerator) generateMACDSignal(values *IndicatorValues) *TradingSignal {
	if values.MACD == 0 || values.MACDSignal == 0 {
		return nil
	}

	// MACD crossover
	if values.MACD > values.MACDSignal && values.MACDHist > 0 {
		return &TradingSignal{
			Type:     SignalBuy,
			Strength: StrengthModerate,
			Reason:   "macd_bullish_crossover",
		}
	}

	if values.MACD < values.MACDSignal && values.MACDHist < 0 {
		return &TradingSignal{
			Type:     SignalSell,
			Strength: StrengthModerate,
			Reason:   "macd_bearish_crossover",
		}
	}

	return nil
}

// generateBollingerSignal generates signal based on Bollinger Bands
func (sg *SignalGenerator) generateBollingerSignal(values *IndicatorValues) *TradingSignal {
	if values.BollingerUpper == 0 || values.BollingerLower == 0 {
		return nil
	}

	if values.CurrentPrice > values.BollingerUpper {
		return &TradingSignal{
			Type:     SignalBreakoutUp,
			Strength: StrengthStrong,
			Reason:   "price_above_upper_bb",
		}
	}

	if values.CurrentPrice < values.BollingerLower {
		return &TradingSignal{
			Type:     SignalBreakoutDown,
			Strength: StrengthStrong,
			Reason:   "price_below_lower_bb",
		}
	}

	// Price returning to middle band
	if values.CurrentPrice > values.BollingerMiddle {
		return &TradingSignal{
			Type:     SignalHold,
			Strength: StrengthWeak,
			Reason:   "price_above_middle_bb",
		}
	}

	return &TradingSignal{
		Type:     SignalHold,
		Strength: StrengthWeak,
		Reason:   "price_below_middle_bb",
	}
}

// generateMovingAverageSignal generates signal based on moving averages
func (sg *SignalGenerator) generateMovingAverageSignal(values *IndicatorValues) *TradingSignal {
	if values.SMA == 0 || values.EMA == 0 {
		return nil
	}

	// EMA above SMA (bullish)
	if values.EMA > values.SMA && values.CurrentPrice > values.EMA {
		return &TradingSignal{
			Type:     SignalBuy,
			Strength: StrengthModerate,
			Reason:   "price_above_ema_sma",
		}
	}

	// EMA below SMA (bearish)
	if values.EMA < values.SMA && values.CurrentPrice < values.EMA {
		return &TradingSignal{
			Type:     SignalSell,
			Strength: StrengthModerate,
			Reason:   "price_below_ema_sma",
		}
	}

	return nil
}

// confirmWithVolume confirms signals with volume analysis
func (sg *SignalGenerator) confirmWithVolume(currentVolume, volumeSMA float64) bool {
	if volumeSMA == 0 {
		return false
	}
	return currentVolume > volumeSMA*sg.thresholds.VolumeMultiplier
}

// combineSignals combines multiple signals into one
func (sg *SignalGenerator) combineSignals(signals []*TradingSignal, volumeConfirmed bool, values *IndicatorValues) *TradingSignal {
	if len(signals) == 0 {
		return &TradingSignal{
			Type:   SignalHold,
			Reason: "no_clear_signal",
		}
	}

	// Count signal types
	buyVotes := 0
	sellVotes := 0
	holdVotes := 0
	totalStrength := 0

	for _, signal := range signals {
		switch signal.Type {
		case SignalBuy, SignalStrongBuy, SignalBreakoutUp:
			buyVotes++
		case SignalSell, SignalStrongSell, SignalBreakoutDown:
			sellVotes++
		default:
			holdVotes++
		}
		totalStrength += int(signal.Strength)
	}

	// Determine final signal
	var finalType SignalType
	var finalStrength SignalStrength
	var confidence float64

	if buyVotes > sellVotes && buyVotes > holdVotes {
		finalType = SignalBuy
		confidence = float64(buyVotes) / float64(len(signals))
	} else if sellVotes > buyVotes && sellVotes > holdVotes {
		finalType = SignalSell
		confidence = float64(sellVotes) / float64(len(signals))
	} else {
		finalType = SignalHold
		confidence = 0.5
	}

	// Adjust for volume confirmation
	if volumeConfirmed {
		confidence *= 1.2
		if confidence > 1.0 {
			confidence = 1.0
		}
	} else {
		confidence *= 0.8
	}

	// Calculate final strength
	avgStrength := totalStrength / len(signals)
	avgStrengthInt := SignalStrength(avgStrength)
	if avgStrengthInt >= StrengthStrong {
		finalStrength = StrengthStrong
	} else if avgStrengthInt >= StrengthModerate {
		finalStrength = StrengthModerate
	} else if avgStrengthInt >= StrengthWeak {
		finalStrength = StrengthWeak
	} else {
		finalStrength = StrengthVeryWeak
	}

	// Generate stop loss and take profit
	stopLoss, takeProfit := sg.calculateStopLossTakeProfit(values, finalType)

	return &TradingSignal{
		Type:       finalType,
		Strength:   finalStrength,
		Confidence: confidence,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
		Reason:     "combined_signal",
	}
}

// calculateStopLossTakeProfit calculates stop loss and take profit levels
func (sg *SignalGenerator) calculateStopLossTakeProfit(values *IndicatorValues, signalType SignalType) (float64, float64) {
	if values.ATR == 0 {
		return 0, 0
	}

	stopLossDistance := values.ATR * sg.thresholds.ATRMultiplier
	takeProfitDistance := stopLossDistance * 2 // Risk/reward ratio of 1:2

	switch signalType {
	case SignalBuy, SignalStrongBuy, SignalBreakoutUp:
		stopLoss := values.CurrentPrice - stopLossDistance
		takeProfit := values.CurrentPrice + takeProfitDistance
		return stopLoss, takeProfit

	case SignalSell, SignalStrongSell, SignalBreakoutDown:
		stopLoss := values.CurrentPrice + stopLossDistance
		takeProfit := values.CurrentPrice - takeProfitDistance
		return stopLoss, takeProfit

	default:
		return 0, 0
	}
}

// DetectBreakout detects if price is breaking out of recent range
func (sg *SignalGenerator) DetectBreakout(currentPrice, upperBound, lowerBound float64) SignalType {
	threshold := sg.thresholds.BreakoutThreshold

	rangeSize := upperBound - lowerBound
	breakoutDistance := rangeSize * threshold

	if currentPrice > upperBound+breakoutDistance {
		return SignalBreakoutUp
	}

	if currentPrice < lowerBound-breakoutDistance {
		return SignalBreakoutDown
	}

	return SignalHold
}

// IsPriceStable determines if price is stable (good for grid trading)
func (sg *SignalGenerator) IsPriceStable(values *IndicatorValues) bool {
	if values == nil || values.ATR == 0 || values.CurrentPrice == 0 {
		return false
	}

	// Calculate volatility percentage
	volatility := (values.ATR / values.CurrentPrice)
	return volatility < sg.thresholds.VolatilityThreshold
}

// CalculateGridSpacing calculates optimal grid spacing based on volatility
func (sg *SignalGenerator) CalculateGridSpacing(currentPrice, atr float64) float64 {
	if currentPrice == 0 || atr == 0 {
		return currentPrice * 0.005 // Default 0.5%
	}

	// Use ATR as base, but ensure minimum spacing
	atrSpacing := atr * 0.5
	minSpacing := currentPrice * 0.0025 // 0.25% minimum
	maxSpacing := currentPrice * 0.01    // 1% maximum

	spacing := math.Max(atrSpacing, minSpacing)
	return math.Min(spacing, maxSpacing)
}