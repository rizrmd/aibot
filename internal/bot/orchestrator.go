package bot

import (
	"aibot/internal/data"
	"aibot/internal/indicators"
	"aibot/internal/strategy"
	"aibot/internal/types"
	"aibot/pkg/stream"
	"aibot/pkg/trading"
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

// TradingMode represents the current trading mode of the bot
type TradingMode string

const (
	ModeGrid      TradingMode = "grid"       // Grid trading mode
	ModeBreakout  TradingMode = "breakout"   // Breakout position management mode
	ModeRecovery  TradingMode = "recovery"   // False breakout recovery mode
	ModeStability TradingMode = "stability"  // Stability detection mode
	ModeIdle      TradingMode = "idle"       // Idle/waiting mode
)

// BotState represents the current state of the trading bot
type BotState struct {
	Mode               TradingMode    `json:"mode"`
	IsActive           bool           `json:"is_active"`
	CurrentSymbol      string         `json:"current_symbol"`
	GridBounds         strategy.GridBounds `json:"grid_bounds,omitempty"`
	BreakoutInfo       *BreakoutInfo `json:"breakout_info,omitempty"`
	LastUpdateTime     time.Time      `json:"last_update_time"`
	SessionStart       time.Time      `json:"session_start"`
	TradeCount         int            `json:"trade_count"`
	SuccessfulTrades   int            `json:"successful_trades"`
	TotalPnL           float64        `json:"total_pnl"`
	MaxDrawdown        float64        `json:"max_drawdown"`
	CurrentDrawdown    float64        `json:"current_drawdown"`
}

// BreakoutInfo contains information about current breakout handling
type BreakoutInfo struct {
	BreakoutType       strategy.BreakoutType    `json:"breakout_type"`
	BreakoutTime       time.Time               `json:"breakout_time"`
	EntryPrice         float64                 `json:"entry_price"`
	ConfirmationCandles int                    `json:"confirmation_candles"`
	IsConfirmed        bool                    `json:"is_confirmed"`
	FalseBreakoutDetected bool                 `json:"false_breakout_detected"`
	RecoveryAction     string                  `json:"recovery_action"`
	StabilityWaitStart *time.Time              `json:"stability_wait_start,omitempty"`
}

// Orchestrator manages the entire trading bot coordination
type Orchestrator struct {
	// Core components
	streamProvider    stream.StreamProvider
	tradingExecutor   trading.TradingExecutor
	candleAggregator  *data.CandleAggregator
	technicalAnalyzer *indicators.TechnicalAnalyzer

	// Strategy components
	gridSetup        *strategy.GridSetup
	gridCalculator   *strategy.GridCalculator
	breakoutDetector *strategy.BreakoutDetector
	falseBreakoutDetector *strategy.FalseBreakoutDetector
	positionManager  *strategy.PositionManager
	stabilityDetector *strategy.PriceStabilityDetector
	riskManager      *strategy.RiskManager

	// Configuration
	config           *BotConfig
	symbols          []string
	activeSymbol     string

	// State management
	state            BotState
	mu               sync.RWMutex
	modeTransitions  map[TradingMode][]TradingMode // Allowed mode transitions

	// Event channels
	dataChan         chan DataUpdate
	signalChan       chan TradingSignal
	riskChan         chan RiskAlert
	controlChan      chan ControlCommand

	// Performance tracking
	performance      PerformanceMetrics

	// Context and shutdown
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
}

// DataUpdate represents incoming data update
type DataUpdate struct {
	Symbol   string
	Ticker   *types.Ticker
	OHLCV    *types.OHLCV
	Time     time.Time
}

// TradingSignal represents a trading signal from any strategy component
type TradingSignal struct {
	Type         string      `json:"type"`         // "grid_setup", "breakout", "false_breakout", "stability"
	Symbol       string      `json:"symbol"`
	Action       string      `json:"action"`       // "buy", "sell", "close", "setup_grid"
	Price        float64     `json:"price"`
	Quantity     float64     `json:"quantity"`
	Confidence   float64     `json:"confidence"`
	Reason       string      `json:"reason"`
	Data         interface{} `json:"data,omitempty"`
	Timestamp    time.Time   `json:"timestamp"`
}

// RiskAlert represents a risk management alert
type RiskAlert struct {
	Level       string    `json:"level"`       // "info", "warning", "critical"
	Type        string    `json:"type"`        // "margin", "drawdown", "concentration", "correlation"
	Message     string    `json:"message"`
	Symbol      string    `json:"symbol,omitempty"`
	Value       float64   `json:"value,omitempty"`
	Threshold   float64   `json:"threshold,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// ControlCommand represents a control command to the orchestrator
type ControlCommand struct {
	Type    string      `json:"type"`    // "start", "stop", "pause", "resume", "switch_mode"
	Payload interface{} `json:"payload,omitempty"`
}

// PerformanceMetrics tracks bot performance
type PerformanceMetrics struct {
	TotalTrades         int64     `json:"total_trades"`
	WinningTrades       int64     `json:"winning_trades"`
	LosingTrades        int64     `json:"losing_trades"`
	TotalPnL            float64   `json:"total_pnl"`
	MaxDrawdown         float64   `json:"max_drawdown"`
	CurrentDrawdown     float64   `json:"current_drawdown"`
	SharpeRatio         float64   `json:"sharpe_ratio"`
	ProfitFactor        float64   `json:"profit_factor"`
	WinRate             float64   `json:"win_rate"`
	AvgTradeDuration    time.Duration `json:"avg_trade_duration"`
	SessionStart        time.Time `json:"session_start"`
	LastTradeTime       time.Time `json:"last_trade_time"`
}

// BotConfig holds configuration for the trading bot
type BotConfig struct {
	// Trading parameters
	InitialBalance      float64  `json:"initial_balance"`
	MaxSymbols          int      `json:"max_symbols"`
	DefaultSymbol       string   `json:"default_symbol"`

	// Strategy parameters
	GridSetupConfig     strategy.GridSetupConfig   `json:"grid_setup_config"`
	BreakoutConfig      strategy.BreakoutConfig    `json:"breakout_config"`
	FalseBreakoutConfig strategy.FalseBreakoutConfig `json:"false_breakout_config"`
	StabilityConfig     strategy.StabilityConfig   `json:"stability_config"`
	RiskManagerConfig   strategy.RiskManagerConfig `json:"risk_manager_config"`

	// Stream and trading config
	StreamConfig        stream.StreamConfig        `json:"stream_config"`
	TradingConfig       trading.ExecutionConfig    `json:"trading_config"`

	// Operational parameters
	UpdateInterval      time.Duration `json:"update_interval"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`

	// Safety parameters
	MaxDailyLoss        float64 `json:"max_daily_loss"`
	MaxConsecutiveLosses int     `json:"max_consecutive_losses"`
}

// NewOrchestrator creates a new trading bot orchestrator
func NewOrchestrator(config *BotConfig) (*Orchestrator, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create core components
	candleAggregator := data.NewCandleAggregator(data.AggregatorConfig{
		BaseInterval: 300 * time.Millisecond,
		MaxHistory:   100,
		Timeframes:   []data.CandleTimeframe{data.Timeframe1s, data.Timeframe3s, data.Timeframe15s},
		Symbols:      []string{config.DefaultSymbol},
	})

	technicalAnalyzer := indicators.NewTechnicalAnalyzer(indicators.AnalyzerConfig{
		MaxHistoryCandles: 100,
	})

	// Create strategy components
	gridSetup := strategy.NewGridSetup(candleAggregator, technicalAnalyzer, config.GridSetupConfig)
	gridCalculator := strategy.NewGridCalculator()
	breakoutDetector := strategy.NewBreakoutDetector(
		config.BreakoutConfig,
		candleAggregator,
		technicalAnalyzer,
	)
	falseBreakoutDetector := strategy.NewFalseBreakoutDetector(config.FalseBreakoutConfig)
	stabilityDetector := strategy.NewPriceStabilityDetector(
		config.StabilityConfig,
		technicalAnalyzer,
		candleAggregator,
	)
	riskManager := strategy.NewRiskManager(config.RiskManagerConfig, config.InitialBalance)

	orchestrator := &Orchestrator{
		candleAggregator:        candleAggregator,
		technicalAnalyzer:      technicalAnalyzer,
		gridSetup:              gridSetup,
		gridCalculator:         gridCalculator,
		breakoutDetector:       breakoutDetector,
		falseBreakoutDetector:  falseBreakoutDetector,
		stabilityDetector:      stabilityDetector,
		riskManager:            riskManager,
		config:                 config,
		symbols:                []string{config.DefaultSymbol},
		activeSymbol:           config.DefaultSymbol,
		state: BotState{
			Mode:         ModeIdle,
			IsActive:     false,
			SessionStart: time.Now(),
		},
		modeTransitions: map[TradingMode][]TradingMode{
			ModeIdle:      {ModeGrid},
			ModeGrid:      {ModeBreakout, ModeRecovery},
			ModeBreakout:  {ModeStability, ModeRecovery, ModeGrid},
			ModeStability: {ModeGrid, ModeBreakout},
			ModeRecovery:  {ModeGrid},
		},
		dataChan:    make(chan DataUpdate, 100),
		signalChan:  make(chan TradingSignal, 50),
		riskChan:    make(chan RiskAlert, 50),
		controlChan: make(chan ControlCommand, 10),
		ctx:         ctx,
		cancel:      cancel,
	}

	return orchestrator, nil
}

// Start starts the trading bot orchestrator
func (o *Orchestrator) Start(streamProvider stream.StreamProvider, tradingExecutor trading.TradingExecutor) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.state.IsActive {
		return fmt.Errorf("orchestrator is already active")
	}

	o.streamProvider = streamProvider
	o.tradingExecutor = tradingExecutor

	// Start data streaming
	if err := o.startDataStreaming(); err != nil {
		return fmt.Errorf("failed to start data streaming: %w", err)
	}

	// Start orchestrator workers
	o.startWorkers()

	// Start in idle mode - will switch to grid after receiving first price data
	o.state.Mode = ModeIdle
	o.state.IsActive = true
	o.state.SessionStart = time.Now()
	o.performance.SessionStart = time.Now()

	log.Printf("🚀 Trading bot orchestrator started for symbol: %s (waiting for price data)", o.activeSymbol)

	// Start a goroutine to initialize grid trading after receiving first price data
	o.wg.Add(1)
	go o.waitForPriceAndInitializeGrid()

	return nil
}

// Stop stops the trading bot orchestrator
func (o *Orchestrator) Stop() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.state.IsActive {
		return nil
	}

	log.Println("🔄 Starting orchestrator shutdown...")

	// Cancel context first to signal all goroutines to stop
	o.cancel()

	// Close all positions
	if err := o.closeAllPositions(); err != nil {
		log.Printf("Error closing positions: %v", err)
	}

	// Stop data streaming (this will close channels)
	if o.streamProvider != nil {
		o.streamProvider.Stop()
	}

	// Wait for workers to finish with a much shorter timeout
	done := make(chan struct{})
	go func() {
		o.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("✅ All workers completed gracefully")
	case <-time.After(3 * time.Second):
		log.Println("⚠️ Worker shutdown timeout reached, exiting immediately")
	}

	o.state.IsActive = false
	log.Println("🛑 Trading bot orchestrator stopped")

	return nil
}

// startDataStreaming starts the data streaming and processing
func (o *Orchestrator) startDataStreaming() error {
	// Start streaming for symbols
	if err := o.streamProvider.Start(o.ctx, o.symbols); err != nil {
		return err
	}

	// Start data processing worker
	o.wg.Add(1)
	go o.dataStreamingWorker()

	return nil
}

// startWorkers starts the orchestrator workers
func (o *Orchestrator) startWorkers() {
	// Signal processing worker
	o.wg.Add(1)
	go o.signalProcessingWorker()

	// Risk management worker
	o.wg.Add(1)
	go o.riskManagementWorker()

	// Mode management worker
	o.wg.Add(1)
	go o.modeManagementWorker()

	// Performance tracking worker
	o.wg.Add(1)
	go o.performanceWorker()

	// Control command worker
	o.wg.Add(1)
	go o.controlWorker()
}

// dataStreamingWorker processes incoming data from stream provider
func (o *Orchestrator) dataStreamingWorker() {
	defer o.wg.Done()

	tickerChan := o.streamProvider.GetTickerChannel()
	ohlcvChan := o.streamProvider.GetOHLCVChannel()

	for {
		select {
		case <-o.ctx.Done():
			return

		case ticker, ok := <-tickerChan:
			if !ok {
				// Channel closed
				return
			}
			if ticker.Symbol == o.activeSymbol {
				o.processTicker(&ticker)
			}

		case ohlcv, ok := <-ohlcvChan:
			if !ok {
				// Channel closed
				return
			}
			if ohlcv.Symbol == o.activeSymbol {
				o.processOHLCV(&ohlcv)
			}
		}
	}
}

// processTicker processes incoming ticker data
func (o *Orchestrator) processTicker(ticker *types.Ticker) {
	// Update candle aggregator
	o.candleAggregator.AddTick(*ticker)

	// Update technical analyzer with new ticker data
	o.technicalAnalyzer.AddCandle(types.OHLCV{
		Symbol:    ticker.Symbol,
		Timestamp: ticker.Timestamp,
		Open:      ticker.Price,
		High:      ticker.Price,
		Low:       ticker.Price,
		Close:     ticker.Price,
		Volume:    ticker.Volume,
	})

	// Send data update to processing pipeline with non-blocking send
	select {
	case o.dataChan <- DataUpdate{
		Symbol: ticker.Symbol,
		Ticker: ticker,
		Time:   ticker.Timestamp,
	}:
	case <-o.ctx.Done():
		// Context cancelled, don't block
		return
	}

	// Process based on current mode
	o.processDataInMode(ticker.Price, ticker.Timestamp)
}

// processOHLCV processes incoming OHLCV candle data
func (o *Orchestrator) processOHLCV(ohlcv *types.OHLCV) {
	// Add candle to technical analyzer
	o.technicalAnalyzer.AddCandle(*ohlcv)

	// Send data update to processing pipeline with non-blocking send
	select {
	case o.dataChan <- DataUpdate{
		Symbol: ohlcv.Symbol,
		OHLCV:  ohlcv,
		Time:   ohlcv.Timestamp,
	}:
	case <-o.ctx.Done():
		// Context cancelled, don't block
		return
	}
}

// processDataInMode processes data based on current trading mode
func (o *Orchestrator) processDataInMode(price float64, timestamp time.Time) {
	o.mu.Lock()
	currentMode := o.state.Mode
	o.mu.Unlock()

	switch currentMode {
	case ModeGrid:
		o.processGridMode(price, timestamp)
	case ModeBreakout:
		o.processBreakoutMode(price, timestamp)
	case ModeRecovery:
		o.processRecoveryMode(price, timestamp)
	case ModeStability:
		o.processStabilityMode(price, timestamp)
	case ModeIdle:
		// No processing in idle mode
	}
}

// processGridMode processes data in grid trading mode
func (o *Orchestrator) processGridMode(price float64, timestamp time.Time) {
	// Check for breakout conditions
	breakoutSignal := o.breakoutDetector.DetectBreakout(
		o.activeSymbol,
		o.state.GridBounds,
		price,
	)

	if breakoutSignal != nil {
		// Breakout detected - switch to breakout mode
		o.signalChan <- TradingSignal{
			Type:       "breakout",
			Symbol:     o.activeSymbol,
			Action:     "switch_mode",
			Confidence: breakoutSignal.Confidence,
			Reason:     fmt.Sprintf("Breakout detected: %s", breakoutSignal.Type),
			Data:       breakoutSignal,
			Timestamp:  timestamp,
		}
		return
	}

	// Check for false breakouts within grid bounds
	if o.state.BreakoutInfo != nil && o.state.BreakoutInfo.FalseBreakoutDetected {
		falseBreakoutSignal := o.falseBreakoutDetector.DetectFalseBreakout(
			o.activeSymbol,
			o.state.BreakoutInfo.EntryPrice,
			price,
			strategy.BreakoutType(o.state.BreakoutInfo.BreakoutType),
			0, // ATR - would get from technical analyzer
			0, // Average volume - would get from data
			timestamp.Sub(o.state.BreakoutInfo.BreakoutTime),
		)

		if falseBreakoutSignal != nil {
			o.signalChan <- TradingSignal{
				Type:       "false_breakout",
				Symbol:     o.activeSymbol,
				Action:     "recovery",
				Confidence: falseBreakoutSignal.Confidence,
				Reason:     falseBreakoutSignal.RecoveryAction,
				Data:       falseBreakoutSignal,
				Timestamp:  timestamp,
			}
		}
	}
}

// processBreakoutMode processes data in breakout mode
func (o *Orchestrator) processBreakoutMode(price float64, timestamp time.Time) {
	if o.state.BreakoutInfo == nil {
		o.switchMode(ModeGrid)
		return
	}

	// Check for false breakout
	falseBreakoutSignal := o.falseBreakoutDetector.DetectFalseBreakout(
		o.activeSymbol,
		o.state.BreakoutInfo.EntryPrice,
		price,
		strategy.BreakoutType(o.state.BreakoutInfo.BreakoutType),
		0, // ATR
		0, // Average volume
		timestamp.Sub(o.state.BreakoutInfo.BreakoutTime),
	)

	if falseBreakoutSignal != nil {
		o.signalChan <- TradingSignal{
			Type:       "false_breakout",
			Symbol:     o.activeSymbol,
			Action:     "switch_mode",
			Confidence: falseBreakoutSignal.Confidence,
			Reason:     "False breakout detected",
			Data:       falseBreakoutSignal,
			Timestamp:  timestamp,
		}
		return
	}

	// Check for price stability (for returning to grid)
	stabilitySignal := o.stabilityDetector.AnalyzeStability(o.activeSymbol, price)

	if stabilitySignal.IsStable && stabilitySignal.RecommendedAction == "Return to grid trading" {
		o.signalChan <- TradingSignal{
			Type:       "stability",
			Symbol:     o.activeSymbol,
			Action:     "switch_mode",
			Confidence: stabilitySignal.Confidence,
			Reason:     "Price stability detected, returning to grid",
			Data:       stabilitySignal,
			Timestamp:  timestamp,
		}
	}
}

// processRecoveryMode processes data in recovery mode
func (o *Orchestrator) processRecoveryMode(price float64, timestamp time.Time) {
	// In recovery mode, focus on minimizing losses and resetting
	// Check if conditions are suitable to return to grid trading
	stabilitySignal := o.stabilityDetector.AnalyzeStability(o.activeSymbol, price)

	if stabilitySignal.IsStable {
		o.signalChan <- TradingSignal{
			Type:       "recovery_complete",
			Symbol:     o.activeSymbol,
			Action:     "switch_mode",
			Confidence: stabilitySignal.Confidence,
			Reason:     "Recovery complete, returning to grid",
			Data:       stabilitySignal,
			Timestamp:  timestamp,
		}
	}
}

// processStabilityMode processes data in stability detection mode
func (o *Orchestrator) processStabilityMode(price float64, timestamp time.Time) {
	// Monitor stability and decide on next action
	stabilitySignal := o.stabilityDetector.AnalyzeStability(o.activeSymbol, price)

	if stabilitySignal.IsStable && stabilitySignal.RecommendedAction == "Return to grid trading" {
		o.signalChan <- TradingSignal{
			Type:       "stability_confirmed",
			Symbol:     o.activeSymbol,
			Action:     "switch_mode",
			Confidence: stabilitySignal.Confidence,
			Reason:     "Stability confirmed, returning to grid",
			Data:       stabilitySignal,
			Timestamp:  timestamp,
		}
	} else if !stabilitySignal.IsStable {
		// If stability is lost, might need to go back to breakout mode
		o.signalChan <- TradingSignal{
			Type:       "stability_lost",
			Symbol:     o.activeSymbol,
			Action:     "switch_mode",
			Confidence: stabilitySignal.Confidence,
			Reason:     "Stability lost, returning to breakout management",
			Data:       stabilitySignal,
			Timestamp:  timestamp,
		}
	}
}

// signalProcessingWorker processes trading signals
func (o *Orchestrator) signalProcessingWorker() {
	defer o.wg.Done()

	for {
		select {
		case <-o.ctx.Done():
			return

		case signal := <-o.signalChan:
			o.processTradingSignal(signal)
		}
	}
}

// processTradingSignal processes a trading signal
func (o *Orchestrator) processTradingSignal(signal TradingSignal) {
	switch signal.Type {
	case "breakout":
		o.handleBreakoutSignal(signal)
	case "false_breakout":
		o.handleFalseBreakoutSignal(signal)
	case "stability", "stability_confirmed":
		o.handleStabilitySignal(signal)
	case "grid_setup":
		o.handleGridSetupSignal(signal)
	}
}

// handleBreakoutSignal handles breakout detection signals
func (o *Orchestrator) handleBreakoutSignal(signal TradingSignal) {
	breakoutData := signal.Data.(*strategy.BreakoutSignal)

	// Update breakout info
	o.mu.Lock()
	o.state.BreakoutInfo = &BreakoutInfo{
		BreakoutType:        breakoutData.Type,
		BreakoutTime:        breakoutData.Timestamp,
		EntryPrice:          breakoutData.Price,
		ConfirmationCandles: 0,
		IsConfirmed:         false,
		FalseBreakoutDetected: false,
	}
	o.mu.Unlock()

	// Switch to breakout mode
	o.switchMode(ModeBreakout)

	log.Printf("🔥 Breakout detected: %s at %.2f (confidence: %.2f)",
		breakoutData.Type, breakoutData.Price, breakoutData.Confidence)
}

// handleFalseBreakoutSignal handles false breakout signals
func (o *Orchestrator) handleFalseBreakoutSignal(signal TradingSignal) {
	falseBreakoutData := signal.Data.(*strategy.FalseBreakoutSignal)

	// Update breakout info
	o.mu.Lock()
	if o.state.BreakoutInfo != nil {
		o.state.BreakoutInfo.FalseBreakoutDetected = true
		o.state.BreakoutInfo.RecoveryAction = falseBreakoutData.RecoveryAction
	}
	o.mu.Unlock()

	// Execute recovery action if specified
	o.executeRecoveryAction(falseBreakoutData.RecoveryAction, falseBreakoutData)

	// Switch to recovery mode
	o.switchMode(ModeRecovery)

	log.Printf("⚠️ False breakout detected: %s (confidence: %.2f)",
		falseBreakoutData.RecoveryAction, falseBreakoutData.Confidence)
}

// handleStabilitySignal handles price stability signals
func (o *Orchestrator) handleStabilitySignal(signal TradingSignal) {
	// Switch to grid mode
	o.switchMode(ModeGrid)

	log.Printf("✅ Price stability confirmed: %s", signal.Reason)
}

// handleGridSetupSignal handles grid setup signals
func (o *Orchestrator) handleGridSetupSignal(signal TradingSignal) {
	// Grid setup is handled during initialization
	log.Printf("🔧 Grid setup completed: %s", signal.Reason)
}

// executeRecoveryAction executes recovery actions for false breakouts
func (o *Orchestrator) executeRecoveryAction(action string, data interface{}) {
	// Get current position
	position, err := o.tradingExecutor.GetPosition(o.activeSymbol)
	if err != nil {
		log.Printf("Error getting position for recovery: %v", err)
		return
	}

	if position == nil {
		return // No position to recover
	}

	switch action {
	case "Close position and take profit":
		var err error
		if position.Size > 0 {
			_, err = o.tradingExecutor.CloseLong(o.activeSymbol, position.Size, 0)
		} else {
			_, err = o.tradingExecutor.CloseShort(o.activeSymbol, -position.Size, 0)
		}
		if err != nil {
			log.Printf("Error closing position for profit: %v", err)
		}

	case "Close position to minimize loss":
		var err error
		if position.Size > 0 {
			_, err = o.tradingExecutor.CloseLong(o.activeSymbol, position.Size, 0)
		} else {
			_, err = o.tradingExecutor.CloseShort(o.activeSymbol, -position.Size, 0)
		}
		if err != nil {
			log.Printf("Error closing position for loss: %v", err)
		}

	case "Consider taking opposite position":
		// Close current position first
		var err error
		if position.Size > 0 {
			_, err = o.tradingExecutor.CloseLong(o.activeSymbol, position.Size, 0)
		} else {
			_, err = o.tradingExecutor.CloseShort(o.activeSymbol, -position.Size, 0)
		}
		if err != nil {
			log.Printf("Error closing position before reversal: %v", err)
			return
		}

		// Calculate opposite position size
		oppositeSize := position.Size * 0.8 // 80% of original size as opposite position

		if position.Size > 0 {
			// Was long, now go short
			_, err = o.tradingExecutor.OpenShort(o.activeSymbol, oppositeSize, 0)
			if err != nil {
				log.Printf("Error opening short position: %v", err)
			}
		} else {
			// Was short, now go long
			_, err = o.tradingExecutor.OpenLong(o.activeSymbol, oppositeSize, 0)
			if err != nil {
				log.Printf("Error opening long position: %v", err)
			}
		}
	}
}

// switchMode switches the trading mode with validation
func (o *Orchestrator) switchMode(newMode TradingMode) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	currentMode := o.state.Mode

	// Validate transition
	allowedModes, exists := o.modeTransitions[currentMode]
	if !exists {
		return fmt.Errorf("no transitions defined for mode %s", currentMode)
	}

	allowed := false
	for _, mode := range allowedModes {
		if mode == newMode {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("transition from %s to %s is not allowed", currentMode, newMode)
	}

	// Perform mode transition
	oldMode := o.state.Mode
	o.state.Mode = newMode
	o.state.LastUpdateTime = time.Now()

	log.Printf("🔄 Mode transition: %s -> %s", oldMode, newMode)

	// Perform mode-specific setup
	switch newMode {
	case ModeGrid:
		return o.setupGridMode()
	case ModeBreakout:
		return o.setupBreakoutMode()
	case ModeRecovery:
		return o.setupRecoveryMode()
	case ModeStability:
		return o.setupStabilityMode()
	case ModeIdle:
		return o.setupIdleMode()
	}

	return nil
}

// waitForPriceAndInitializeGrid waits for sufficient historical data then initializes grid trading
func (o *Orchestrator) waitForPriceAndInitializeGrid() {
	defer o.wg.Done()

	// Wait for sufficient historical data
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	priceCheck := time.NewTicker(500 * time.Millisecond)
	defer priceCheck.Stop()

	timeout := time.After(5 * time.Minute) // 5 minute timeout for data collection
	havePrice := false

	for {
		select {
		case <-o.ctx.Done():
			return
		case <-timeout:
			log.Printf("❌ Timeout waiting for sufficient data after 5 minutes")
			return
		case <-priceCheck.C:
			// First check if we have any price data at all
			if !havePrice {
				currentPrice := o.candleAggregator.GetLatestPrice(o.activeSymbol)
				if currentPrice > 0 {
					havePrice = true
					log.Printf("📊 Received initial price data: %.2f, waiting for sufficient historical data...", currentPrice)
				}
			}
		case <-ticker.C:
			// Every 3 seconds, check if we have sufficient data for grid setup
			historicalCandles := o.candleAggregator.GetCandles(o.activeSymbol, data.Timeframe3s, 50)

			if len(historicalCandles) >= 50 {
				currentPrice := o.candleAggregator.GetLatestPrice(o.activeSymbol)
				log.Printf("📈 Sufficient data collected: %d candles, current price: %.2f", len(historicalCandles), currentPrice)

				// Check market conditions
				suitable, reason := o.gridSetup.ShouldSetupGrid(o.activeSymbol)
				if !suitable {
					log.Printf("⚠️ Market conditions not suitable: %s, continuing to wait...", reason)
					continue
				}

				// Initialize grid trading
				if err := o.initializeGridTrading(); err != nil {
					log.Printf("❌ Failed to initialize grid trading: %v", err)
					continue // Don't return, keep trying
				}

				// Switch to grid mode
				if err := o.switchMode(ModeGrid); err != nil {
					log.Printf("❌ Failed to switch to grid mode: %v", err)
					continue
				}

				log.Printf("✅ Grid trading started successfully with proper analysis!")
				return
			} else {
				if havePrice {
					log.Printf("⏳ Collecting data: %d/50 candles needed (%.1f%% complete)",
						len(historicalCandles), float64(len(historicalCandles))*100/50)
				}
			}
		}
	}
}

// setupGridMode sets up grid trading mode
func (o *Orchestrator) setupGridMode() error {
	// Reset breakout info
	o.state.BreakoutInfo = nil

	// Initialize or reinitialize grid
	return o.initializeGridTrading()
}

// setupBreakoutMode sets up breakout mode
func (o *Orchestrator) setupBreakoutMode() error {
	// Ensure breakout info exists
	if o.state.BreakoutInfo == nil {
		return fmt.Errorf("breakout info not available for breakout mode")
	}

	// Initialize position management for breakout
	return nil
}

// setupRecoveryMode sets up recovery mode
func (o *Orchestrator) setupRecoveryMode() error {
	// Recovery mode setup
	log.Println("🔄 Recovery mode activated")
	return nil
}

// setupStabilityMode sets up stability detection mode
func (o *Orchestrator) setupStabilityMode() error {
	// Start stability detection timing
	if o.state.BreakoutInfo != nil {
		now := time.Now()
		o.state.BreakoutInfo.StabilityWaitStart = &now
	}

	log.Println("⏳ Stability detection mode activated")
	return nil
}


// setupIdleMode sets up idle mode
func (o *Orchestrator) setupIdleMode() error {
	log.Println("😴 Idle mode activated")
	return nil
}

// initializeGridTrading initializes grid trading setup
func (o *Orchestrator) initializeGridTrading() error {
	// Get current price
	currentPrice := o.candleAggregator.GetLatestPrice(o.activeSymbol)
	if currentPrice == 0 {
		return fmt.Errorf("no current price available for grid setup")
	}

	// Require sufficient historical data for proper analysis
	historicalCandles := o.candleAggregator.GetCandles(o.activeSymbol, data.Timeframe3s, 50)
	if len(historicalCandles) < 50 {
		return fmt.Errorf("insufficient historical data for grid setup: need 50 candles, have %d", len(historicalCandles))
	}

	// Check if market conditions are suitable for grid trading
	suitable, reason := o.gridSetup.ShouldSetupGrid(o.activeSymbol)
	if !suitable {
		return fmt.Errorf("market conditions not suitable for grid trading: %s", reason)
	}

	// Perform comprehensive market analysis for grid setup
	gridParams, err := o.gridSetup.AnalyzeAndSetup(o.activeSymbol, o.config.InitialBalance)
	if err != nil {
		return fmt.Errorf("failed to analyze market for grid setup: %w", err)
	}

	// Calculate optimal grid parameters based on actual market analysis
	volatilityCategory := "medium"
	if gridParams.Volatility < 1.0 {
		volatilityCategory = "low"
	} else if gridParams.Volatility > 3.0 {
		volatilityCategory = "high"
	}

	gridCalcResult := o.gridCalculator.CalculateOptimalGrid(
		currentPrice,
		gridParams.Volatility,
		o.config.InitialBalance,
		volatilityCategory,
	)

	// Set grid bounds based on analysis
	o.state.GridBounds = strategy.GridBounds{
		UpperBound: gridCalcResult.UpperBound,
		LowerBound: gridCalcResult.LowerBound,
		Center:     currentPrice,
		Range:      gridCalcResult.TotalRange,
	}

	log.Printf("✅ Grid trading initialized: Center=%.2f, Upper=%.2f, Lower=%.2f, Range=%.2f%%, Levels=%d, Spacing=%.2f%%, Volatility=%.3f (%s)",
		currentPrice, gridCalcResult.UpperBound, gridCalcResult.LowerBound,
		gridCalcResult.TotalRange*100, gridCalcResult.GridLevels, gridCalcResult.GridSpacing*100,
		gridParams.Volatility, volatilityCategory)

	return nil
}

// riskManagementWorker handles risk management
func (o *Orchestrator) riskManagementWorker() {
	defer o.wg.Done()

	ticker := time.NewTicker(o.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			return

		case <-ticker.C:
			// Perform risk assessment
			riskAssessment := o.riskManager.AssessRisk()

			// Send risk alerts if needed
			if riskAssessment.PortfolioHealth != "healthy" {
				o.riskChan <- RiskAlert{
					Level:     riskAssessment.PortfolioHealth,
					Type:      "portfolio_health",
					Message:   fmt.Sprintf("Portfolio health: %s (risk level: %.2f)",
						riskAssessment.PortfolioHealth, riskAssessment.OverallRiskLevel),
					Timestamp: time.Now(),
				}
			}

			// Check for critical risk conditions
			if riskAssessment.MarginCallRisk {
				o.handleCriticalRisk("margin_call", riskAssessment)
			}

		case alert := <-o.riskChan:
			o.handleRiskAlert(alert)
		}
	}
}

// handleCriticalRisk handles critical risk conditions
func (o *Orchestrator) handleCriticalRisk(riskType string, assessment *strategy.RiskAssessment) {
	log.Printf("🚨 CRITICAL RISK DETECTED: %s", riskType)

	// Emergency actions
	switch riskType {
	case "margin_call":
		// Close all positions immediately
		if err := o.closeAllPositions(); err != nil {
			log.Printf("Error closing positions in emergency: %v", err)
		}
		// Switch to idle mode
		o.switchMode(ModeIdle)
	}
}

// handleRiskAlert handles risk alerts
func (o *Orchestrator) handleRiskAlert(alert RiskAlert) {
	log.Printf("⚠️ Risk Alert [%s]: %s", alert.Level, alert.Message)

	// Take action based on alert level
	if alert.Level == "critical" {
		o.handleCriticalRisk(alert.Type, nil)
	}
}

// closeAllPositions closes all open positions
func (o *Orchestrator) closeAllPositions() error {
	position, err := o.tradingExecutor.GetPosition(o.activeSymbol)
	if err != nil {
		return err
	}

	if position != nil && math.Abs(position.Size) > 0 {
		var err error
		if position.Size > 0 {
			_, err = o.tradingExecutor.CloseLong(o.activeSymbol, position.Size, 0)
		} else {
			_, err = o.tradingExecutor.CloseShort(o.activeSymbol, -position.Size, 0)
		}
		if err != nil {
			return err
		}
		log.Printf("📉 Emergency position close: %.4f @ %.2f", position.Size, position.EntryPrice)
	}

	return nil
}

// modeManagementWorker manages mode transitions and state
func (o *Orchestrator) modeManagementWorker() {
	defer o.wg.Done()

	for {
		select {
		case <-o.ctx.Done():
			return

		default:
			// Periodic mode health checks
			time.Sleep(5 * time.Second)
			o.checkModeHealth()
		}
	}
}

// checkModeHealth performs health checks on current mode
func (o *Orchestrator) checkModeHealth() {
	o.mu.RLock()
	currentMode := o.state.Mode
	lastUpdate := o.state.LastUpdateTime
	o.mu.RUnlock()

	// Check if mode has been inactive for too long
	if time.Since(lastUpdate) > 30*time.Second {
		log.Printf("⚠️ Mode %s inactive for %v", currentMode, time.Since(lastUpdate))

		// Auto-switch to grid if stuck in other modes
		if currentMode != ModeGrid && currentMode != ModeIdle {
			log.Printf("🔄 Auto-switching to grid mode due to inactivity")
			_ = o.switchMode(ModeGrid)
		}
	}
}

// performanceWorker tracks performance metrics
func (o *Orchestrator) performanceWorker() {
	defer o.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			return

		case <-ticker.C:
			o.updatePerformanceMetrics()
		}
	}
}

// updatePerformanceMetrics updates performance tracking
func (o *Orchestrator) updatePerformanceMetrics() {
	// Get current position and balance
	position, err := o.tradingExecutor.GetPosition(o.activeSymbol)
	if err != nil {
		return
	}

	balance, err := o.tradingExecutor.GetAvailableBalance()
	if err != nil {
		return
	}

	// Update performance metrics
	o.mu.Lock()

	if position != nil {
		o.performance.TotalPnL = position.UnrealizedPnL
	}

	// Calculate drawdown
	peak := o.config.InitialBalance
	if balance > peak {
		peak = balance
	}
	drawdown := (peak - balance) / peak
	o.performance.CurrentDrawdown = drawdown

	if drawdown > o.performance.MaxDrawdown {
		o.performance.MaxDrawdown = drawdown
	}

	o.state.TotalPnL = o.performance.TotalPnL
	o.state.CurrentDrawdown = o.performance.CurrentDrawdown
	o.state.MaxDrawdown = o.performance.MaxDrawdown

	o.mu.Unlock()
}

// controlWorker processes control commands
func (o *Orchestrator) controlWorker() {
	defer o.wg.Done()

	for {
		select {
		case <-o.ctx.Done():
			return

		case cmd := <-o.controlChan:
			o.processControlCommand(cmd)
		}
	}
}

// processControlCommand processes control commands
func (o *Orchestrator) processControlCommand(cmd ControlCommand) {
	switch cmd.Type {
	case "stop":
		_ = o.Stop()
	case "pause":
		o.switchMode(ModeIdle)
	case "resume":
		_ = o.switchMode(ModeGrid)
	case "switch_mode":
		if mode, ok := cmd.Payload.(TradingMode); ok {
			_ = o.switchMode(mode)
		}
	}
}

// GetState returns current bot state
func (o *Orchestrator) GetState() BotState {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.state
}

// GetPerformance returns performance metrics
func (o *Orchestrator) GetPerformance() PerformanceMetrics {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.performance
}

// SendControlCommand sends a control command to the orchestrator
func (o *Orchestrator) SendControlCommand(cmd ControlCommand) {
	select {
	case o.controlChan <- cmd:
	default:
		log.Printf("Control channel full, dropping command: %s", cmd.Type)
	}
}

// Helper function
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}