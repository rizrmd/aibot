package strategy

import (
	"aibot/internal/data"
	"aibot/internal/indicators"
	"aibot/internal/types"
	"math"
	"time"
)

// PriceStabilityDetector analyzes market conditions to detect price stability
// after breakouts for optimal timing to return to grid trading
type PriceStabilityDetector struct {
	// Configuration parameters
	StabilityWindow       int     `json:"stability_window"`        // Analysis window (10 candles = 3s)
	VolatilityThreshold   float64 `json:"volatility_threshold"`    // Max volatility for stability (0.5%)
	MomentumThreshold     float64 `json:"momentum_threshold"`      // Max momentum for stability (0.2%)
	PriceConformity       float64 `json:"price_conformity"`        // Min price conformity (80%)
	RangeContraction      float64 `json:"range_contraction"`       // Min range contraction (70%)
	MinStabilityPeriod    int     `json:"min_stability_period"`    // Minimum stable periods (3 consecutive checks)

	// Analysis timeframes
	PrimaryTimeframe      data.CandleTimeframe `json:"primary_timeframe"`    // 3s for primary analysis
	SecondaryTimeframe    data.CandleTimeframe `json:"secondary_timeframe"`  // 15s for trend confirmation

	// State tracking
	stabilityChecks       []StabilityCheck `json:"stability_checks"`
	consecutiveStable    int              `json:"consecutive_stable"`
	lastAnalysisTime     time.Time        `json:"last_analysis_time"`
	isCurrentlyStable    bool             `json:"is_currently_stable"`
	stabilityStartTime   *time.Time       `json:"stability_start_time,omitempty"`

	// Technical analysis
	technicalAnalyzer     *indicators.TechnicalAnalyzer
	candleAggregator      *data.CandleAggregator

	// Performance tracking
	totalChecks           int     `json:"total_checks"`
	stablePeriods         int     `json:"stable_periods"`
	falseStablePeriods    int     `json:"false_stable_periods"`
	avgStabilityDuration  time.Duration `json:"avg_stability_duration"`
}

// StabilityCheck represents a single stability analysis result
type StabilityCheck struct {
	Timestamp             time.Time `json:"timestamp"`
	IsStable              bool      `json:"is_stable"`
	Confidence            float64   `json:"confidence"`
	VolatilityScore       float64   `json:"volatility_score"`
	MomentumScore         float64   `json:"momentum_score"`
	RangeContractionScore float64   `json:"range_contraction_score"`
	PriceConformityScore  float64   `json:"price_conformity_score"`
	OverallScore          float64   `json:"overall_score"`
	Reason                string    `json:"reason"`
	Symbol                string    `json:"symbol"`
}

// StabilitySignal represents a detected stability condition
type StabilitySignal struct {
	IsStable          bool      `json:"is_stable"`
	Confidence        float64   `json:"confidence"`
	Duration          time.Duration `json:"duration"`
	RecommendedAction string    `json:"recommended_action"`
	Reason            string    `json:"reason"`
	Timestamp         time.Time `json:"timestamp"`
	Symbol            string    `json:"symbol"`
	PriceLevel        float64   `json:"price_level"`
	Volatility        float64   `json:"volatility"`
	RiskLevel         string    `json:"risk_level"`
}

// StabilityConfig holds configuration for stability detection
type StabilityConfig struct {
	AnalysisWindow       int     `json:"analysis_window"`        // 10 candles
	VolatilityThreshold  float64 `json:"volatility_threshold"`   // 0.5%
	MomentumThreshold    float64 `json:"momentum_threshold"`     // 0.2%
	PriceConformity      float64 `json:"price_conformity"`       // 80%
	RangeContraction     float64 `json:"range_contraction"`      // 70%
	MinStabilityPeriods  int     `json:"min_stability_periods"`  // 3 consecutive checks
	PrimaryTimeframe     string  `json:"primary_timeframe"`      // "3s"
	SecondaryTimeframe   string  `json:"secondary_timeframe"`    // "15s"
}

// NewPriceStabilityDetector creates a new price stability detector
func NewPriceStabilityDetector(
	config StabilityConfig,
	analyzer *indicators.TechnicalAnalyzer,
	aggregator *data.CandleAggregator,
) *PriceStabilityDetector {

	// Set defaults
	if config.AnalysisWindow == 0 {
		config.AnalysisWindow = 10 // 10 candles = 3s at 300ms intervals
	}
	if config.VolatilityThreshold == 0 {
		config.VolatilityThreshold = 0.005 // 0.5%
	}
	if config.MomentumThreshold == 0 {
		config.MomentumThreshold = 0.002 // 0.2%
	}
	if config.PriceConformity == 0 {
		config.PriceConformity = 0.8 // 80%
	}
	if config.RangeContraction == 0 {
		config.RangeContraction = 0.7 // 70%
	}
	if config.MinStabilityPeriods == 0 {
		config.MinStabilityPeriods = 3
	}
	if config.PrimaryTimeframe == "" {
		config.PrimaryTimeframe = "3s"
	}
	if config.SecondaryTimeframe == "" {
		config.SecondaryTimeframe = "15s"
	}

	return &PriceStabilityDetector{
		StabilityWindow:      config.AnalysisWindow,
		VolatilityThreshold:  config.VolatilityThreshold,
		MomentumThreshold:    config.MomentumThreshold,
		PriceConformity:      config.PriceConformity,
		RangeContraction:     config.RangeContraction,
		MinStabilityPeriod:   config.MinStabilityPeriods,
		PrimaryTimeframe:     parseTimeframe(config.PrimaryTimeframe),
		SecondaryTimeframe:   parseTimeframe(config.SecondaryTimeframe),
		stabilityChecks:      make([]StabilityCheck, 0),
		technicalAnalyzer:    analyzer,
		candleAggregator:     aggregator,
	}
}

// AnalyzeStability performs comprehensive stability analysis
func (ps *PriceStabilityDetector) AnalyzeStability(symbol string, currentPrice float64) *StabilitySignal {
	// Get candle data for analysis
	primaryCandles := ps.candleAggregator.GetCandles(symbol, ps.PrimaryTimeframe, ps.StabilityWindow)
	secondaryCandles := ps.candleAggregator.GetCandles(symbol, ps.SecondaryTimeframe, ps.StabilityWindow/2)

	if len(primaryCandles) < ps.StabilityWindow/2 {
		return ps.createInsufficientDataSignal(symbol, currentPrice)
	}

	// Perform stability analysis
	check := ps.performStabilityCheck(symbol, currentPrice, primaryCandles, secondaryCandles)
	ps.addStabilityCheck(check)

	// Update stability state
	ps.updateStabilityState(check)

	// Generate stability signal
	signal := ps.generateStabilitySignal(check, currentPrice)

	// Update tracking
	ps.lastAnalysisTime = time.Now()
	ps.totalChecks++

	return signal
}

// performStabilityCheck executes the detailed stability analysis
func (ps *PriceStabilityDetector) performStabilityCheck(
	symbol string,
	currentPrice float64,
	primaryCandles, secondaryCandles []types.OHLCV,
) StabilityCheck {

	check := StabilityCheck{
		Timestamp: time.Now(),
		Symbol:    symbol,
	}

	// 1. Volatility Analysis
	volatilityScore, volatility := ps.analyzeVolatility(primaryCandles)
	check.VolatilityScore = volatilityScore

	// 2. Momentum Analysis
	momentumScore, momentum := ps.analyzeMomentum(primaryCandles)
	check.MomentumScore = momentumScore

	// 3. Range Contraction Analysis
	rangeContractionScore := ps.analyzeRangeContraction(primaryCandles)
	check.RangeContractionScore = rangeContractionScore

	// 4. Price Conformity Analysis
	priceConformityScore := ps.analyzePriceConformity(primaryCandles)
	check.PriceConformityScore = priceConformityScore

	// 5. Trend Consistency Analysis (secondary timeframe)
	trendConsistencyScore := ps.analyzeTrendConsistency(secondaryCandles)

	// Calculate overall confidence score
	check.OverallScore = ps.calculateOverallConfidence(
		volatilityScore,
		momentumScore,
		rangeContractionScore,
		priceConformityScore,
		trendConsistencyScore,
	)

	// Determine if stable
	check.IsStable = ps.isStable(check.OverallScore)
	check.Confidence = check.OverallScore

	// Generate reason
	check.Reason = ps.generateStabilityReason(check, volatility, momentum)

	return check
}

// analyzeVolatility measures price volatility against threshold
func (ps *PriceStabilityDetector) analyzeVolatility(candles []types.OHLCV) (float64, float64) {
	if len(candles) < 2 {
		return 0.0, 0.0
	}

	// Calculate price changes
	priceChanges := make([]float64, len(candles)-1)
	for i := 1; i < len(candles); i++ {
		change := (candles[i].Close - candles[i-1].Close) / candles[i-1].Close
		priceChanges[i-1] = math.Abs(change)
	}

	// Calculate average volatility
	avgVolatility := calculateAverage(priceChanges)

	// Calculate volatility score (lower volatility = higher score)
	score := math.Max(0, 1.0-(avgVolatility/ps.VolatilityThreshold))

	return score, avgVolatility
}

// analyzeMomentum measures price momentum against threshold
func (ps *PriceStabilityDetector) analyzeMomentum(candles []types.OHLCV) (float64, float64) {
	if len(candles) < 3 {
		return 0.0, 0.0
	}

	// Calculate momentum over different periods
	shortMomentum := ps.calculateMomentum(candles, 3)   // Last 3 candles
	longMomentum := ps.calculateMomentum(candles, len(candles)/2) // Half window

	// Average momentum
	avgMomentum := (math.Abs(shortMomentum) + math.Abs(longMomentum)) / 2

	// Momentum score (lower momentum = higher score)
	score := math.Max(0, 1.0-(avgMomentum/ps.MomentumThreshold))

	return score, avgMomentum
}

// analyzeRangeContraction measures if price range is contracting
func (ps *PriceStabilityDetector) analyzeRangeContraction(candles []types.OHLCV) float64 {
	if len(candles) < 4 {
		return 0.0
	}

	// Calculate range for first half and second half
	midpoint := len(candles) / 2

	firstHalfRange := ps.calculatePriceRange(candles[:midpoint])
	secondHalfRange := ps.calculatePriceRange(candles[midpoint:])

	if firstHalfRange == 0 {
		return 0.0
	}

	// Range contraction ratio (smaller ratio = more contraction)
	contractionRatio := secondHalfRange / firstHalfRange
	targetContraction := 1.0 - ps.RangeContraction

	// Score based on how much range has contracted
	if contractionRatio <= targetContraction {
		return 1.0 // Full score for sufficient contraction
	}

	// Partial score for partial contraction
	score := math.Max(0, 1.0-((contractionRatio-targetContraction)/(1.0-targetContraction)))

	return score
}

// analyzePriceConformity measures how well prices conform to a stable pattern
func (ps *PriceStabilityDetector) analyzePriceConformity(candles []types.OHLCV) float64 {
	if len(candles) < 5 {
		return 0.0
	}

	// Calculate linear regression of prices
	prices := make([]float64, len(candles))
	for i, candle := range candles {
		prices[i] = candle.Close
	}

	// Calculate standard deviation from mean
	mean := calculateAverage(prices)
	sumSquares := 0.0
	for _, price := range prices {
		deviation := price - mean
		sumSquares += deviation * deviation
	}
	stdDev := math.Sqrt(sumSquares / float64(len(prices)))

	// Coefficient of variation (lower = more conforming)
	if mean == 0 {
		return 0.0
	}
	coefficientOfVariation := stdDev / mean

	// Target coefficient of variation for stability
	targetCV := 0.01 // 1% variation target

	// Score based on conformity (lower CV = higher score)
	if coefficientOfVariation <= targetCV {
		return 1.0
	}

	score := math.Max(0, 1.0-(coefficientOfVariation/targetCV))
	return score
}

// analyzeTrendConsistency measures trend consistency across timeframes
func (ps *PriceStabilityDetector) analyzeTrendConsistency(candles []types.OHLCV) float64 {
	if len(candles) < 4 {
		return 0.5 // Neutral score for insufficient data
	}

	// Calculate short and long term trends
	shortTrend := ps.calculateTrendDirection(candles[len(candles)/2:])
	longTrend := ps.calculateTrendDirection(candles)

	// Trend consistency (both pointing in same direction)
	if (shortTrend > 0 && longTrend > 0) || (shortTrend < 0 && longTrend < 0) {
		return 1.0 // Consistent trends
	} else if shortTrend == 0 || longTrend == 0 {
		return 0.5 // One trend is flat
	} else {
		return 0.0 // Conflicting trends
	}
}

// calculateMomentum calculates price momentum over specified period
func (ps *PriceStabilityDetector) calculateMomentum(candles []types.OHLCV, period int) float64 {
	if len(candles) < period || period < 2 {
		return 0.0
	}

	startPrice := candles[len(candles)-period].Close
	endPrice := candles[len(candles)-1].Close

	return (endPrice - startPrice) / startPrice
}

// calculatePriceRange calculates the price range of candles
func (ps *PriceStabilityDetector) calculatePriceRange(candles []types.OHLCV) float64 {
	if len(candles) == 0 {
		return 0.0
	}

	minPrice := candles[0].Low
	maxPrice := candles[0].High

	for _, candle := range candles {
		if candle.Low < minPrice {
			minPrice = candle.Low
		}
		if candle.High > maxPrice {
			maxPrice = candle.High
		}
	}

	if minPrice == 0 {
		return 0.0
	}

	return (maxPrice - minPrice) / minPrice
}

// calculateTrendDirection calculates trend direction (-1, 0, 1)
func (ps *PriceStabilityDetector) calculateTrendDirection(candles []types.OHLCV) float64 {
	if len(candles) < 2 {
		return 0.0
	}

	startPrice := candles[0].Close
	endPrice := candles[len(candles)-1].Close
	change := (endPrice - startPrice) / startPrice

	threshold := 0.001 // 0.1% threshold for trend determination

	if change > threshold {
		return 1.0 // Uptrend
	} else if change < -threshold {
		return -1.0 // Downtrend
	} else {
		return 0.0 // Sideways
	}
}

// calculateOverallConfidence combines all analysis scores
func (ps *PriceStabilityDetector) calculateOverallConfidence(
	volatility, momentum, rangeContraction, priceConformity, trendConsistency float64,
) float64 {
	// Weight the different factors
	weights := map[string]float64{
		"volatility":        0.3, // Most important factor
		"momentum":          0.25,
		"range_contraction": 0.2,
		"price_conformity":  0.15,
		"trend_consistency": 0.1,
	}

	overall := volatility*weights["volatility"] +
		momentum*weights["momentum"] +
		rangeContraction*weights["range_contraction"] +
		priceConformity*weights["price_conformity"] +
		trendConsistency*weights["trend_consistency"]

	return math.Min(1.0, math.Max(0.0, overall))
}

// isStable determines if market conditions meet stability criteria
func (ps *PriceStabilityDetector) isStable(overallScore float64) bool {
	// Minimum overall score threshold
	minOverallScore := 0.7 // 70% overall confidence

	return overallScore >= minOverallScore
}

// generateStabilityReason creates human-readable stability reason
func (ps *PriceStabilityDetector) generateStabilityReason(check StabilityCheck, volatility, momentum float64) string {
	if !check.IsStable {
		reasons := []string{}
		if check.VolatilityScore < 0.5 {
			reasons = append(reasons, "high volatility")
		}
		if check.MomentumScore < 0.5 {
			reasons = append(reasons, "strong momentum")
		}
		if check.RangeContractionScore < 0.5 {
			reasons = append(reasons, "expanding range")
		}
		if check.PriceConformityScore < 0.5 {
			reasons = append(reasons, "price inconsistency")
		}

		if len(reasons) > 0 {
			return "Unstable due to: " + joinStrings(reasons, ", ")
		}
		return "Unstable market conditions"
	}

	return "Stable market conditions detected"
}

// addStabilityCheck adds a stability check to history
func (ps *PriceStabilityDetector) addStabilityCheck(check StabilityCheck) {
	ps.stabilityChecks = append(ps.stabilityChecks, check)

	// Keep only last 50 checks
	if len(ps.stabilityChecks) > 50 {
		ps.stabilityChecks = ps.stabilityChecks[1:]
	}
}

// updateStabilityState updates the stability state tracking
func (ps *PriceStabilityDetector) updateStabilityState(check StabilityCheck) {
	if check.IsStable {
		if !ps.isCurrentlyStable {
			// Transition to stable
			ps.isCurrentlyStable = true
			now := time.Now()
			ps.stabilityStartTime = &now
			ps.consecutiveStable = 1
		} else {
			// Continue stable period
			ps.consecutiveStable++
		}
		ps.stablePeriods++
	} else {
		if ps.isCurrentlyStable {
			// Transition from stable
			ps.isCurrentlyStable = false
			if ps.stabilityStartTime != nil {
				duration := time.Since(*ps.stabilityStartTime)
				ps.updateAverageStabilityDuration(duration)
			}
			ps.consecutiveStable = 0
			ps.falseStablePeriods++
		}
	}
}

// updateAverageStabilityDuration updates the average stability duration
func (ps *PriceStabilityDetector) updateAverageStabilityDuration(duration time.Duration) {
	if ps.avgStabilityDuration == 0 {
		ps.avgStabilityDuration = duration
	} else {
		// Simple moving average
		ps.avgStabilityDuration = time.Duration((int64(ps.avgStabilityDuration) + int64(duration)) / 2)
	}
}

// generateStabilitySignal creates the final stability signal
func (ps *PriceStabilityDetector) generateStabilitySignal(check StabilityCheck, currentPrice float64) *StabilitySignal {
	signal := &StabilitySignal{
		IsStable:   check.IsStable,
		Confidence: check.Confidence,
		Timestamp:  check.Timestamp,
		Symbol:     check.Symbol,
		PriceLevel: currentPrice,
		Reason:     check.Reason,
	}

	// Calculate duration if currently stable
	if ps.isCurrentlyStable && ps.stabilityStartTime != nil {
		signal.Duration = time.Since(*ps.stabilityStartTime)
	}

	// Determine recommended action
	if check.IsStable && ps.consecutiveStable >= ps.MinStabilityPeriod {
		signal.RecommendedAction = "Return to grid trading"
	} else if check.IsStable {
		signal.RecommendedAction = "Monitor for continued stability"
	} else {
		signal.RecommendedAction = "Maintain breakout position management"
	}

	// Determine risk level
	signal.RiskLevel = ps.determineRiskLevel(check, currentPrice)

	return signal
}

// determineRiskLevel determines the current risk level
func (ps *PriceStabilityDetector) determineRiskLevel(check StabilityCheck, currentPrice float64) string {
	if check.IsStable {
		return "low"
	}

	if check.VolatilityScore < 0.3 || check.MomentumScore < 0.3 {
		return "high"
	}

	return "medium"
}

// createInsufficientDataSignal creates a signal when insufficient data is available
func (ps *PriceStabilityDetector) createInsufficientDataSignal(symbol string, currentPrice float64) *StabilitySignal {
	return &StabilitySignal{
		IsStable:          false,
		Confidence:        0.0,
		Timestamp:         time.Now(),
		Symbol:            symbol,
		PriceLevel:        currentPrice,
		Reason:            "Insufficient data for stability analysis",
		RecommendedAction: "Wait for more data",
		RiskLevel:         "unknown",
	}
}

// GetStabilityStats returns stability detection statistics
func (ps *PriceStabilityDetector) GetStabilityStats() map[string]interface{} {
	stabilityRate := float64(0)
	if ps.totalChecks > 0 {
		stabilityRate = float64(ps.stablePeriods) / float64(ps.totalChecks) * 100
	}

	return map[string]interface{}{
		"total_checks":           ps.totalChecks,
		"stable_periods":         ps.stablePeriods,
		"false_stable_periods":   ps.falseStablePeriods,
		"stability_rate":         stabilityRate,
		"consecutive_stable":     ps.consecutiveStable,
		"is_currently_stable":    ps.isCurrentlyStable,
		"avg_stability_duration": ps.avgStabilityDuration.String(),
		"recent_checks":          len(ps.stabilityChecks),
	}
}

// Reset resets the stability detector state
func (ps *PriceStabilityDetector) Reset() {
	ps.stabilityChecks = make([]StabilityCheck, 0)
	ps.consecutiveStable = 0
	ps.lastAnalysisTime = time.Time{}
	ps.isCurrentlyStable = false
	ps.stabilityStartTime = nil
	ps.totalChecks = 0
	ps.stablePeriods = 0
	ps.falseStablePeriods = 0
	ps.avgStabilityDuration = 0
}

// Helper functions
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// parseTimeframe converts string timeframe to CandleTimeframe
func parseTimeframe(tf string) data.CandleTimeframe {
	switch tf {
	case "1s":
		return data.Timeframe1s
	case "3s":
		return data.Timeframe3s
	case "15s":
		return data.Timeframe15s
	case "30s":
		return data.Timeframe30s
	case "1m":
		return data.Timeframe1m
	default:
		return data.Timeframe3s // Default to 3s
	}
}