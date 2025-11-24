package indicators

import (
	"aibot/internal/types"
	"github.com/cinar/indicator"
	"sync"
)

// TechnicalAnalyzer wraps the Indicator Go library for technical analysis
type TechnicalAnalyzer struct {
	// Data storage per symbol
	data map[string]*SymbolData
	mu   sync.RWMutex
}

// SymbolData stores OHLCV data and calculated indicators for a symbol
type SymbolData struct {
	Symbol string
	Candles []types.OHLCV
	// Indicator values
	// Trend indicators
	SMA    []float64  // Simple Moving Average
	EMA    []float64  // Exponential Moving Average
	// Momentum indicators
	RSI    []float64  // Relative Strength Index
	MACD   []float64  // MACD line
	MACDSignal []float64 // MACD signal line
	MACDHist []float64  // MACD histogram
	// Volatility indicators
	ATR    []float64  // Average True Range
	BollingerUpper []float64 // Bollinger Bands upper
	BollingerMiddle []float64 // Bollinger Bands middle
	BollingerLower []float64 // Bollinger Bands lower
	// Volume indicators
	VolumeSMA []float64 // Volume Simple Moving Average
}

// AnalyzerConfig holds configuration for technical analysis
type AnalyzerConfig struct {
	MaxHistoryCandles int `json:"max_history_candles"`
	// Trend indicator periods
	SMAPeriod int `json:"sma_period"`
	EMAPeriod int `json:"ema_period"`
	// Momentum indicator periods
	RSIPeriod int `json:"rsi_period"`
	MACDFast int `json:"macd_fast"`
	MACDSlow int `json:"macd_slow"`
	MACDSignal int `json:"macd_signal"`
	// Volatility indicator periods
	ATRPeriod int `json:"atr_period"`
	BollingerPeriod int `json:"bollinger_period"`
	BollingerStdDev float64 `json:"bollinger_std_dev"`
	// Volume indicator periods
	VolumeSMAPeriod int `json:"volume_sma_period"`
}

// NewTechnicalAnalyzer creates a new technical analyzer
func NewTechnicalAnalyzer(config AnalyzerConfig) *TechnicalAnalyzer {
	// Set defaults
	if config.MaxHistoryCandles == 0 {
		config.MaxHistoryCandles = 200
	}
	if config.SMAPeriod == 0 {
		config.SMAPeriod = 20
	}
	if config.EMAPeriod == 0 {
		config.EMAPeriod = 20
	}
	if config.RSIPeriod == 0 {
		config.RSIPeriod = 14
	}
	if config.MACDFast == 0 {
		config.MACDFast = 12
	}
	if config.MACDSlow == 0 {
		config.MACDSlow = 26
	}
	if config.MACDSignal == 0 {
		config.MACDSignal = 9
	}
	if config.ATRPeriod == 0 {
		config.ATRPeriod = 14
	}
	if config.BollingerPeriod == 0 {
		config.BollingerPeriod = 20
	}
	if config.BollingerStdDev == 0 {
		config.BollingerStdDev = 2.0
	}
	if config.VolumeSMAPeriod == 0 {
		config.VolumeSMAPeriod = 20
	}

	return &TechnicalAnalyzer{
		data: make(map[string]*SymbolData),
	}
}

// AddCandle adds a new OHLCV candle and updates all indicators
func (ta *TechnicalAnalyzer) AddCandle(candle types.OHLCV) {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	symbolData, exists := ta.data[candle.Symbol]
	if !exists {
		symbolData = &SymbolData{
			Symbol:  candle.Symbol,
			Candles: make([]types.OHLCV, 0),
		}
		ta.data[candle.Symbol] = symbolData
	}

	// Add new candle
	symbolData.Candles = append(symbolData.Candles, candle)

	// Limit history size
	if len(symbolData.Candles) > 200 { // Maximum history
		symbolData.Candles = symbolData.Candles[1:]
	}

	// Update all indicators
	ta.updateIndicators(symbolData)
}

// AddCandles adds multiple candles at once
func (ta *TechnicalAnalyzer) AddCandles(candles []types.OHLCV) {
	for _, candle := range candles {
		ta.AddCandle(candle)
	}
}

// GetIndicatorValues returns current indicator values for a symbol
func (ta *TechnicalAnalyzer) GetIndicatorValues(symbol string) *IndicatorValues {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	symbolData, exists := ta.data[symbol]
	if !exists || len(symbolData.Candles) == 0 {
		return nil
	}

	return &IndicatorValues{
		Symbol:         symbol,
		CurrentPrice:   ta.getCurrentPrice(symbolData),
		SMA:            ta.getLastValue(symbolData.SMA),
		EMA:            ta.getLastValue(symbolData.EMA),
		RSI:            ta.getLastValue(symbolData.RSI),
		MACD:           ta.getLastValue(symbolData.MACD),
		MACDSignal:     ta.getLastValue(symbolData.MACDSignal),
		MACDHist:       ta.getLastValue(symbolData.MACDHist),
		ATR:            ta.getLastValue(symbolData.ATR),
		BollingerUpper: ta.getLastValue(symbolData.BollingerUpper),
		BollingerMiddle: ta.getLastValue(symbolData.BollingerMiddle),
		BollingerLower: ta.getLastValue(symbolData.BollingerLower),
		VolumeSMA:      ta.getLastValue(symbolData.VolumeSMA),
	}
}

// IndicatorValues represents current indicator values
type IndicatorValues struct {
	Symbol         string  `json:"symbol"`
	CurrentPrice   float64 `json:"current_price"`
	SMA            float64 `json:"sma"`
	EMA            float64 `json:"ema"`
	RSI            float64 `json:"rsi"`
	MACD           float64 `json:"macd"`
	MACDSignal     float64 `json:"macd_signal"`
	MACDHist       float64 `json:"macd_hist"`
	ATR            float64 `json:"atr"`
	BollingerUpper float64 `json:"bollinger_upper"`
	BollingerMiddle float64 `json:"bollinger_middle"`
	BollingerLower float64 `json:"bollinger_lower"`
	VolumeSMA      float64 `json:"volume_sma"`
}

// updateIndicators recalculates all indicators for a symbol
func (ta *TechnicalAnalyzer) updateIndicators(symbolData *SymbolData) {
	if len(symbolData.Candles) < 2 {
		return
	}

	// Extract price and volume arrays
	closes := ta.extractCloses(symbolData.Candles)
	highs := ta.extractHighs(symbolData.Candles)
	lows := ta.extractLows(symbolData.Candles)
	volumes := ta.extractVolumes(symbolData.Candles)

	// Update trend indicators
	symbolData.SMA = indicator.Sma(20, closes)
	symbolData.EMA = indicator.Ema(20, closes)

	// Update momentum indicators
	rsiValues, _ := indicator.Rsi(closes)
	symbolData.RSI = rsiValues
	macdLine, signalLine := indicator.Macd(closes)
	symbolData.MACD = macdLine
	symbolData.MACDSignal = signalLine
	// Calculate MACD histogram
	symbolData.MACDHist = make([]float64, len(macdLine))
	for i := range macdLine {
		if i < len(signalLine) {
			symbolData.MACDHist[i] = macdLine[i] - signalLine[i]
		}
	}

	// Update volatility indicators
	atrValues, _ := indicator.Atr(14, highs, lows, closes)
	symbolData.ATR = atrValues
	bbUpper, bbMiddle, bbLower := indicator.BollingerBands(closes)
	symbolData.BollingerUpper = bbUpper
	symbolData.BollingerMiddle = bbMiddle
	symbolData.BollingerLower = bbLower

	// Update volume indicators
	symbolData.VolumeSMA = indicator.Sma(20, volumes)
}

// Helper functions to extract data arrays
func (ta *TechnicalAnalyzer) extractCloses(candles []types.OHLCV) []float64 {
	closes := make([]float64, len(candles))
	for i, candle := range candles {
		closes[i] = candle.Close
	}
	return closes
}

func (ta *TechnicalAnalyzer) extractHighs(candles []types.OHLCV) []float64 {
	highs := make([]float64, len(candles))
	for i, candle := range candles {
		highs[i] = candle.High
	}
	return highs
}

func (ta *TechnicalAnalyzer) extractLows(candles []types.OHLCV) []float64 {
	lows := make([]float64, len(candles))
	for i, candle := range candles {
		lows[i] = candle.Low
	}
	return lows
}

func (ta *TechnicalAnalyzer) extractVolumes(candles []types.OHLCV) []float64 {
	volumes := make([]float64, len(candles))
	for i, candle := range candles {
		volumes[i] = candle.Volume
	}
	return volumes
}

// getCurrentPrice returns the most recent price
func (ta *TechnicalAnalyzer) getCurrentPrice(symbolData *SymbolData) float64 {
	if len(symbolData.Candles) == 0 {
		return 0
	}
	return symbolData.Candles[len(symbolData.Candles)-1].Close
}

// getLastValue returns the last non-zero value from an array
func (ta *TechnicalAnalyzer) getLastValue(values []float64) float64 {
	for i := len(values) - 1; i >= 0; i-- {
		if values[i] != 0 {
			return values[i]
		}
	}
	return 0
}

// GetHistoricalData returns historical OHLCV data for a symbol
func (ta *TechnicalAnalyzer) GetHistoricalData(symbol string, limit int) []types.OHLCV {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	symbolData, exists := ta.data[symbol]
	if !exists {
		return nil
	}

	if limit <= 0 || limit > len(symbolData.Candles) {
		limit = len(symbolData.Candles)
	}

	start := len(symbolData.Candles) - limit
	result := make([]types.OHLCV, limit)
	copy(result, symbolData.Candles[start:])
	return result
}

// GetSymbols returns list of all tracked symbols
func (ta *TechnicalAnalyzer) GetSymbols() []string {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	symbols := make([]string, 0, len(ta.data))
	for symbol := range ta.data {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// Clear removes all data for a symbol
func (ta *TechnicalAnalyzer) Clear(symbol string) {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	delete(ta.data, symbol)
}

// ClearAll removes all data
func (ta *TechnicalAnalyzer) ClearAll() {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	ta.data = make(map[string]*SymbolData)
}