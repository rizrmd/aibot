package strategy

import (
	"math"
	"time"
)

// RiskManager manages portfolio risk and calculates optimal position sizes
type RiskManager struct {
	// Risk configuration
	MaxPortfolioRisk      float64 `json:"max_portfolio_risk"`       // Max portfolio risk (5%)
	MaxPositionRisk       float64 `json:"max_position_risk"`        // Max risk per position (2%)
	MaxCorrelation        float64 `json:"max_correlation"`          // Max correlation between positions (0.7)
	MaxDrawdown           float64 `json:"max_drawdown"`             // Max allowed drawdown (10%)
	MinRiskRewardRatio    float64 `json:"min_risk_reward_ratio"`    // Minimum risk/reward ratio (1.5)

	// Position sizing
	MinPositionSize       float64 `json:"min_position_size"`        // Minimum position size (0.001 BTC)
	MaxPositionSize       float64 `json:"max_position_size"`        // Maximum position size (1.0 BTC)
	DefaultLeverage       float64 `json:"default_leverage"`         // Default leverage (5x)
	MaxLeverage           float64 `json:"max_leverage"`             // Maximum leverage (10x)

	// Risk metrics
	PortfolioValue        float64 `json:"portfolio_value"`          // Current portfolio value
	AvailableMargin       float64 `json:"available_margin"`         // Available margin for trading
	UsedMargin            float64 `json:"used_margin"`              // Currently used margin
	TotalExposure         float64 `json:"total_exposure"`           // Total exposure across all positions
	CurrentDrawdown       float64 `json:"current_drawdown"`         // Current portfolio drawdown
	MaxDrawdownReached    float64 `json:"max_drawdown_reached"`     // Maximum drawdown ever reached

	// State tracking
	positions             []RiskPosition `json:"positions"`
	riskMetrics           RiskMetrics    `json:"risk_metrics"`
	marginCalls           int            `json:"margin_calls"`
	lastRiskAssessment    time.Time      `json:"last_risk_assessment"`

	// Risk adjustment factors
	VolatilityMultiplier  float64 `json:"volatility_multiplier"`    // Volatility risk multiplier
	CorrelationPenalty    float64 `json:"correlation_penalty"`      // Penalty for correlated positions
	ConcentrationLimit    float64 `json:"concentration_limit"`      // Max concentration in one asset (30%)
}

// RiskPosition represents a position from risk management perspective
type RiskPosition struct {
	Symbol           string    `json:"symbol"`
	PositionSize     float64   `json:"position_size"`
	NotionalValue    float64   `json:"notional_value"`
	MarginUsed       float64   `json:"margin_used"`
	RiskAmount       float64   `json:"risk_amount"`
	RiskPercentage   float64   `json:"risk_percentage"`
	ExpectedReturn   float64   `json:"expected_return"`
	RiskRewardRatio  float64   `json:"risk_reward_ratio"`
	Volatility       float64   `json:"volatility"`
	Correlations     map[string]float64 `json:"correlations"`
	OpenTime         time.Time `json:"open_time"`
	StopLoss         float64   `json:"stop_loss"`
	TakeProfit       float64   `json:"take_profit"`
}

// RiskMetrics contains current portfolio risk metrics
type RiskMetrics struct {
	PortfolioVolatility    float64              `json:"portfolio_volatility"`
	TotalRisk              float64              `json:"total_risk"`
	RiskPerDollar         float64              `json:"risk_per_dollar"`
	SharpeRatio           float64              `json:"sharpe_ratio"`
	SortinoRatio          float64              `json:"sortino_ratio"`
	ValueAtRisk           float64              `json:"value_at_risk"`           // 95% VaR
	ExpectedShortfall     float64              `json:"expected_shortfall"`      // CVaR
	MaxCorrelation        float64              `json:"max_correlation"`
	ConcentrationRisk     float64              `json:"concentration_risk"`
	DiversificationScore  float64              `json:"diversification_score"`
	StressTestResults     map[string]float64   `json:"stress_test_results"`
}

// PositionSizingRequest contains parameters for position sizing calculation
type PositionSizingRequest struct {
	Symbol           string  `json:"symbol"`
	EntryPrice       float64 `json:"entry_price"`
	StopLoss         float64 `json:"stop_loss"`
	TakeProfit       float64 `json:"take_profit"`
	Confidence       float64 `json:"confidence"`        // Signal confidence (0-1)
	Volatility       float64 `json:"volatility"`        // Asset volatility
	ExpectedReturn   float64 `json:"expected_return"`   // Expected return percentage
	Leverage         float64 `json:"leverage"`          // Desired leverage (optional)
	MaxRiskPercent   float64 `json:"max_risk_percent"`  // Max risk for this position
}

// PositionSizingResult contains the calculated position sizing
type PositionSizingResult struct {
	RecommendedSize     float64 `json:"recommended_size"`
	NotionalValue       float64 `json:"notional_value"`
	MarginRequired      float64 `json:"margin_required"`
	RiskAmount          float64 `json:"risk_amount"`
	RiskPercentage      float64 `json:"risk_percentage"`
	LeverageUsed        float64 `json:"leverage_used"`
	RiskRewardRatio     float64 `json:"risk_reward_ratio"`
	ConfidenceScore     float64 `json:"confidence_score"`
	AcceptableRisk      bool    `json:"acceptable_risk"`
	Reason              string  `json:"reason"`
	Warnings            []string `json:"warnings"`
}

// RiskAssessment represents a comprehensive risk assessment
type RiskAssessment struct {
	Timestamp              time.Time             `json:"timestamp"`
	PortfolioHealth        string                `json:"portfolio_health"`        // "healthy", "warning", "critical"
	OverallRiskLevel       float64               `json:"overall_risk_level"`      // 0-1
	RiskFactors            []string              `json:"risk_factors"`
	RecommendedActions     []string              `json:"recommended_actions"`
	RiskLimitBreaches      []string              `json:"risk_limit_breaches"`
	MarginCallRisk         bool                  `json:"margin_call_risk"`
	LiquidityRisk          float64               `json:"liquidity_risk"`
	ConcentrationRisk      float64               `json:"concentration_risk"`
	SystemicRisk           float64               `json:"systemic_risk"`
	RiskMetrics            map[string]float64    `json:"risk_metrics"`
}

// RiskManagerConfig holds configuration for risk management
type RiskManagerConfig struct {
	MaxPortfolioRisk     float64 `json:"max_portfolio_risk"`     // 5%
	MaxPositionRisk      float64 `json:"max_position_risk"`      // 2%
	MaxCorrelation       float64 `json:"max_correlation"`        // 0.7
	MaxDrawdown          float64 `json:"max_drawdown"`           // 10%
	MinRiskRewardRatio   float64 `json:"min_risk_reward_ratio"`  // 1.5
	DefaultLeverage      float64 `json:"default_leverage"`       // 5x
	MaxLeverage          float64 `json:"max_leverage"`           // 10x
	ConcentrationLimit   float64 `json:"concentration_limit"`    // 30%
	VolatilityMultiplier float64 `json:"volatility_multiplier"`  // 1.5
}

// NewRiskManager creates a new risk manager
func NewRiskManager(config RiskManagerConfig, initialBalance float64) *RiskManager {
	// Set defaults
	if config.MaxPortfolioRisk == 0 {
		config.MaxPortfolioRisk = 0.05 // 5%
	}
	if config.MaxPositionRisk == 0 {
		config.MaxPositionRisk = 0.02 // 2%
	}
	if config.MaxCorrelation == 0 {
		config.MaxCorrelation = 0.7
	}
	if config.MaxDrawdown == 0 {
		config.MaxDrawdown = 0.10 // 10%
	}
	if config.MinRiskRewardRatio == 0 {
		config.MinRiskRewardRatio = 1.5
	}
	if config.DefaultLeverage == 0 {
		config.DefaultLeverage = 5.0
	}
	if config.MaxLeverage == 0 {
		config.MaxLeverage = 10.0
	}
	if config.ConcentrationLimit == 0 {
		config.ConcentrationLimit = 0.3 // 30%
	}
	if config.VolatilityMultiplier == 0 {
		config.VolatilityMultiplier = 1.5
	}

	return &RiskManager{
		MaxPortfolioRisk:     config.MaxPortfolioRisk,
		MaxPositionRisk:      config.MaxPositionRisk,
		MaxCorrelation:       config.MaxCorrelation,
		MaxDrawdown:          config.MaxDrawdown,
		MinRiskRewardRatio:   config.MinRiskRewardRatio,
		DefaultLeverage:      config.DefaultLeverage,
		MaxLeverage:          config.MaxLeverage,
		ConcentrationLimit:   config.ConcentrationLimit,
		VolatilityMultiplier: config.VolatilityMultiplier,
		MinPositionSize:      0.001, // 0.001 BTC minimum
		MaxPositionSize:      1.0,   // 1.0 BTC maximum
		PortfolioValue:       initialBalance,
		AvailableMargin:      initialBalance,
		positions:            make([]RiskPosition, 0),
		riskMetrics: RiskMetrics{
			StressTestResults: make(map[string]float64),
		},
		lastRiskAssessment: time.Now(),
	}
}

// CalculatePositionSize calculates optimal position size based on risk parameters
func (rm *RiskManager) CalculatePositionSize(req PositionSizingRequest) *PositionSizingResult {
	// Validate basic requirements
	if req.EntryPrice <= 0 || req.StopLoss <= 0 {
		return &PositionSizingResult{
			AcceptableRisk: false,
			Reason:         "Invalid price parameters",
		}
	}

	// Calculate risk/reward ratio
	riskRewardRatio := rm.calculateRiskRewardRatio(req.EntryPrice, req.StopLoss, req.TakeProfit)
	if riskRewardRatio < rm.MinRiskRewardRatio {
		return &PositionSizingResult{
			AcceptableRisk: false,
			Reason:         "Risk/reward ratio too low",
		}
	}

	// Calculate base position size based on risk
	riskPerUnit := math.Abs(req.EntryPrice - req.StopLoss) / req.EntryPrice
	riskAmount := rm.PortfolioValue * rm.MaxPositionRisk

	// Adjust for volatility
	volatilityAdjustedRisk := riskPerUnit * (1 + (req.Volatility * rm.VolatilityMultiplier))

	// Adjust for confidence
	confidenceMultiplier := 0.5 + (req.Confidence * 0.5) // 0.5 to 1.0 based on confidence

	// Calculate position size
	basePositionSize := (riskAmount * confidenceMultiplier) / (volatilityAdjustedRisk * req.EntryPrice)

	// Apply leverage
	leverage := rm.DefaultLeverage
	if req.Leverage > 0 {
		leverage = math.Min(req.Leverage, rm.MaxLeverage)
	}
	positionSize := basePositionSize * leverage

	// Check portfolio constraints
	positionSize = rm.applyPortfolioConstraints(req.Symbol, positionSize, req.EntryPrice)

	// Calculate final metrics
	notionalValue := positionSize * req.EntryPrice
	marginRequired := notionalValue / leverage
	actualRiskAmount := positionSize * math.Abs(req.EntryPrice - req.StopLoss)
	riskPercentage := actualRiskAmount / rm.PortfolioValue

	// Generate warnings
	warnings := rm.generateWarnings(req, positionSize, riskPercentage)

	// Determine if risk is acceptable
	acceptableRisk := riskPercentage <= rm.MaxPositionRisk &&
		marginRequired <= rm.AvailableMargin &&
		rm.checkCorrelationLimits(req.Symbol, positionSize, req.EntryPrice)

	result := &PositionSizingResult{
		RecommendedSize: positionSize,
		NotionalValue:   notionalValue,
		MarginRequired:  marginRequired,
		RiskAmount:      actualRiskAmount,
		RiskPercentage:  riskPercentage,
		LeverageUsed:    leverage,
		RiskRewardRatio: riskRewardRatio,
		ConfidenceScore: req.Confidence,
		AcceptableRisk:  acceptableRisk,
		Reason:         rm.generatePositionReason(positionSize, riskPercentage, acceptableRisk),
		Warnings:       warnings,
	}

	return result
}

// UpdatePortfolio updates portfolio state after trade execution
func (rm *RiskManager) UpdatePortfolio(trade TradeUpdate) {
	// Update portfolio value based on PnL
	rm.PortfolioValue += trade.RealizedPnL

	// Update positions
	rm.updatePositions(trade)

	// Update margin usage
	rm.updateMarginUsage()

	// Update risk metrics
	rm.calculateRiskMetrics()

	rm.lastRiskAssessment = time.Now()
}

// AssessRisk performs comprehensive risk assessment
func (rm *RiskManager) AssessRisk() *RiskAssessment {
	assessment := &RiskAssessment{
		Timestamp: time.Now(),
		RiskFactors: make([]string, 0),
		RecommendedActions: make([]string, 0),
		RiskLimitBreaches: make([]string, 0),
		RiskMetrics: make(map[string]float64),
	}

	// Calculate overall risk level
	overallRisk := rm.calculateOverallRisk()
	assessment.OverallRiskLevel = overallRisk

	// Determine portfolio health
	assessment.PortfolioHealth = rm.determinePortfolioHealth(overallRisk)

	// Check various risk factors
	rm.checkDrawdownRisk(assessment)
	rm.checkConcentrationRisk(assessment)
	rm.checkCorrelationRisk(assessment)
	rm.checkLeverageRisk(assessment)
	rm.checkLiquidityRisk(assessment)
	rm.checkMarginCallRisk(assessment)

	// Perform stress tests
	rm.performStressTests(assessment)

	// Generate recommended actions
	rm.generateRecommendedActions(assessment)

	return assessment
}

// calculateRiskRewardRatio calculates risk/reward ratio
func (rm *RiskManager) calculateRiskRewardRatio(entryPrice, stopLoss, takeProfit float64) float64 {
	risk := math.Abs(entryPrice - stopLoss)
	reward := math.Abs(takeProfit - entryPrice)

	if risk == 0 {
		return 0
	}

	return reward / risk
}

// applyPortfolioConstraints applies portfolio-level constraints to position size
func (rm *RiskManager) applyPortfolioConstraints(symbol string, positionSize, price float64) float64 {
	// Check minimum/maximum position size
	if positionSize < rm.MinPositionSize {
		positionSize = 0 // Position too small to open
	}
	if positionSize > rm.MaxPositionSize {
		positionSize = rm.MaxPositionSize
	}

	// Check margin availability
	notionalValue := positionSize * price
	marginRequired := notionalValue / rm.DefaultLeverage

	if marginRequired > rm.AvailableMargin {
		// Reduce position size to fit available margin
		maxNotionalValue := rm.AvailableMargin * rm.DefaultLeverage
		positionSize = maxNotionalValue / price
	}

	// Check concentration limits
	existingPosition := rm.getPositionForSymbol(symbol)
	totalPosition := existingPosition + positionSize
	concentrationValue := totalPosition * price
	maxConcentrationValue := rm.PortfolioValue * rm.ConcentrationLimit

	if concentrationValue > maxConcentrationValue {
		// Reduce position to respect concentration limit
		maxPositionSize := (maxConcentrationValue - existingPosition*price) / price
		positionSize = math.Min(positionSize, maxPositionSize)
	}

	return math.Max(0, positionSize)
}

// checkCorrelationLimits checks if adding position would violate correlation limits
func (rm *RiskManager) checkCorrelationLimits(symbol string, positionSize, price float64) bool {
	// For simplicity, assume all crypto assets have some correlation
	// In a real implementation, you'd use actual correlation data

	newNotionalValue := positionSize * price
	totalExposure := rm.TotalExposure + newNotionalValue

	// Check if total exposure exceeds limits relative to portfolio value
	maxTotalExposure := rm.PortfolioValue * rm.MaxLeverage
	return totalExposure <= maxTotalExposure
}

// generateWarnings generates risk warnings for the position
func (rm *RiskManager) generateWarnings(req PositionSizingRequest, positionSize, riskPercentage float64) []string {
	warnings := make([]string, 0)

	if riskPercentage > rm.MaxPositionRisk*0.8 {
		warnings = append(warnings, "Position risk approaching maximum limit")
	}

	if req.Volatility > 0.05 { // 5% volatility
		warnings = append(warnings, "High volatility detected")
	}

	if req.Confidence < 0.5 {
		warnings = append(warnings, "Low confidence signal")
	}

	if req.Leverage > rm.DefaultLeverage*1.5 {
		warnings = append(warnings, "High leverage usage")
	}

	notionalValue := positionSize * req.EntryPrice
	if notionalValue > rm.PortfolioValue*0.3 {
		warnings = append(warnings, "Large position relative to portfolio")
	}

	return warnings
}

// generatePositionReason generates a reason for the position sizing
func (rm *RiskManager) generatePositionReason(positionSize, riskPercentage float64, acceptableRisk bool) string {
	if !acceptableRisk {
		return "Position rejected due to risk constraints"
	}

	if riskPercentage > rm.MaxPositionRisk*0.8 {
		return "Position sized near maximum risk limit"
	}

	if positionSize < rm.MinPositionSize {
		return "Position too small to execute"
	}

	return "Position sized within risk parameters"
}

// updatePositions updates the risk position tracking
func (rm *RiskManager) updatePositions(trade TradeUpdate) {
	// Find existing position for the symbol
	var existingPos *RiskPosition
	var existingIndex int = -1

	for i, pos := range rm.positions {
		if pos.Symbol == trade.Symbol {
			existingPos = &rm.positions[i]
			existingIndex = i
			break
		}
	}

	if existingPos != nil {
		// Update existing position
		existingPos.PositionSize += trade.Quantity
		existingPos.NotionalValue = existingPos.PositionSize * trade.Price

		// Remove position if size is zero or very small
		if math.Abs(existingPos.PositionSize) < rm.MinPositionSize {
			rm.positions = append(rm.positions[:existingIndex], rm.positions[existingIndex+1:]...)
		}
	} else if math.Abs(trade.Quantity) >= rm.MinPositionSize {
		// Add new position
		riskAmount := math.Abs(trade.Quantity * (trade.Price - trade.StopLoss))
		riskPercentage := riskAmount / rm.PortfolioValue

		newPos := RiskPosition{
			Symbol:          trade.Symbol,
			PositionSize:    trade.Quantity,
			NotionalValue:   trade.Quantity * trade.Price,
			RiskAmount:      riskAmount,
			RiskPercentage:  riskPercentage,
			Volatility:      trade.Volatility,
			Correlations:    make(map[string]float64),
			OpenTime:        trade.Timestamp,
			StopLoss:        trade.StopLoss,
			TakeProfit:      trade.TakeProfit,
		}

		rm.positions = append(rm.positions, newPos)
	}
}

// updateMarginUsage updates margin usage calculations
func (rm *RiskManager) updateMarginUsage() {
	rm.UsedMargin = 0
	rm.TotalExposure = 0

	for _, pos := range rm.positions {
		marginForPosition := pos.NotionalValue / rm.DefaultLeverage
		rm.UsedMargin += marginForPosition
		rm.TotalExposure += pos.NotionalValue
	}

	rm.AvailableMargin = rm.PortfolioValue - rm.UsedMargin
}

// calculateRiskMetrics calculates comprehensive risk metrics
func (rm *RiskManager) calculateRiskMetrics() {
	if len(rm.positions) == 0 {
		rm.riskMetrics = RiskMetrics{
			StressTestResults: make(map[string]float64),
		}
		return
	}

	// Calculate portfolio volatility
	rm.riskMetrics.PortfolioVolatility = rm.calculatePortfolioVolatility()

	// Calculate total risk
	rm.riskMetrics.TotalRisk = rm.calculateTotalRisk()

	// Calculate VaR and Expected Shortfall
	rm.riskMetrics.ValueAtRisk = rm.calculateValueAtRisk(0.95)
	rm.riskMetrics.ExpectedShortfall = rm.calculateExpectedShortfall(0.95)

	// Calculate concentration risk
	rm.riskMetrics.ConcentrationRisk = rm.calculateConcentrationRisk()

	// Calculate diversification score
	rm.riskMetrics.DiversificationScore = rm.calculateDiversificationScore()

	// Calculate maximum correlation
	rm.riskMetrics.MaxCorrelation = rm.calculateMaxCorrelation()
}

// calculateOverallRisk calculates overall portfolio risk level
func (rm *RiskManager) calculateOverallRisk() float64 {
	if rm.PortfolioValue == 0 {
		return 1.0 // Maximum risk when no portfolio value
	}

	// Risk factors weighting
	drawdownRisk := math.Max(0, rm.CurrentDrawdown/rm.MaxDrawdown)
	marginRisk := rm.UsedMargin / rm.PortfolioValue
	concentrationRisk := rm.riskMetrics.ConcentrationRisk
	volatilityRisk := math.Min(1.0, rm.riskMetrics.PortfolioVolatility*20) // Scale volatility risk

	// Weighted average of risk factors
	overallRisk := (drawdownRisk*0.3 + marginRisk*0.3 + concentrationRisk*0.2 + volatilityRisk*0.2)

	return math.Min(1.0, overallRisk)
}

// determinePortfolioHealth determines portfolio health status
func (rm *RiskManager) determinePortfolioHealth(riskLevel float64) string {
	if riskLevel >= 0.8 {
		return "critical"
	} else if riskLevel >= 0.6 {
		return "warning"
	} else {
		return "healthy"
	}
}

// checkDrawdownRisk checks drawdown-related risks
func (rm *RiskManager) checkDrawdownRisk(assessment *RiskAssessment) {
	if rm.CurrentDrawdown > rm.MaxDrawdown*0.8 {
		assessment.RiskFactors = append(assessment.RiskFactors, "High drawdown detected")
		assessment.RiskLimitBreaches = append(assessment.RiskLimitBreaches, "Approaching maximum drawdown limit")
	}
}

// checkConcentrationRisk checks concentration-related risks
func (rm *RiskManager) checkConcentrationRisk(assessment *RiskAssessment) {
	concentrationRisk := rm.calculateConcentrationRisk()
	assessment.ConcentrationRisk = concentrationRisk

	if concentrationRisk > rm.ConcentrationLimit {
		assessment.RiskFactors = append(assessment.RiskFactors, "High concentration in few assets")
		assessment.RecommendedActions = append(assessment.RecommendedActions, "Consider diversifying positions")
	}
}

// checkCorrelationRisk checks correlation-related risks
func (rm *RiskManager) checkCorrelationRisk(assessment *RiskAssessment) {
	maxCorrelation := rm.calculateMaxCorrelation()
	if maxCorrelation > rm.MaxCorrelation {
		assessment.RiskFactors = append(assessment.RiskFactors, "High correlation between positions")
		assessment.RecommendedActions = append(assessment.RecommendedActions, "Reduce exposure to correlated assets")
	}
}

// checkLeverageRisk checks leverage-related risks
func (rm *RiskManager) checkLeverageRisk(assessment *RiskAssessment) {
	effectiveLeverage := rm.TotalExposure / rm.PortfolioValue
	if effectiveLeverage > rm.DefaultLeverage*1.5 {
		assessment.RiskFactors = append(assessment.RiskFactors, "High leverage usage")
		assessment.RiskLimitBreaches = append(assessment.RiskLimitBreaches, "Excessive leverage detected")
	}
}

// checkLiquidityRisk checks liquidity-related risks
func (rm *RiskManager) checkLiquidityRisk(assessment *RiskAssessment) {
	// Simple liquidity risk based on margin usage
	liquidityRisk := rm.UsedMargin / rm.PortfolioValue
	assessment.LiquidityRisk = liquidityRisk

	if liquidityRisk > 0.8 {
		assessment.RiskFactors = append(assessment.RiskFactors, "Low liquidity available")
		assessment.RecommendedActions = append(assessment.RecommendedActions, "Reduce position sizes to free up margin")
	}
}

// checkMarginCallRisk checks margin call risk
func (rm *RiskManager) checkMarginCallRisk(assessment *RiskAssessment) {
	marginUsageRatio := rm.UsedMargin / rm.PortfolioValue
	assessment.MarginCallRisk = marginUsageRatio > 0.9

	if assessment.MarginCallRisk {
		assessment.RiskFactors = append(assessment.RiskFactors, "Margin call risk")
		assessment.RiskLimitBreaches = append(assessment.RiskLimitBreaches, "Critical: Near margin call")
	}
}

// performStressTests performs stress tests on the portfolio
func (rm *RiskManager) performStressTests(assessment *RiskAssessment) {
	// Market crash scenario (-20%)
	crashLoss := rm.TotalExposure * 0.20
	assessment.RiskMetrics["market_crash_20_percent"] = crashLoss / rm.PortfolioValue

	// Volatility spike scenario (3x volatility increase)
	volatilityLoss := rm.TotalExposure * rm.riskMetrics.PortfolioVolatility * 2
	assessment.RiskMetrics["volatility_spike"] = volatilityLoss / rm.PortfolioValue

	// Correlation breakdown scenario
	correlationLoss := rm.TotalExposure * 0.10 // 10% loss if correlations increase
	assessment.RiskMetrics["correlation_breakdown"] = correlationLoss / rm.PortfolioValue
}

// generateRecommendedActions generates recommended actions based on risk assessment
func (rm *RiskManager) generateRecommendedActions(assessment *RiskAssessment) {
	if assessment.OverallRiskLevel > 0.7 {
		assessment.RecommendedActions = append(assessment.RecommendedActions, "Reduce overall portfolio risk")
	}

	if rm.CurrentDrawdown > rm.MaxDrawdown*0.5 {
		assessment.RecommendedActions = append(assessment.RecommendedActions, "Consider reducing position sizes")
	}

	if assessment.MarginCallRisk {
		assessment.RecommendedActions = append(assessment.RecommendedActions, "Immediate position reduction required")
	}

	if len(assessment.RecommendedActions) == 0 {
		assessment.RecommendedActions = append(assessment.RecommendedActions, "Risk levels are acceptable")
	}
}

// Helper functions
func (rm *RiskManager) getPositionForSymbol(symbol string) float64 {
	for _, pos := range rm.positions {
		if pos.Symbol == symbol {
			return pos.PositionSize
		}
	}
	return 0
}

func (rm *RiskManager) calculatePortfolioVolatility() float64 {
	if len(rm.positions) == 0 {
		return 0
	}

	// Simple approximation: weighted average of individual volatilities
	totalValue := 0.0
	weightedVolatility := 0.0

	for _, pos := range rm.positions {
		weightedVolatility += pos.NotionalValue * pos.Volatility
		totalValue += pos.NotionalValue
	}

	if totalValue == 0 {
		return 0
	}

	return weightedVolatility / totalValue
}

func (rm *RiskManager) calculateTotalRisk() float64 {
	totalRisk := 0.0
	for _, pos := range rm.positions {
		totalRisk += pos.RiskAmount
	}
	return totalRisk / rm.PortfolioValue
}

func (rm *RiskManager) calculateValueAtRisk(confidence float64) float64 {
	// Simplified VaR calculation using normal distribution
	if rm.riskMetrics.PortfolioVolatility == 0 {
		return 0
	}

	// For 95% confidence, z-score is approximately 1.65
	zScore := 1.65
	return rm.PortfolioValue * rm.riskMetrics.PortfolioVolatility * zScore
}

func (rm *RiskManager) calculateExpectedShortfall(confidence float64) float64 {
	// Simplified Expected Shortfall (CVaR) calculation
	_ = rm.calculateValueAtRisk(confidence) // Calculate VaR for consistency
	// For normal distribution, expected shortfall at 95% is approximately 2.06 sigma
	return rm.PortfolioValue * rm.riskMetrics.PortfolioVolatility * 2.06
}

func (rm *RiskManager) calculateConcentrationRisk() float64 {
	if len(rm.positions) == 0 || rm.TotalExposure == 0 {
		return 0
	}

	// Calculate Herfindahl-Hirschman Index for concentration
	hhi := 0.0
	for _, pos := range rm.positions {
		weight := pos.NotionalValue / rm.TotalExposure
		hhi += weight * weight
	}

	return hhi
}

func (rm *RiskManager) calculateDiversificationScore() float64 {
	concentration := rm.calculateConcentrationRisk()
	// Convert concentration to diversification score (inverse relationship)
	return 1.0 - concentration
}

func (rm *RiskManager) calculateMaxCorrelation() float64 {
	// Simplified: assume 0.7 correlation between all crypto assets
	// In a real implementation, you'd calculate actual correlations
	if len(rm.positions) <= 1 {
		return 0
	}
	return 0.7
}

// TradeUpdate represents a trade execution update
type TradeUpdate struct {
	Symbol        string    `json:"symbol"`
	Quantity      float64   `json:"quantity"`      // Positive for long, negative for short
	Price         float64   `json:"price"`
	RealizedPnL   float64   `json:"realized_pnl"`
	Timestamp     time.Time `json:"timestamp"`
	StopLoss      float64   `json:"stop_loss"`
	TakeProfit    float64   `json:"take_profit"`
	Volatility    float64   `json:"volatility"`
}

// GetRiskStats returns current risk management statistics
func (rm *RiskManager) GetRiskStats() map[string]interface{} {
	return map[string]interface{}{
		"portfolio_value":       rm.PortfolioValue,
		"available_margin":      rm.AvailableMargin,
		"used_margin":           rm.UsedMargin,
		"total_exposure":        rm.TotalExposure,
		"current_drawdown":      rm.CurrentDrawdown,
		"position_count":        len(rm.positions),
		"margin_calls":          rm.marginCalls,
		"portfolio_health":      rm.AssessRisk().PortfolioHealth,
		"overall_risk_level":    rm.calculateOverallRisk(),
		"risk_metrics":          rm.riskMetrics,
	}
}