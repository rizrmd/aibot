package strategy

import (
	"math"
)

// GridCalculator calculates fee-aware grid parameters for optimal profitability
type GridCalculator struct {
	// Fee rates (in percentage, e.g., 0.0002 for 0.02%)
	MakerFee    float64 `json:"maker_fee"`    // 0.02%
	TakerFee    float64 `json:"taker_fee"`    // 0.06%

	// Profit requirements
	MinProfitPerLevel float64 `json:"min_profit_per_level"` // Minimum profit per level (0.15%)
	TargetROI        float64 `json:"target_roi"`           // Target ROI per day (1.0%)

	// Grid constraints
	MinGridSpacing    float64 `json:"min_grid_spacing"`    // Minimum grid spacing (0.25%)
	MaxGridSpacing    float64 `json:"max_grid_spacing"`    // Maximum grid spacing (1.0%)
	MinGridLevels     int     `json:"min_grid_levels"`     // Minimum grid levels (10)
	MaxGridLevels     int     `json:"max_grid_levels"`     // Maximum grid levels (30)

	// Risk management
	MaxRiskPerTrade   float64 `json:"max_risk_per_trade"`   // Maximum risk per trade (2.0%)
	PositionSizeRatio float64 `json:"position_size_ratio"` // Position size as % of capital
}

// GridCalculationResult contains optimized grid parameters
type GridCalculationResult struct {
	GridSpacing      float64 `json:"grid_spacing"`
	GridLevels       int     `json:"grid_levels"`
	TotalRange       float64 `json:"total_range"`
	UpperBound       float64 `json:"upper_bound"`
	LowerBound       float64 `json:"lower_bound"`
	PositionSize     float64 `json:"position_size"`
	ExpectedProfit   float64 `json:"expected_profit"`
	ProfitMargin     float64 `json:"profit_margin"`
	BreakevenLevels  int     `json:"breakeven_levels"`
	WinRateRequired  float64 `json:"win_rate_required"`
	OptimizationScore float64 `json:"optimization_score"`
	SetupReason      string  `json:"setup_reason"`
}

// FeeAnalysis contains detailed fee analysis for grid strategy
type FeeAnalysis struct {
	TotalFeesPerRoundTrip float64 `json:"total_fees_per_round_trip"` // Maker + Taker
	MakerFeePerLevel      float64 `json:"maker_fee_per_level"`
	TakerFeePerLevel      float64 `json:"taker_fee_per_level"`
	MinProfitableSpread   float64 `json:"min_profitable_spread"`
	FeeBurden            float64 `json:"fee_burden"` // Fees as % of spread
}

// NewGridCalculator creates a new grid calculator with default settings
func NewGridCalculator() *GridCalculator {
	return &GridCalculator{
		MakerFee:           0.0002, // 0.02%
		TakerFee:           0.0006, // 0.06%
		MinProfitPerLevel:  0.0015, // 0.15%
		TargetROI:          0.01,   // 1.0%
		MinGridSpacing:     0.0025, // 0.25%
		MaxGridSpacing:     0.01,   // 1.0%
		MinGridLevels:      10,
		MaxGridLevels:      30,
		MaxRiskPerTrade:    0.02,   // 2.0%
		PositionSizeRatio:  0.05,   // 5% of capital
	}
}

// CalculateOptimalGrid calculates the most profitable grid configuration
func (gc *GridCalculator) CalculateOptimalGrid(
	currentPrice, volatility, accountBalance float64,
	volatilityCategory string, // "low", "medium", "high"
) *GridCalculationResult {

	// Calculate fee analysis
	feeAnalysis := gc.calculateFeeAnalysis(currentPrice)

	// Calculate base grid spacing from volatility
	baseSpacing := gc.calculateVolatilityBasedSpacing(currentPrice, volatility, volatilityCategory)

	// Optimize spacing for profitability
	optimalSpacing := gc.optimizeSpacingForProfit(baseSpacing, feeAnalysis)

	// Calculate optimal number of levels
	optimalLevels := gc.calculateOptimalLevels(currentPrice, optimalSpacing, volatilityCategory)

	// Calculate grid bounds
	upperBound, lowerBound := gc.calculateGridBounds(currentPrice, optimalSpacing, optimalLevels, volatility)

	// Calculate position size
	positionSize := gc.calculatePositionSize(accountBalance, currentPrice, optimalLevels)

	// Calculate expected profits
	expectedProfit, profitMargin := gc.calculateExpectedProfits(currentPrice, optimalSpacing, positionSize, feeAnalysis)

	// Calculate breakeven requirements
	breakevenLevels, winRateRequired := gc.calculateBreakevenRequirements(feeAnalysis, profitMargin)

	// Calculate optimization score
	score := gc.calculateOptimizationScore(profitMargin, optimalLevels, feeAnalysis, volatilityCategory)

	return &GridCalculationResult{
		GridSpacing:        optimalSpacing,
		GridLevels:         optimalLevels,
		TotalRange:         (upperBound - lowerBound) / currentPrice * 100,
		UpperBound:         upperBound,
		LowerBound:         lowerBound,
		PositionSize:       positionSize,
		ExpectedProfit:     expectedProfit,
		ProfitMargin:       profitMargin,
		BreakevenLevels:    breakevenLevels,
		WinRateRequired:    winRateRequired,
		OptimizationScore:  score,
		SetupReason:        gc.generateSetupReason(volatilityCategory, optimalLevels, optimalSpacing),
	}
}

// calculateFeeAnalysis performs detailed fee analysis
func (gc *GridCalculator) calculateFeeAnalysis(currentPrice float64) *FeeAnalysis {
	totalFeesPerRoundTrip := gc.MakerFee + gc.TakerFee

	return &FeeAnalysis{
		TotalFeesPerRoundTrip: totalFeesPerRoundTrip,
		MakerFeePerLevel:      gc.MakerFee,
		TakerFeePerLevel:      gc.TakerFee,
		MinProfitableSpread:   totalFeesPerRoundTrip + gc.MinProfitPerLevel,
		FeeBurden:            (totalFeesPerRoundTrip / (gc.MinProfitPerLevel + totalFeesPerRoundTrip)) * 100,
	}
}

// calculateVolatilityBasedSpacing calculates base spacing from volatility
func (gc *GridCalculator) calculateVolatilityBasedSpacing(currentPrice, volatility float64, category string) float64 {
	var spacing float64

	switch category {
	case "low":
		// Low volatility: tighter grid (0.3-0.5%)
		spacing = math.Max(0.003, volatility*0.5)
	case "medium":
		// Medium volatility: moderate grid (0.5-0.8%)
		spacing = math.Max(0.005, volatility*0.7)
	case "high":
		// High volatility: wider grid (0.8-1.2%)
		spacing = math.Max(0.008, volatility*1.0)
	default:
		// Default to medium
		spacing = math.Max(0.005, volatility*0.7)
	}

	// Constrain to min/max bounds
	spacing = math.Max(gc.MinGridSpacing, math.Min(gc.MaxGridSpacing, spacing))

	return spacing
}

// optimizeSpacingForProfit optimizes grid spacing for maximum profitability
func (gc *GridCalculator) optimizeSpacingForProfit(baseSpacing float64, feeAnalysis *FeeAnalysis) float64 {
	// Calculate minimum profitable spacing
	minProfitableSpacing := feeAnalysis.MinProfitableSpread

	// Ensure spacing covers fees and minimum profit
	optimalSpacing := math.Max(baseSpacing, minProfitableSpacing)

	// Add buffer for market inefficiencies and slippage
	optimalSpacing *= 1.1 // 10% buffer

	// Ensure it's still within reasonable bounds
	optimalSpacing = math.Max(gc.MinGridSpacing, math.Min(gc.MaxGridSpacing, optimalSpacing))

	return optimalSpacing
}

// calculateOptimalLevels calculates optimal number of grid levels
func (gc *GridCalculator) calculateOptimalLevels(currentPrice, spacing float64, category string) int {
	// Calculate how many levels fit in a reasonable price range
	// Typical range: 3-10% of current price
	var priceRange float64

	switch category {
	case "low":
		priceRange = 0.03 // 3%
	case "medium":
		priceRange = 0.05 // 5%
	case "high":
		priceRange = 0.08 // 8%
	default:
		priceRange = 0.05 // 5%
	}

	// Calculate levels based on spacing
	levelCount := int((currentPrice * priceRange) / (currentPrice * spacing))

	// Constrain to min/max bounds
	levelCount = maxInt(gc.MinGridLevels, minInt(gc.MaxGridLevels, levelCount))

	return levelCount
}

// calculateGridBounds calculates upper and lower bounds for the grid
func (gc *GridCalculator) calculateGridBounds(currentPrice, spacing float64, levels int, volatility float64) (float64, float64) {
	// Calculate total grid range
	totalRange := spacing * float64(levels)

	// Add volatility buffer
	volatilityBuffer := volatility * 2 // 2x ATR as buffer
	totalRange += volatilityBuffer / currentPrice

	// Calculate bounds centered around current price
	halfRange := totalRange / 2
	upperBound := currentPrice * (1 + halfRange)
	lowerBound := currentPrice * (1 - halfRange)

	// Ensure lower bound is positive
	if lowerBound < 0 {
		lowerBound = 0.01
		upperBound = lowerBound + (currentPrice * totalRange)
	}

	return upperBound, lowerBound
}

// calculatePositionSize calculates optimal position size per grid level
func (gc *GridCalculator) calculatePositionSize(accountBalance, currentPrice float64, levels int) float64 {
	// Calculate risk per trade
	maxRiskPerTrade := accountBalance * gc.MaxRiskPerTrade

	// Calculate position size per level (distributed across all levels)
	riskPerLevel := maxRiskPerTrade / float64(levels)

	// Position size = Risk / (Price * Risk%)
	positionSize := riskPerLevel / (currentPrice * 0.02) // 2% risk per position

	return positionSize
}

// calculateExpectedProfits calculates expected profit and profit margins
func (gc *GridCalculator) calculateExpectedProfits(currentPrice, spacing, positionSize float64, feeAnalysis *FeeAnalysis) (float64, float64) {
	// Calculate notional value per level
	notionalValue := positionSize * currentPrice

	// Calculate gross profit per level
	grossProfit := spacing * notionalValue

	// Calculate total fees per round trip
	totalFees := feeAnalysis.TotalFeesPerRoundTrip * notionalValue

	// Net profit
	netProfit := grossProfit - totalFees

	// Profit margin as percentage of notional value
	profitMargin := (netProfit / notionalValue) * 100

	return netProfit, profitMargin
}

// calculateBreakevenRequirements calculates breakeven requirements
func (gc *GridCalculator) calculateBreakevenRequirements(feeAnalysis *FeeAnalysis, profitMargin float64) (int, float64) {
	// Calculate how many levels need to be profitable to cover losses from other levels
	feeBurden := feeAnalysis.TotalFeesPerRoundTrip * 100

	// Breakeven levels: need enough profitable trades to cover fees
	breakevenLevels := int(math.Ceil(feeBurden / profitMargin))

	// Required win rate to be profitable
	winRateRequired := (feeBurden / (profitMargin + feeBurden)) * 100

	return breakevenLevels, winRateRequired
}

// calculateOptimizationScore calculates how well optimized the grid is
func (gc *GridCalculator) calculateOptimizationScore(profitMargin float64, levels int, feeAnalysis *FeeAnalysis, volatilityCategory string) float64 {
	// Score components (0-100 each)
	profitScore := min(100, profitMargin*1000) // Convert to 0-100 scale
	feeScore := max(0, 100-(feeAnalysis.FeeBurden*10)) // Lower fee burden = higher score
	levelScore := gc.calculateLevelScore(levels, volatilityCategory)

	// Weighted average
	score := (profitScore*0.5 + feeScore*0.3 + levelScore*0.2)

	return score
}

// calculateLevelScore scores the number of levels based on volatility
func (gc *GridCalculator) calculateLevelScore(levels int, volatilityCategory string) float64 {
	var optimalLevels int

	switch volatilityCategory {
	case "low":
		optimalLevels = gc.MaxGridLevels // More levels for stable markets
	case "medium":
		optimalLevels = (gc.MinGridLevels + gc.MaxGridLevels) / 2
	case "high":
		optimalLevels = gc.MinGridLevels // Fewer levels for volatile markets
	default:
		optimalLevels = (gc.MinGridLevels + gc.MaxGridLevels) / 2
	}

	// Score based on distance from optimal
	distance := math.Abs(float64(levels - optimalLevels))
	score := max(0, 100-distance*5) // Deduct 5 points per level away from optimal

	return score
}

// generateSetupReason generates a human-readable reason for the grid setup
func (gc *GridCalculator) generateSetupReason(volatilityCategory string, levels int, spacing float64) string {
	reason := "Optimized for " + volatilityCategory + " volatility"

	if levels >= gc.MaxGridLevels-2 {
		reason += " (maximum grid levels for precision)"
	} else if levels <= gc.MinGridLevels+2 {
		reason += " (minimum grid levels for risk management)"
	}

	if spacing <= gc.MinGridSpacing*1.2 {
		reason += " (tight spacing for high-frequency trading)"
	} else if spacing >= gc.MaxGridSpacing*0.8 {
		reason += " (wide spacing for volatility accommodation)"
	}

	return reason
}

// ValidateGridConfiguration validates if a grid configuration is profitable
func (gc *GridCalculator) ValidateGridConfiguration(spacing, positionSize, currentPrice float64) (bool, string) {
	// Calculate notional value
	notionalValue := positionSize * currentPrice

	// Calculate fees
	totalFees := (gc.MakerFee + gc.TakerFee) * notionalValue

	// Calculate gross profit
	grossProfit := spacing * notionalValue

	// Net profit must be positive
	netProfit := grossProfit - totalFees

	if netProfit <= 0 {
		return false, "Grid spacing too small to cover fees"
	}

	// Minimum profit margin
	profitMargin := (netProfit / notionalValue) * 100
	if profitMargin < gc.MinProfitPerLevel*100 {
		return false, "Profit margin below minimum threshold"
	}

	// Maximum spacing
	if spacing > gc.MaxGridSpacing {
		return false, "Grid spacing too wide for effective trading"
	}

	return true, "Grid configuration is valid and profitable"
}

// Helper functions
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}