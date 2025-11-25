package backtest

import (
	"aibot/internal/bot"
	"aibot/internal/config"
	"aibot/internal/data"
	"aibot/internal/indicators"
	"aibot/internal/logging"
	"aibot/internal/strategy"
	"aibot/internal/types"
	"aibot/pkg/stream"
	"aibot/pkg/trading"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Engine represents the backtesting engine
type Engine struct {
	// Configuration
	config        *BacktestConfig
	logger        *logging.Logger

	// Data
	historicalData  []types.OHLCV
	symbols        []string

	// Components
	candleAggregator  *data.CandleAggregator
	technicalAnalyzer *indicators.TechnicalAnalyzer
	streamProvider    *BacktestStreamProvider
	tradingExecutor   *BacktestTradingExecutor
	orchestrator      *bot.Orchestrator

	// Results
	results         *BacktestResults
	progress        *ProgressTracker

	// State
	currentIndex    int
	isRunning       bool
	mu              sync.RWMutex
}

// BacktestConfig holds backtesting configuration
type BacktestConfig struct {
	// Data settings
	DataDirectory     string    `json:"data_directory"`
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	Symbols           []string  `json:"symbols"`
	Timeframe         string    `json:"timeframe"`

	// Initial conditions
	InitialBalance    float64   `json:"initial_balance"`
	InitialLeverage   float64   `json:"initial_leverage"`

	// Trading simulation
	Commission        float64   `json:"commission"`        // Per trade commission
	Slippage          float64   `json:"slippage"`          // Price slippage
	Latency           time.Duration `json:"latency"`       // Execution latency
	FillProbability   float64   `json:"fill_probability"`  // Order fill probability

	// Output settings
	ResultsDirectory  string    `json:"results_directory"`
	DetailedReports   bool      `json:"detailed_reports"`
	GenerateCharts    bool      `json:"generate_charts"`
	ExportTrades      bool      `json:"export_trades"`
	ExportPerformance bool      `json:"export_performance"`

	// Progress tracking
	ProgressInterval  time.Duration `json:"progress_interval"`

	// Advanced settings
	EnableOptimization bool     `json:"enable_optimization"`
	OptimizationParams map[string][]float64 `json:"optimization_params"`
}

// BacktestResults contains comprehensive backtesting results
type BacktestResults struct {
	// Metadata
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	Duration          time.Duration `json:"duration"`
	TotalTicks        int       `json:"total_ticks"`
	ProcessedTicks    int       `json:"processed_ticks"`

	// Performance metrics
	InitialBalance    float64   `json:"initial_balance"`
	FinalBalance      float64   `json:"final_balance"`
	TotalReturn       float64   `json:"total_return"`
	TotalReturnPercent float64  `json:"total_return_percent"`
	MaxDrawdown       float64   `json:"max_drawdown"`
	MaxDrawdownPercent float64  `json:"max_drawdown_percent"`
	SharpeRatio       float64   `json:"sharpe_ratio"`
	SortinoRatio      float64   `json:"sortino_ratio"`
	ProfitFactor      float64   `json:"profit_factor"`
	WinRate           float64   `json:"win_rate"`

	// Trading statistics
	TotalTrades       int64     `json:"total_trades"`
	WinningTrades     int64     `json:"winning_trades"`
	LosingTrades      int64     `json:"losing_trades"`
	AvgWin            float64   `json:"avg_win"`
	AvgLoss           float64   `json:"avg_loss"`
	LargestWin        float64   `json:"largest_win"`
	LargestLoss       float64   `json:"largest_loss"`
	AvgTradeDuration  time.Duration `json:"avg_trade_duration"`

	// Strategy-specific metrics
	GridTrades        int64     `json:"grid_trades"`
	BreakoutTrades    int64     `json:"breakout_trades"`
	FalseBreakouts    int64     `json:"false_breakouts"`
	SuccessfulGrids   int64     `json:"successful_grids"`
	FailedGrids       int64     `json:"failed_grids"`

	// Risk metrics
	Volatility        float64   `json:"volatility"`
	ValueAtRisk       float64   `json:"value_at_risk"`
	ExpectedShortfall float64   `json:"expected_shortfall"`
	MarginCalls       int       `json:"margin_calls"`
	LeverageStats     LeverageStats `json:"leverage_stats"`

	// Trade history
	Trades            []TradeRecord `json:"trades"`
	PerformanceSeries []PerformancePoint `json:"performance_series"`
	EquityCurve       []EquityPoint `json:"equity_curve"`

	// Detailed analysis
	DailyReturns      []DailyReturn `json:"daily_returns"`
	MonthlyStats     []MonthlyStats `json:"monthly_stats"`
	DrawdownPeriods  []DrawdownPeriod `json:"drawdown_periods"`

	// Optimization results (if enabled)
	OptimizationResults *OptimizationResults `json:"optimization_results,omitempty"`
}

// TradeRecord represents a single trade
type TradeRecord struct {
	Timestamp     time.Time `json:"timestamp"`
	Symbol        string    `json:"symbol"`
	Strategy      string    `json:"strategy"`
	Action        string    `json:"action"`        // "buy", "sell", "long", "short"
	Quantity      float64   `json:"quantity"`
	Price         float64   `json:"price"`
	Commission    float64   `json:"commission"`
	Slippage      float64   `json:"slippage"`
	PnL           float64   `json:"pnl"`
	Reason        string    `json:"reason"`
}

// PerformancePoint represents performance at a specific time
type PerformancePoint struct {
	Timestamp     time.Time `json:"timestamp"`
	Balance       float64   `json:"balance"`
	PnL           float64   `json:"pnl"`
	Drawdown      float64   `json:"drawdown"`
	PositionSize  float64   `json:"position_size"`
	OpenOrders    int       `json:"open_orders"`
	WinRate       float64   `json:"win_rate"`
}

// EquityPoint represents equity curve point
type EquityPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	Equity        float64   `json:"equity"`
	Drawdown      float64   `json:"drawdown"`
	PeakEquity    float64   `json:"peak_equity"`
}

// LeverageStats contains leverage usage statistics
type LeverageStats struct {
	AvgLeverage     float64 `json:"avg_leverage"`
	MaxLeverage     float64 `json:"max_leverage"`
	LeverageHours   float64 `json:"leverage_hours"`
	MarginCallCount int     `json:"margin_call_count"`
}

// DailyReturn represents daily performance
type DailyReturn struct {
	Date     time.Time `json:"date"`
	Return   float64   `json:"return"`
	Balance  float64   `json:"balance"`
	Trades   int       `json:"trades"`
}

// MonthlyStats represents monthly performance statistics
type MonthlyStats struct {
	Month       time.Month `json:"month"`
	Year        int        `json:"year"`
	Return      float64    `json:"return"`
	Volatility  float64    `json:"volatility"`
	MaxDrawdown float64    `json:"max_drawdown"`
	Trades      int        `json:"trades"`
	WinRate     float64    `json:"win_rate"`
}

// DrawdownPeriod represents a drawdown period
type DrawdownPeriod struct {
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	Duration    time.Duration `json:"duration"`
	Depth       float64       `json:"depth"`
	Recovery    bool          `json:"recovery"`
}

// OptimizationResults contains optimization results
type OptimizationResults struct {
	BestParameters  map[string]float64 `json:"best_parameters"`
	BestReturn      float64            `json:"best_return"`
	BestSharpe      float64            `json:"best_sharpe"`
	AllRuns         []OptimizationRun  `json:"all_runs"`
	Iterations      int                `json:"iterations"`
	TotalRuntime    time.Duration      `json:"total_runtime"`
}

// OptimizationRun represents a single optimization run
type OptimizationRun struct {
	Parameters map[string]float64 `json:"parameters"`
	Return     float64            `json:"return"`
	Sharpe     float64            `json:"sharpe"`
	Drawdown   float64            `json:"drawdown"`
	Trades     int64              `json:"trades"`
}

// ProgressTracker tracks backtesting progress
type ProgressTracker struct {
	StartTime      time.Time `json:"start_time"`
	CurrentIndex   int       `json:"current_index"`
	TotalTicks     int       `json:"total_ticks"`
	ProcessedTicks int       `json:"processed_ticks"`
	PercentDone    float64   `json:"percent_done"`
	EstimatedTime  time.Duration `json:"estimated_time"`
	TicksPerSecond float64   `json:"ticks_per_second"`
	mu             sync.RWMutex
}

// BacktestStreamProvider provides historical data for backtesting
type BacktestStreamProvider struct {
	data         []types.OHLCV
	currentIndex int
	tickerChan   chan *types.Ticker
	ohlcvChan    chan *types.OHLCV
	config       BacktestConfig
	mu           sync.Mutex
}

// BacktestTradingExecutor simulates trading for backtesting
type BacktestTradingExecutor struct {
	balance      float64
	positions    map[string]*BacktestPosition
	orders       map[string]*BacktestOrder
	trades       []TradeRecord
	config       BacktestConfig
	mu           sync.Mutex
	orderID      int64
}

// BacktestPosition represents a position in backtesting
type BacktestPosition struct {
	Symbol      string    `json:"symbol"`
	Size        float64   `json:"size"`
	EntryPrice  float64   `json:"entry_price"`
	OpenTime    time.Time `json:"open_time"`
	UnrealizedPnL float64 `json:"unrealized_pnl"`
	RealizedPnL   float64 `json:"realized_pnl"`
}

// BacktestOrder represents an order in backtesting
type BacktestOrder struct {
	ID          string    `json:"id"`
	Symbol      string    `json:"symbol"`
	Type        string    `json:"type"`
	Side        string    `json:"side"`
	Quantity    float64   `json:"quantity"`
	Price       float64   `json:"price"`
	FilledQty   float64   `json:"filled_qty"`
	FilledPrice float64   `json:"filled_price"`
	Status      string    `json:"status"`
	CreateTime  time.Time `json:"create_time"`
	FillTime    time.Time `json:"fill_time"`
}

// NewEngine creates a new backtesting engine
func NewEngine(cfg config.BacktestConfig, logger *logging.Logger) (*Engine, error) {
	// Convert config
	backtestConfig := BacktestConfig{
		DataDirectory:     cfg.DataDirectory,
		StartTime:         cfg.StartTime,
		EndTime:           cfg.EndTime,
		InitialBalance:    cfg.InitialBalance,
		Commission:        cfg.Commission,
		Slippage:          cfg.Slippage,
		Latency:           cfg.Latency,
		ResultsDirectory:  cfg.ResultsDirectory,
		DetailedReports:   cfg.DetailedReports,
		GenerateCharts:    cfg.GenerateCharts,
		ExportTrades:      cfg.ExportTrades,
		ExportPerformance: cfg.ExportPerformance,
		ProgressInterval:  1 * time.Second,
	}

	engine := &Engine{
		config:   &backtestConfig,
		logger:   logger,
		results:  &BacktestResults{
			StartTime: backtestConfig.StartTime,
			EndTime:   backtestConfig.EndTime,
		},
		progress: &ProgressTracker{
			StartTime: time.Now(),
		},
	}

	// Initialize components
	if err := engine.initializeComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	return engine, nil
}

// initializeComponents initializes backtesting components
func (e *Engine) initializeComponents() error {
	// Load historical data
	if err := e.loadHistoricalData(); err != nil {
		return fmt.Errorf("failed to load historical data: %w", err)
	}

	// Initialize candle aggregator
	e.candleAggregator = data.NewCandleAggregator(data.AggregatorConfig{
		BaseInterval: 300 * time.Millisecond,
		MaxHistory:   100,
		Timeframes:   []data.CandleTimeframe{data.Timeframe1s, data.Timeframe3s, data.Timeframe15s},
		Symbols:      e.symbols,
	})

	// Initialize technical analyzer
	e.technicalAnalyzer = indicators.NewTechnicalAnalyzer(indicators.AnalyzerConfig{
		MaxHistoryCandles: 100,
	})

	// Initialize stream provider
	e.streamProvider = NewBacktestStreamProvider(e.historicalData, e.config)

	// Initialize trading executor
	e.tradingExecutor = NewBacktestTradingExecutor(e.config)

	e.progress.TotalTicks = len(e.historicalData)

	return nil
}

// loadHistoricalData loads historical OHLCV data from CSV files
func (e *Engine) loadHistoricalData() error {
	if e.config.DataDirectory == "" {
		return fmt.Errorf("data directory must be specified for backtesting")
	}

	if len(e.config.Symbols) == 0 {
		e.config.Symbols = []string{"BTCUSDT"} // Default symbol
	}

	var allData []types.OHLCV
	var loadedSymbols []string

	// Load data for each symbol
	for _, symbol := range e.config.Symbols {
		symbolData, err := e.loadSymbolData(symbol)
		if err != nil {
			return fmt.Errorf("failed to load data for symbol %s: %w", symbol, err)
		}

		allData = append(allData, symbolData...)
		loadedSymbols = append(loadedSymbols, symbol)
	}

	if len(allData) == 0 {
		return fmt.Errorf("no historical data found for any symbols")
	}

	// Sort data by timestamp
	e.historicalData = allData
	e.symbols = loadedSymbols

	e.logger.Infof("Loaded %d candles for %d symbols", len(e.historicalData), len(e.symbols))
	return nil
}

// loadSymbolData loads data for a specific symbol
func (e *Engine) loadSymbolData(symbol string) ([]types.OHLCV, error) {
	// Try multiple CSV filename formats
	csvFiles := []string{
		filepath.Join(e.config.DataDirectory, symbol+".csv"),
		filepath.Join(e.config.DataDirectory, strings.ToLower(symbol)+".csv"),
		filepath.Join(e.config.DataDirectory, strings.ToUpper(symbol)+".csv"),
		filepath.Join(e.config.DataDirectory, symbol+"_"+e.config.Timeframe+".csv"),
	}

	var file *os.File
	var err error

	// Try each possible filename
	for _, csvFile := range csvFiles {
		file, err = os.Open(csvFile)
		if err == nil {
			e.logger.Infof("Loading data from: %s", csvFile)
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("CSV file not found for symbol %s (tried: %v)", symbol, csvFiles)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read and validate header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Validate required columns
	requiredColumns := []string{"timestamp", "open", "high", "low", "close", "volume"}
	if !e.validateCSVHeader(header, requiredColumns) {
		return nil, fmt.Errorf("invalid CSV header format. Required columns: %v", requiredColumns)
	}

	var data []types.OHLCV
	lineNumber := 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV record at line %d: %w", lineNumber, err)
		}
		lineNumber++

		// Skip empty records
		if len(record) < len(requiredColumns) {
			continue
		}

		// Parse CSV record
		candle, err := e.parseCSVRecord(record, symbol, requiredColumns)
		if err != nil {
			e.logger.Warnf("Skipping line %d due to parse error: %v", lineNumber, err)
			continue
		}

		// Filter by date range
		if !candle.Timestamp.Before(e.config.StartTime) && !candle.Timestamp.After(e.config.EndTime) {
			data = append(data, candle)
		}
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no data found for symbol %s in specified date range %s to %s",
			symbol, e.config.StartTime.Format("2006-01-02"), e.config.EndTime.Format("2006-01-02"))
	}

	e.logger.Infof("Loaded %d candles for %s from %s to %s",
		len(data), symbol,
		data[0].Timestamp.Format(time.RFC3339),
		data[len(data)-1].Timestamp.Format(time.RFC3339))

	return data, nil
}

// validateCSVHeader validates that CSV contains required columns
func (e *Engine) validateCSVHeader(header []string, required []string) bool {
	if len(header) < len(required) {
		return false
	}

	headerLower := make([]string, len(header))
	for i, h := range header {
		headerLower[i] = strings.ToLower(strings.TrimSpace(h))
	}

	for _, req := range required {
		found := false
		for _, h := range headerLower {
			if h == req {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// parseCSVRecord parses a single CSV record into OHLCV data
func (e *Engine) parseCSVRecord(record []string, symbol string, columns []string) (types.OHLCV, error) {
	var timestamp time.Time
	var open, high, low, close, volume float64
	var err error

	// Try different timestamp formats
	timestampStr := record[0]
	timestampFormats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05.000",
		time.RFC3339,
		"2006/01/02 15:04:05",
	}

	for _, format := range timestampFormats {
		timestamp, err = time.Parse(format, timestampStr)
		if err == nil {
			break
		}
	}

	if err != nil {
		return types.OHLCV{}, fmt.Errorf("invalid timestamp format: %s", timestampStr)
	}

	open, err = parseFloat(record[1])
	if err != nil {
		return types.OHLCV{}, fmt.Errorf("invalid open price: %s", record[1])
	}

	high, err = parseFloat(record[2])
	if err != nil {
		return types.OHLCV{}, fmt.Errorf("invalid high price: %s", record[2])
	}

	low, err = parseFloat(record[3])
	if err != nil {
		return types.OHLCV{}, fmt.Errorf("invalid low price: %s", record[3])
	}

	close, err = parseFloat(record[4])
	if err != nil {
		return types.OHLCV{}, fmt.Errorf("invalid close price: %s", record[4])
	}

	volume, err = parseFloat(record[5])
	if err != nil {
		return types.OHLCV{}, fmt.Errorf("invalid volume: %s", record[5])
	}

	// Validate OHLC relationships
	if high < low || high < open || high < close || low > open || low > close {
		return types.OHLCV{}, fmt.Errorf("invalid OHLC relationships: O=%.2f H=%.2f L=%.2f C=%.2f", open, high, low, close)
	}

	return types.OHLCV{
		Symbol:    symbol,
		Timestamp: timestamp,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
	}, nil
}


// Run executes the backtest
func (e *Engine) Run() (*BacktestResults, error) {
	e.mu.Lock()
	if e.isRunning {
		e.mu.Unlock()
		return nil, fmt.Errorf("backtest is already running")
	}
	e.isRunning = true
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.isRunning = false
		e.mu.Unlock()
	}()

	e.logger.Infof("Starting backtest from %s to %s",
		e.config.StartTime.Format(time.RFC3339),
		e.config.EndTime.Format(time.RFC3339))

	// Initialize results
	e.results.StartTime = time.Now()

	// Create bot configuration
	botConfig := e.createBotConfig()

	// Create orchestrator
	orchestrator, err := bot.NewOrchestrator(botConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator: %w", err)
	}
	e.orchestrator = orchestrator

	// Start orchestrator
	if err := orchestrator.Start(e.streamProvider, e.tradingExecutor); err != nil {
		return nil, fmt.Errorf("failed to start orchestrator: %w", err)
	}

	// Start progress reporter
	progressDone := make(chan struct{})
	go e.progressReporter(progressDone)

	// Process historical data
	e.processHistoricalData()

	// Stop progress reporter
	close(progressDone)

	// Stop orchestrator
	if err := orchestrator.Stop(); err != nil {
		e.logger.Errorf("Error stopping orchestrator: %v", err)
	}

	// Calculate final results
	e.calculateFinalResults()

	e.results.Duration = time.Since(e.results.StartTime)

	e.logger.Infof("Backtest completed in %v", e.results.Duration)
	e.logger.Infof("Final balance: $%.2f (Return: %.2f%%)",
		e.results.FinalBalance, e.results.TotalReturnPercent)

	return e.results, nil
}

// processHistoricalData processes the historical data
func (e *Engine) processHistoricalData() {
	for i, candle := range e.historicalData {
		e.currentIndex = i

		// Update progress
		e.updateProgress(i + 1)

		// Send data to stream provider
		e.streamProvider.AddCandle(candle)

		// Small delay to simulate real-time processing
		time.Sleep(1 * time.Millisecond)

		// Check for interrupt (in real implementation)
		// if e.shouldStop() {
		//     break
		// }
	}

	e.progress.ProcessedTicks = len(e.historicalData)
	e.progress.PercentDone = 100.0
}

// updateProgress updates progress tracking
func (e *Engine) updateProgress(processed int) {
	e.progress.mu.Lock()
	defer e.progress.mu.Unlock()

	e.progress.CurrentIndex = processed
	e.progress.ProcessedTicks = processed
	e.progress.PercentDone = float64(processed) / float64(e.progress.TotalTicks) * 100

	if processed > 0 {
		elapsed := time.Since(e.progress.StartTime)
		e.progress.TicksPerSecond = float64(processed) / elapsed.Seconds()

		remaining := float64(e.progress.TotalTicks-processed) / e.progress.TicksPerSecond
		e.progress.EstimatedTime = time.Duration(remaining) * time.Second
	}
}

// progressReporter reports progress periodically
func (e *Engine) progressReporter(done chan struct{}) {
	ticker := time.NewTicker(e.config.ProgressInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			e.progress.mu.RLock()
			e.logger.Infof("Progress: %.1f%% (%d/%d) - %.0f ticks/sec - ETA: %v",
				e.progress.PercentDone,
				e.progress.ProcessedTicks,
				e.progress.TotalTicks,
				e.progress.TicksPerSecond,
				e.progress.EstimatedTime.Round(time.Second))
			e.progress.mu.RUnlock()
		}
	}
}

// createBotConfig creates bot configuration from backtest config
func (e *Engine) createBotConfig() *bot.BotConfig {
	return &bot.BotConfig{
		InitialBalance:    e.config.InitialBalance,
		MaxSymbols:        1,
		DefaultSymbol:     "BTCUSDT",
		// Add other configuration parameters as needed
	}
}

// calculateFinalResults calculates final backtesting results
func (e *Engine) calculateFinalResults() {
	e.results.ProcessedTicks = len(e.historicalData)
	e.results.InitialBalance = e.config.InitialBalance

	// Get final balance from trading executor
	e.results.FinalBalance = e.tradingExecutor.GetBalance()
	e.results.TotalReturn = e.results.FinalBalance - e.results.InitialBalance
	e.results.TotalReturnPercent = (e.results.TotalReturn / e.results.InitialBalance) * 100

	// Get trades from executor
	e.results.Trades = e.tradingExecutor.GetTrades()
	e.results.TotalTrades = int64(len(e.results.Trades))

	// Calculate performance metrics
	e.calculatePerformanceMetrics()

	// Generate equity curve
	e.generateEquityCurve()
}

// calculatePerformanceMetrics calculates performance metrics
func (e *Engine) calculatePerformanceMetrics() {
	if len(e.results.Trades) == 0 {
		return
	}

	var totalWin, totalLoss float64
	var winCount, lossCount int64
	var totalDuration time.Duration
	var largestWin, largestLoss float64

	for _, trade := range e.results.Trades {
		if trade.PnL > 0 {
			totalWin += trade.PnL
			winCount++
			if trade.PnL > largestWin {
				largestWin = trade.PnL
			}
		} else {
			totalLoss += math.Abs(trade.PnL)
			lossCount++
			if trade.PnL < largestLoss {
				largestLoss = trade.PnL
			}
		}
	}

	e.results.WinningTrades = winCount
	e.results.LosingTrades = lossCount
	e.results.WinRate = float64(winCount) / float64(e.results.TotalTrades) * 100
	e.results.LargestWin = largestWin
	e.results.LargestLoss = largestLoss

	if winCount > 0 {
		e.results.AvgWin = totalWin / float64(winCount)
	}
	if lossCount > 0 {
		e.results.AvgLoss = totalLoss / float64(lossCount)
	}

	if totalLoss > 0 {
		e.results.ProfitFactor = totalWin / totalLoss
	}

	if e.results.TotalTrades > 0 {
		e.results.AvgTradeDuration = totalDuration / time.Duration(e.results.TotalTrades)
	}

	// Calculate Sharpe ratio (simplified)
	e.calculateSharpeRatio()
}

// calculateSharpeRatio calculates Sharpe ratio
func (e *Engine) calculateSharpeRatio() {
	if len(e.results.Trades) < 2 {
		return
	}

	// Calculate daily returns
	returns := make([]float64, len(e.results.Trades))
	for i, trade := range e.results.Trades {
		if i == 0 {
			returns[i] = trade.PnL / e.config.InitialBalance
		} else {
			returns[i] = trade.PnL / e.config.InitialBalance
		}
	}

	// Calculate average return and standard deviation
	avgReturn := average(returns)
	variance := 0.0
	for _, ret := range returns {
		variance += math.Pow(ret-avgReturn, 2)
	}
	variance /= float64(len(returns))
	stdDev := math.Sqrt(variance)

	if stdDev > 0 {
		// Annualized Sharpe ratio (assuming daily returns)
		e.results.SharpeRatio = (avgReturn / stdDev) * math.Sqrt(252)
	}
}

// generateEquityCurve generates equity curve points
func (e *Engine) generateEquityCurve() {
	balance := e.config.InitialBalance
	peak := balance

	// Add initial point
	e.results.EquityCurve = append(e.results.EquityCurve, EquityPoint{
		Timestamp:  e.config.StartTime,
		Equity:     balance,
		Drawdown:   0,
		PeakEquity: peak,
	})

	// Process trades chronologically
	for _, trade := range e.results.Trades {
		balance += trade.PnL
		if balance > peak {
			peak = balance
		}
		drawdown := (peak - balance) / peak * 100

		point := EquityPoint{
			Timestamp:  trade.Timestamp,
			Equity:     balance,
			Drawdown:   drawdown,
			PeakEquity: peak,
		}
		e.results.EquityCurve = append(e.results.EquityCurve, point)
	}

	// Calculate max drawdown
	maxDrawdown := 0.0
	for _, point := range e.results.EquityCurve {
		if point.Drawdown > maxDrawdown {
			maxDrawdown = point.Drawdown
		}
	}
	e.results.MaxDrawdown = maxDrawdown
	e.results.MaxDrawdownPercent = maxDrawdown
}

// SaveResults saves backtesting results to files
func (e *Engine) SaveResults() error {
	if err := os.MkdirAll(e.config.ResultsDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	baseName := fmt.Sprintf("backtest_%s", timestamp)

	// Save JSON results
	if err := e.saveJSONResults(filepath.Join(e.config.ResultsDirectory, baseName+".json")); err != nil {
		return err
	}

	// Export trades if enabled
	if e.config.ExportTrades {
		if err := e.exportTrades(filepath.Join(e.config.ResultsDirectory, baseName+"_trades.csv")); err != nil {
			return err
		}
	}

	// Export performance if enabled
	if e.config.ExportPerformance {
		if err := e.exportPerformance(filepath.Join(e.config.ResultsDirectory, baseName+"_performance.csv")); err != nil {
			return err
		}
	}

	e.logger.Infof("Results saved to %s", e.config.ResultsDirectory)
	return nil
}

// saveJSONResults saves results as JSON
func (e *Engine) saveJSONResults(filename string) error {
	data, err := json.MarshalIndent(e.results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	return ioutil.WriteFile(filename, data, 0644)
}

// exportTrades exports trades to CSV
func (e *Engine) exportTrades(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create trades file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Timestamp", "Symbol", "Strategy", "Action", "Quantity", "Price", "Commission", "PnL", "Reason"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write trades
	for _, trade := range e.results.Trades {
		record := []string{
			trade.Timestamp.Format(time.RFC3339),
			trade.Symbol,
			trade.Strategy,
			trade.Action,
			fmt.Sprintf("%.6f", trade.Quantity),
			fmt.Sprintf("%.2f", trade.Price),
			fmt.Sprintf("%.4f", trade.Commission),
			fmt.Sprintf("%.4f", trade.PnL),
			trade.Reason,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// exportPerformance exports performance data to CSV
func (e *Engine) exportPerformance(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create performance file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Timestamp", "Equity", "Drawdown", "PeakEquity"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write equity curve
	for _, point := range e.results.EquityCurve {
		record := []string{
			point.Timestamp.Format(time.RFC3339),
			fmt.Sprintf("%.2f", point.Equity),
			fmt.Sprintf("%.2f", point.Drawdown),
			fmt.Sprintf("%.2f", point.PeakEquity),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// GetProgress returns current progress
func (e *Engine) GetProgress() ProgressTracker {
	e.progress.mu.RLock()
	defer e.progress.mu.RUnlock()
	return *e.progress
}

// IsRunning returns whether the backtest is currently running
func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isRunning
}

// Stop stops the backtesting
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.isRunning && e.orchestrator != nil {
		_ = e.orchestrator.Stop()
	}
}

// Helper functions
func parseFloat(s string) (float64, error) {
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	return result, err
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

