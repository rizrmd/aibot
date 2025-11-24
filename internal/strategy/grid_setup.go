package strategy

import (
	"aibot/internal/data"
	"aibot/internal/indicators"
	"aibot/internal/types"
	"fmt"
	"math"
)

// GridSetup analyzes historical data and sets up optimal grid parameters
type GridSetup struct {
	aggregator *data.CandleAggregator
	analyzer   *indicators.TechnicalAnalyzer
	config     GridSetupConfig
}

// GridSetupConfig holds configuration for grid setup
type GridSetupConfig struct {
	// Historical data requirements
	MinHistoryCandles int     `json:"min_history_candles"` // Minimum candles needed for setup
	AnalysisTimeframe data.CandleTimeframe `json:"analysis_timeframe"` // Timeframe for analysis

	// Grid parameters
	DefaultGridLevels int     `json:"default_grid_levels"` // Default number of grid levels
	MinGridLevels     int     `json:"min_grid_levels"`     // Minimum grid levels
	MaxGridLevels     int     `json:"max_grid_levels"`     // Maximum grid levels

	// Volatility and range parameters
	MinPriceRange     float64 `json:"min_price_range"`      // Minimum price range as percentage
	MaxPriceRange     float64 `json:"max_price_range"`      // Maximum price range as percentage
	RangeMultiplier   float64 `json:"range_multiplier"`     // Multiplier for range calculation
	ATRMultiplier     float64 `json:"atr_multiplier"`       // ATR multiplier for bounds

	// Risk parameters
	MaxRiskPerTrade   float64 `json:"max_risk_per_trade"`   // Maximum risk per trade
	DefaultLeverage   float64 `json:"default_leverage"`     // Default leverage

	// Fee parameters (for profitability calculations)
	MakerFee          float64 `json:"maker_fee"`            // 0.02% = 0.0002
	TakerFee          float64 `json:"taker_fee"`            // 0.06% = 0.0006
	MinProfitPerLevel float64 `json:"min_profit_per_level"` // Minimum profit per grid level
}

// GridParameters represents calculated grid parameters
type GridParameters struct {
	Symbol        string    `json:"symbol"`
	UpperBound    float64   `json:"upper_bound"`
	LowerBound    float64   `json:"lower_bound"`
	GridSpacing   float64   `json:"grid_spacing"`
	GridLevels    int       `json:"grid_levels"`
	TotalQuantity float64   `json:"total_quantity"`
	PositionSize  float64   `json:"position_size"`
	SetupTime     int64     `json:"setup_time"`
	// Analysis results
	CurrentPrice  float64   `json:"current_price"`
	RecentHigh    float64   `json:"recent_high"`
	RecentLow     float64   `json:"recent_low"`
	ATR           float64   `json:"atr"`
	Volatility    float64   `json:"volatility"`
	Trend         string    `json:"trend"`
	Confidence    float64   `json:"confidence"`
	SetupReason   string    `json:"setup_reason"`
}

// NewGridSetup creates a new grid setup instance
func NewGridSetup(aggregator *data.CandleAggregator, analyzer *indicators.TechnicalAnalyzer, config GridSetupConfig) *GridSetup {
	// Set defaults
	if config.MinHistoryCandles == 0 {
		config.MinHistoryCandles = 100 // Minimum 100 candles
	}
	if config.AnalysisTimeframe == "" {
		config.AnalysisTimeframe = data.Timeframe3s
	}
	if config.DefaultGridLevels == 0 {
		config.DefaultGridLevels = 20
	}
	if config.MinGridLevels == 0 {
		config.MinGridLevels = 10
	}
	if config.MaxGridLevels == 0 {
		config.MaxGridLevels = 30
	}
	if config.MinPriceRange == 0 {
		config.MinPriceRange = 0.03 // 3%
	}
	if config.MaxPriceRange == 0 {
		config.MaxPriceRange = 0.10 // 10%
	}
	if config.RangeMultiplier == 0 {
		config.RangeMultiplier = 1.2
	}
	if config.ATRMultiplier == 0 {
		config.ATRMultiplier = 2.0
	}
	if config.MakerFee == 0 {
		config.MakerFee = 0.0002 // 0.02%
	}
	if config.TakerFee == 0 {
		config.TakerFee = 0.0006 // 0.06%
	}
	if config.MinProfitPerLevel == 0 {
		config.MinProfitPerLevel = 0.0015 // 0.15%
	}

	return &GridSetup{
		aggregator: aggregator,
		analyzer:   analyzer,
		config:     config,
	}
}

// AnalyzeAndSetup analyzes historical data and returns optimal grid parameters
func (gs *GridSetup) AnalyzeAndSetup(symbol string, accountBalance float64) (*GridParameters, error) {
	// Get historical candles
	candles := gs.aggregator.GetCandles(symbol, gs.config.AnalysisTimeframe, gs.config.MinHistoryCandles)
	if len(candles) < gs.config.MinHistoryCandles {
		return nil, fmt.Errorf("insufficient historical data: got %d, need %d", len(candles), gs.config.MinHistoryCandles)
	}

	// Add candles to analyzer for indicator calculation
	gs.analyzer.AddCandles(candles)

	// Get indicator values
	indicatorValues := gs.analyzer.GetIndicatorValues(symbol)
	if indicatorValues == nil {
		return nil, fmt.Errorf("failed to get indicator values for %s", symbol)
	}

	// Analyze market conditions
	marketAnalysis := gs.analyzeMarketConditions(candles, indicatorValues)

	// Calculate grid bounds
	upperBound, lowerBound := gs.calculateGridBounds(indicatorValues, marketAnalysis)

	// Calculate optimal grid spacing and levels
	gridSpacing, gridLevels := gs.calculateOptimalSpacingAndLevels(upperBound, lowerBound, indicatorValues)

	// Calculate position sizing
	positionSize, totalQuantity := gs.calculatePositionSize(upperBound, lowerBound, gridLevels, accountBalance)

	// Validate profitability
	if !gs.validateProfitability(gridSpacing, gridLevels, positionSize) {
		return nil, fmt.Errorf("grid setup not profitable for %s with current conditions", symbol)
	}

	return &GridParameters{
		Symbol:        symbol,
		UpperBound:    upperBound,
		LowerBound:    lowerBound,
		GridSpacing:   gridSpacing,
		GridLevels:    gridLevels,
		TotalQuantity: totalQuantity,
		PositionSize:  positionSize,
		SetupTime:     candles[len(candles)-1].Timestamp.Unix(),
		CurrentPrice:  indicatorValues.CurrentPrice,
		RecentHigh:    marketAnalysis.recentHigh,
		RecentLow:     marketAnalysis.recentLow,
		ATR:           indicatorValues.ATR,
		Volatility:    marketAnalysis.volatility,
		Trend:         marketAnalysis.trend,
		Confidence:    marketAnalysis.confidence,
		SetupReason:   marketAnalysis.setupReason,
	}, nil
}

// MarketAnalysis represents market condition analysis
type MarketAnalysis struct {
	recentHigh     float64
	recentLow      float64
	volatility     float64
	trend          string
	confidence     float64
	setupReason    string
	volumeProfile  float64
	priceMomentum  float64
}

// analyzeMarketConditions analyzes market conditions for grid setup
func (gs *GridSetup) analyzeMarketConditions(candles []types.OHLCV, indicatorValues *indicators.IndicatorValues) MarketAnalysis {
	// Calculate recent high and low (last 50% of candles)
	halfCandles := len(candles) / 2
	recentCandles := candles[halfCandles:]

	var recentHigh, recentLow float64
	for _, candle := range recentCandles {
		if recentHigh == 0 || candle.High > recentHigh {
			recentHigh = candle.High
		}
		if recentLow == 0 || candle.Low < recentLow {
			recentLow = candle.Low
		}
	}

	// Calculate volatility (ATR as percentage of price)
	var volatility float64
	if indicatorValues.CurrentPrice > 0 && indicatorValues.ATR > 0 {
		volatility = (indicatorValues.ATR / indicatorValues.CurrentPrice) * 100
	}

	// Determine trend
	trend := "sideways"
	confidence := 0.5
	setupReason := "normal_market"

	if indicatorValues.SMA > 0 && indicatorValues.EMA > 0 {
		if indicatorValues.EMA > indicatorValues.SMA && indicatorValues.CurrentPrice > indicatorValues.EMA {
			trend = "bullish"
			confidence = 0.7
			setupReason = "uptrend_detected"
		} else if indicatorValues.EMA < indicatorValues.SMA && indicatorValues.CurrentPrice < indicatorValues.EMA {
			trend = "bearish"
			confidence = 0.7
			setupReason = "downtrend_detected"
		}
	}

	// Adjust confidence based on volatility
	if volatility > 3.0 { // High volatility
		confidence *= 0.8
		setupReason += "_high_volatility"
	} else if volatility < 1.0 { // Low volatility
		confidence *= 1.2
		setupReason += "_low_volatility"
	}

	// Calculate volume profile (average volume)
	var totalVolume float64
	for _, candle := range candles {
		totalVolume += candle.Volume
	}
	volumeProfile := totalVolume / float64(len(candles))

	// Calculate price momentum
	var priceMomentum float64
	if len(candles) >= 10 {
		oldPrice := candles[len(candles)-10].Close
		newPrice := candles[len(candles)-1].Close
		if oldPrice > 0 {
			priceMomentum = ((newPrice - oldPrice) / oldPrice) * 100
		}
	}

	return MarketAnalysis{
		recentHigh:    recentHigh,
		recentLow:     recentLow,
		volatility:    volatility,
		trend:         trend,
		confidence:    confidence,
		setupReason:   setupReason,
		volumeProfile: volumeProfile,
		priceMomentum: priceMomentum,
	}
}

// calculateGridBounds calculates optimal upper and lower bounds for the grid
func (gs *GridSetup) calculateGridBounds(indicatorValues *indicators.IndicatorValues, analysis MarketAnalysis) (float64, float64) {
	currentPrice := indicatorValues.CurrentPrice

	// Method 1: Use recent high/low with multiplier
	rangeFromRecent := (analysis.recentHigh - analysis.recentLow) * gs.config.RangeMultiplier
	upperFromRecent := currentPrice + (rangeFromRecent * 0.6)
	lowerFromRecent := currentPrice - (rangeFromRecent * 0.4)

	// Method 2: Use ATR-based bounds
	atrUpper := currentPrice + (indicatorValues.ATR * gs.config.ATRMultiplier)
	atrLower := currentPrice - (indicatorValues.ATR * gs.config.ATRMultiplier)

	// Method 3: Use Bollinger Bands
	var bbUpper, bbLower float64
	if indicatorValues.BollingerUpper > 0 && indicatorValues.BollingerLower > 0 {
		bbUpper = indicatorValues.BollingerUpper
		bbLower = indicatorValues.BollingerLower
	} else {
		bbUpper = atrUpper
		bbLower = atrLower
	}

	// Choose best method based on market conditions
	var upperBound, lowerBound float64

	switch analysis.trend {
	case "bullish":
		// Wider upper bound for uptrends
		upperBound = math.Max(upperFromRecent, math.Max(atrUpper, bbUpper))
		lowerBound = math.Min(lowerFromRecent, math.Min(atrLower, bbLower))
	case "bearish":
		// Wider lower bound for downtrends
		upperBound = math.Max(upperFromRecent, math.Max(atrUpper, bbUpper))
		lowerBound = math.Min(lowerFromRecent, math.Min(atrLower, bbLower))
	default:
		// Balanced bounds for sideways markets
		upperBound = (upperFromRecent + atrUpper + bbUpper) / 3
		lowerBound = (lowerFromRecent + atrLower + bbLower) / 3
	}

	// Ensure minimum price range
	minRange := currentPrice * gs.config.MinPriceRange
	maxRange := currentPrice * gs.config.MaxPriceRange

	rangeSize := upperBound - lowerBound
	if rangeSize < minRange {
		center := (upperBound + lowerBound) / 2
		upperBound = center + minRange/2
		lowerBound = center - minRange/2
	} else if rangeSize > maxRange {
		center := (upperBound + lowerBound) / 2
		upperBound = center + maxRange/2
		lowerBound = center - maxRange/2
	}

	// Ensure bounds are positive
	if lowerBound < 0 {
		lowerBound = 0.01
	}

	return upperBound, lowerBound
}

// calculateOptimalSpacingAndLevels calculates optimal grid spacing and number of levels
func (gs *GridSetup) calculateOptimalSpacingAndLevels(upperBound, lowerBound float64, indicatorValues *indicators.IndicatorValues) (float64, int) {
	totalRange := upperBound - lowerBound

	// Calculate minimum profitable spacing
	totalFees := gs.config.MakerFee + gs.config.TakerFee
	minProfitableSpacing := totalFees + gs.config.MinProfitPerLevel

	// Calculate volatility-based spacing
	var volBasedSpacing float64
	if indicatorValues.ATR > 0 {
		volBasedSpacing = indicatorValues.ATR * 0.5 // Use half ATR as base spacing
	}

	// Choose larger of minimum profitable or volatility-based spacing
	baseSpacing := math.Max(minProfitableSpacing, volBasedSpacing)

	// Calculate number of levels based on spacing
	gridLevels := int(totalRange / baseSpacing)

	// Adjust to be within min/max levels
	if gridLevels < gs.config.MinGridLevels {
		gridLevels = gs.config.MinGridLevels
		baseSpacing = totalRange / float64(gridLevels)
	} else if gridLevels > gs.config.MaxGridLevels {
		gridLevels = gs.config.MaxGridLevels
		baseSpacing = totalRange / float64(gridLevels)
	}

	// Ensure spacing is still profitable
	if baseSpacing < minProfitableSpacing {
		gridLevels = int(totalRange / minProfitableSpacing)
		if gridLevels < gs.config.MinGridLevels {
			gridLevels = gs.config.MinGridLevels
		}
		baseSpacing = totalRange / float64(gridLevels)
	}

	return baseSpacing, gridLevels
}

// calculatePositionSize calculates optimal position size and total quantity
func (gs *GridSetup) calculatePositionSize(upperBound, lowerBound float64, gridLevels int, accountBalance float64) (float64, float64) {
	// Risk per grid level (default 2% risk per trade if not specified)
	riskPerLevel := accountBalance * 0.02

	// Calculate average grid level price
	avgPrice := (upperBound + lowerBound) / 2

	// Position size per level (amount of currency)
	positionSize := riskPerLevel / avgPrice

	// Total quantity needed for all levels
	totalQuantity := positionSize * float64(gridLevels)

	return positionSize, totalQuantity
}

// validateProfitability checks if the grid setup will be profitable
func (gs *GridSetup) validateProfitability(gridSpacing float64, gridLevels int, positionSize float64) bool {
	// Calculate expected profit per level
	avgPrice := 100.0 // Assumed average price for calculation
	notionalPerLevel := positionSize * avgPrice

	// Profit calculation: grid spacing * notional - fees
	totalFees := (gs.config.MakerFee + gs.config.TakerFee) * notionalPerLevel
	expectedProfit := (gridSpacing * notionalPerLevel) - totalFees

	// Profit must be positive and meet minimum
	return expectedProfit > 0 && (expectedProfit/notionalPerLevel) > gs.config.MinProfitPerLevel
}

// GetRecommendedGridLevels returns recommended grid levels based on volatility
func (gs *GridSetup) GetRecommendedGridLevels(volatility float64) int {
	if volatility > 3.0 { // High volatility
		return gs.config.MinGridLevels // Fewer, wider levels
	} else if volatility > 1.5 { // Medium volatility
		return gs.config.DefaultGridLevels
	} else { // Low volatility
		return gs.config.MaxGridLevels // More, tighter levels
	}
}

// ShouldSetupGrid determines if conditions are suitable for grid setup
func (gs *GridSetup) ShouldSetupGrid(symbol string) (bool, string) {
	candles := gs.aggregator.GetCandles(symbol, gs.config.AnalysisTimeframe, 50)
	if len(candles) < gs.config.MinHistoryCandles {
		return false, "insufficient_data"
	}

	indicatorValues := gs.analyzer.GetIndicatorValues(symbol)
	if indicatorValues == nil {
		return false, "no_indicators"
	}

	// Check if price is stable enough for grid
	if indicatorValues.ATR > 0 && indicatorValues.CurrentPrice > 0 {
		volatility := (indicatorValues.ATR / indicatorValues.CurrentPrice) * 100
		if volatility > 5.0 { // Too volatile
			return false, "high_volatility"
		}
	}

	// Check for extreme trends (RSI)
	if indicatorValues.RSI > 80 || indicatorValues.RSI < 20 {
		return false, "extreme_trend"
	}

	return true, "suitable_conditions"
}