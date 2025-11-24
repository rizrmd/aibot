package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// Config represents the complete application configuration
type Config struct {
	// Application settings
	App      AppConfig      `json:"app"`
	Trading  TradingConfig  `json:"trading"`
	Strategy StrategyConfig `json:"strategy"`
	Risk     RiskConfig     `json:"risk"`
	Stream   StreamConfig   `json:"stream"`
	Database DatabaseConfig `json:"database"`
	Logging  LoggingConfig  `json:"logging"`
	Backtest BacktestConfig `json:"backtest"`
}

// AppConfig contains basic application configuration
type AppConfig struct {
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	Environment string        `json:"environment"` // "development", "production", "test"
	Timezone    string        `json:"timezone"`
	Debug       bool          `json:"debug"`
	Enabled     bool          `json:"enabled"`         // Backtest enabled
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
	MaxGoroutines   int       `json:"max_goroutines"`
}

// TradingConfig contains trading-related configuration
type TradingConfig struct {
	// Account settings
	InitialBalance    float64 `json:"initial_balance"`
	MaxDailyLoss      float64 `json:"max_daily_loss"`
	MaxConsecutiveLosses int    `json:"max_consecutive_losses"`

	// Position settings
	DefaultLeverage   float64 `json:"default_leverage"`
	MaxLeverage       float64 `json:"max_leverage"`
	MinPositionSize   float64 `json:"min_position_size"`
	MaxPositionSize   float64 `json:"max_position_size"`

	// Fee settings
	MakerFee          float64 `json:"maker_fee"`
	TakerFee          float64 `json:"taker_fee"`
	Slippage          float64 `json:"slippage"`

	// Execution settings
	ExecutionType     string `json:"execution_type"` // "simulation", "live"
	OrderTimeout      time.Duration `json:"order_timeout"`
	RetryAttempts     int    `json:"retry_attempts"`
	RetryDelay        time.Duration `json:"retry_delay"`

	// Market settings
	SupportedSymbols   []string `json:"supported_symbols"`
	DefaultSymbol      string   `json:"default_symbol"`
	MaxSymbols         int      `json:"max_symbols"`
}

// StrategyConfig contains strategy-specific configuration
type StrategyConfig struct {
	// Grid strategy
	Grid GridConfig `json:"grid"`

	// Breakout detection
	Breakout BreakoutConfig `json:"breakout"`

	// False breakout detection
	FalseBreakout FalseBreakoutConfig `json:"false_breakout"`

	// Price stability detection
	Stability StabilityConfig `json:"stability"`

	// Technical analysis
	Technical TechnicalConfig `json:"technical"`
}

// GridConfig contains grid trading configuration
type GridConfig struct {
	// Grid setup
	MinGridSpacing    float64 `json:"min_grid_spacing"`    // 0.25%
	MaxGridSpacing    float64 `json:"max_grid_spacing"`    // 1.0%
	MinGridLevels     int     `json:"min_grid_levels"`     // 10
	MaxGridLevels     int     `json:"max_grid_levels"`     // 30

	// Profitability
	MinProfitPerLevel float64 `json:"min_profit_per_level"` // 0.15%
	TargetROI         float64 `json:"target_roi"`          // 1.0%

	// Market analysis
	HistoryWindow     int     `json:"history_window"`      // 200 candles
	VolatilityLookback int    `json:"volatility_lookback"` // 100 periods
	MinDataPoints     int     `json:"min_data_points"`     // 50

	// Grid bounds
	PriceBuffer       float64 `json:"price_buffer"`        // 5% buffer around current price
	RangeExpansion    float64 `json:"range_expansion"`     // 2x ATR for range expansion
}

// BreakoutConfig contains breakout detection configuration
type BreakoutConfig struct {
	// Confirmation
	ConfirmationCandles int     `json:"confirmation_candles"`  // 3 candles
	MinBreakoutStrength float64 `json:"min_breakout_strength"` // 0.3%

	// Volume analysis
	VolumeMultiplier    float64 `json:"volume_multiplier"`     // 1.5x average volume
	MinVolumeCandles    int     `json:"min_volume_candles"`    // 10 candles for volume average

	// Momentum
	MomentumThreshold   float64 `json:"momentum_threshold"`    // 0.5%
	RSIOverbought       float64 `json:"rsi_overbought"`        // 70
	RSIOversold         float64 `json:"rsi_oversold"`          // 30

	// ATR
	ATRMultiple         float64 `json:"atr_multiple"`          // 1.5x ATR

	// Performance tracking
	MaxFalseBreakouts   int     `json:"max_false_breakouts"`   // Consecutive false breakout limit
	ConfidenceThreshold float64 `json:"confidence_threshold"`  // 0.6 minimum confidence
}

// FalseBreakoutConfig contains false breakout detection configuration
type FalseBreakoutConfig struct {
	// Detection thresholds
	PriceReversionThreshold float64 `json:"price_reversion_threshold"` // 0.5%
	StrongReversalThreshold float64 `json:"strong_reversal_threshold"` // 1.0%

	// Confirmation
	ConfirmationCandles    int     `json:"confirmation_candles"`       // 3 candles
	VolumeDeclineThreshold float64 `json:"volume_decline_threshold"`   // 50% drop

	// Timing
	MomentumReversalMs     int     `json:"momentum_reversal_ms"`      // 900ms

	// Technical indicators
	ATRMultiple            float64 `json:"atr_multiple"`               // 1.5x
	StdDevMultiplier       float64 `json:"std_dev_multiplier"`        // 2.0x

	// Pattern detection
	MaxFakeoutFrequency    float64 `json:"max_fakeout_frequency"`     // Per hour
}

// StabilityConfig contains price stability detection configuration
type StabilityConfig struct {
	// Analysis parameters
	AnalysisWindow      int     `json:"analysis_window"`       // 10 candles
	VolatilityThreshold  float64 `json:"volatility_threshold"`  // 0.5%
	MomentumThreshold    float64 `json:"momentum_threshold"`    // 0.2%
	PriceConformity      float64 `json:"price_conformity"`      // 80%
	RangeContraction     float64 `json:"range_contraction"`     // 70%
	MinStabilityPeriods  int     `json:"min_stability_periods"` // 3 consecutive checks

	// Timeframes
	PrimaryTimeframe     string  `json:"primary_timeframe"`     // "3s"
	SecondaryTimeframe   string  `json:"secondary_timeframe"`   // "15s"

	// Risk levels
	LowRiskVolatility    float64 `json:"low_risk_volatility"`    // < 0.3%
	MediumRiskVolatility float64 `json:"medium_risk_volatility"` // 0.3-0.7%
	HighRiskVolatility   float64 `json:"high_risk_volatility"`   // > 0.7%
}

// TechnicalConfig contains technical analysis configuration
type TechnicalConfig struct {
	// Indicators
	IndicatorSettings map[string]interface{} `json:"indicator_settings"`

	// Timeframes
	AnalysisTimeframes []string `json:"analysis_timeframes"`

	// Data requirements
	MinHistoryCandles  int `json:"min_history_candles"`
	MaxHistoryCandles  int `json:"max_history_candles"`

	// Update intervals
	IndicatorUpdateInterval time.Duration `json:"indicator_update_interval"`
}

// RiskConfig contains risk management configuration
type RiskConfig struct {
	// Portfolio risk
	MaxPortfolioRisk     float64 `json:"max_portfolio_risk"`     // 5%
	MaxDrawdown          float64 `json:"max_drawdown"`           // 10%
	MaxCorrelation       float64 `json:"max_correlation"`        // 0.7

	// Position risk
	MaxPositionRisk      float64 `json:"max_position_risk"`      // 2%
	MinRiskRewardRatio   float64 `json:"min_risk_reward_ratio"`  // 1.5

	// Concentration
	ConcentrationLimit   float64 `json:"concentration_limit"`    // 30% in one asset

	// Volatility adjustment
	VolatilityMultiplier float64 `json:"volatility_multiplier"`  // 1.5

	// Risk assessment intervals
	RiskAssessmentInterval time.Duration `json:"risk_assessment_interval"`

	// Emergency conditions
	MarginCallThreshold   float64 `json:"margin_call_threshold"`   // 90% margin usage
	EmergencyStopLoss     float64 `json:"emergency_stop_loss"`     // 15% portfolio loss
}

// StreamConfig contains streaming data configuration
type StreamConfig struct {
	// Connection
	ProviderType      string        `json:"provider_type"`       // "simulation", "binance", "bitmex"
	ConnectTimeout    time.Duration `json:"connect_timeout"`
	ReadTimeout       time.Duration `json:"read_timeout"`
	WriteTimeout      time.Duration `json:"write_timeout"`
	PingInterval      time.Duration `json:"ping_interval"`
	ReconnectDelay    time.Duration `json:"reconnect_delay"`
	MaxReconnects     int           `json:"max_reconnects"`

	// Data processing
	BufferSize        int           `json:"buffer_size"`
	BatchSize         int           `json:"batch_size"`
	BatchTimeout      time.Duration `json:"batch_timeout"`

	// Simulation (if using simulation provider)
	SimulationSpeed   float64 `json:"simulation_speed"`   // 1.0 = real-time
	RandomVolatility  float64 `json:"random_volatility"`  // 0.1 = 10% volatility
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	// Connection
	Driver   string `json:"driver"`   // "sqlite", "postgres", "mysql"
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`

	// Connection pool
	MaxOpenConns int           `json:"max_open_conns"`
	MaxIdleConns int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`

	// Settings
	SSLMode   string `json:"ssl_mode"`
	Timezone  string `json:"timezone"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	// Output
	Level      string `json:"level"`       // "debug", "info", "warn", "error"
	Format     string `json:"format"`      // "json", "text"
	Output     string `json:"output"`      // "stdout", "file", "both"
	Directory  string `json:"directory"`   // Log file directory

	// File rotation
	MaxSize    int    `json:"max_size"`    // Max MB per file
	MaxBackups int    `json:"max_backups"` // Max number of old files
	MaxAge     int    `json:"max_age"`     // Max days to retain
	Compress   bool   `json:"compress"`    // Compress old files

	// Performance
	FlushInterval time.Duration `json:"flush_interval"`
	BufferSize    int           `json:"buffer_size"`

	// Structured logging
	EnableStructured bool     `json:"enable_structured"`
	Fields          []string `json:"fields"` // Fields to include in structured logs
}

// BacktestConfig contains backtesting configuration
type BacktestConfig struct {
	// Data
	DataDirectory      string        `json:"data_directory"`
	StartTime         time.Time     `json:"start_time"`
	EndTime           time.Time     `json:"end_time"`
	InitialBalance    float64       `json:"initial_balance"`

	// Execution
	Commission        float64       `json:"commission"`
	Slippage          float64       `json:"slippage"`
	Latency           time.Duration `json:"latency"`

	// Output
	ResultsDirectory   string        `json:"results_directory"`
	DetailedReports    bool          `json:"detailed_reports"`
	GenerateCharts     bool          `json:"generate_charts"`
	ExportTrades       bool          `json:"export_trades"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:        "AI Trading Bot",
			Version:     "1.0.0",
			Environment: "development",
			Timezone:    "UTC",
			Debug:       true,
			Enabled:     false,
			ShutdownTimeout: 30 * time.Second,
			MaxGoroutines:   100,
		},
		Trading: TradingConfig{
			InitialBalance:      10000.0,
			MaxDailyLoss:        500.0,   // 5% of initial balance
			MaxConsecutiveLosses: 5,
			DefaultLeverage:     5.0,
			MaxLeverage:         10.0,
			MinPositionSize:     0.001,
			MaxPositionSize:     1.0,
			MakerFee:            0.0002, // 0.02%
			TakerFee:            0.0006, // 0.06%
			Slippage:            0.0005, // 0.05%
			ExecutionType:       "simulation",
			OrderTimeout:        30 * time.Second,
			RetryAttempts:       3,
			RetryDelay:          1 * time.Second,
			SupportedSymbols:    []string{"BTCUSDT"},
			DefaultSymbol:       "BTCUSDT",
			MaxSymbols:          1,
		},
		Strategy: StrategyConfig{
			Grid: GridConfig{
				MinGridSpacing:      0.0025, // 0.25%
				MaxGridSpacing:      0.01,   // 1.0%
				MinGridLevels:       10,
				MaxGridLevels:       30,
				MinProfitPerLevel:   0.0015, // 0.15%
				TargetROI:           0.01,   // 1.0%
				HistoryWindow:       200,
				VolatilityLookback:  100,
				MinDataPoints:       50,
				PriceBuffer:         0.05,   // 5%
				RangeExpansion:      2.0,    // 2x ATR
			},
			Breakout: BreakoutConfig{
				ConfirmationCandles:  3,
				MinBreakoutStrength:  0.003, // 0.3%
				VolumeMultiplier:     1.5,
				MinVolumeCandles:     10,
				MomentumThreshold:    0.005, // 0.5%
				RSIOverbought:        70,
				RSIOversold:          30,
				ATRMultiple:          1.5,
				MaxFalseBreakouts:    3,
				ConfidenceThreshold:  0.6,
			},
			FalseBreakout: FalseBreakoutConfig{
				PriceReversionThreshold: 0.005, // 0.5%
				StrongReversalThreshold: 0.01,  // 1.0%
				ConfirmationCandles:     3,
				VolumeDeclineThreshold:  0.5,   // 50%
				MomentumReversalMs:      900,
				ATRMultiple:             1.5,
				StdDevMultiplier:        2.0,
				MaxFakeoutFrequency:     5.0, // Per hour
			},
			Stability: StabilityConfig{
				AnalysisWindow:      10,
				VolatilityThreshold:  0.005, // 0.5%
				MomentumThreshold:    0.002, // 0.2%
				PriceConformity:      0.8,   // 80%
				RangeContraction:     0.7,   // 70%
				MinStabilityPeriods:  3,
				PrimaryTimeframe:     "3s",
				SecondaryTimeframe:   "15s",
				LowRiskVolatility:    0.003, // < 0.3%
				MediumRiskVolatility: 0.007, // 0.3-0.7%
				HighRiskVolatility:   0.007, // > 0.7%
			},
			Technical: TechnicalConfig{
				IndicatorSettings: map[string]interface{}{
					"rsi_period":     14,
					"macd_fast":      12,
					"macd_slow":      26,
					"macd_signal":    9,
					"atr_period":     14,
					"bb_period":      20,
					"bb_stddev":      2,
					"sma_period":     20,
					"ema_period":     20,
				},
				AnalysisTimeframes:    []string{"1s", "3s", "15s"},
				MinHistoryCandles:     50,
				MaxHistoryCandles:     200,
				IndicatorUpdateInterval: 1 * time.Second,
			},
		},
		Risk: RiskConfig{
			MaxPortfolioRisk:        0.05, // 5%
			MaxDrawdown:             0.10, // 10%
			MaxCorrelation:          0.7,
			MaxPositionRisk:         0.02, // 2%
			MinRiskRewardRatio:      1.5,
			ConcentrationLimit:      0.3,  // 30%
			VolatilityMultiplier:    1.5,
			RiskAssessmentInterval:  1 * time.Minute,
			MarginCallThreshold:     0.9, // 90% margin usage
			EmergencyStopLoss:       0.15, // 15% portfolio loss
		},
		Stream: StreamConfig{
			ProviderType:    "simulation",
			ConnectTimeout:  10 * time.Second,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    10 * time.Second,
			PingInterval:    20 * time.Second,
			ReconnectDelay:  5 * time.Second,
			MaxReconnects:   5,
			BufferSize:      1000,
			BatchSize:       100,
			BatchTimeout:    1 * time.Second,
			SimulationSpeed: 1.0,
			RandomVolatility: 0.1,
		},
		Database: DatabaseConfig{
			Driver:         "sqlite",
			Database:       "trading_bot.db",
			MaxOpenConns:   25,
			MaxIdleConns:   5,
			ConnMaxLifetime: 5 * time.Minute,
			SSLMode:        "disable",
			Timezone:       "UTC",
		},
		Logging: LoggingConfig{
			Level:            "info",
			Format:           "json",
			Output:           "both",
			Directory:        "./logs",
			MaxSize:          100, // MB
			MaxBackups:       10,
			MaxAge:           30, // days
			Compress:         true,
			FlushInterval:    5 * time.Second,
			BufferSize:       1000,
			EnableStructured: true,
			Fields:           []string{"timestamp", "level", "component", "message", "symbol", "price", "pnl"},
		},
		Backtest: BacktestConfig{
			InitialBalance:    10000.0,
			Commission:        0.0004, // 0.04% (average of maker/taker)
			Slippage:          0.0005, // 0.05%
			Latency:           50 * time.Millisecond,
			ResultsDirectory:  "./backtest_results",
			DetailedReports:   true,
			GenerateCharts:    true,
			ExportTrades:      true,
		},
	}
}

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config if file doesn't exist
		defaultConfig := DefaultConfig()
		if err := SaveConfig(defaultConfig, configPath); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return defaultConfig, nil
	}

	// Read file
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config, configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate app config
	if c.App.Name == "" {
		return fmt.Errorf("app name is required")
	}

	// Validate trading config
	if c.Trading.InitialBalance <= 0 {
		return fmt.Errorf("initial balance must be positive")
	}
	if c.Trading.MaxLeverage <= 0 {
		return fmt.Errorf("max leverage must be positive")
	}
	if c.Trading.DefaultLeverage > c.Trading.MaxLeverage {
		return fmt.Errorf("default leverage cannot exceed max leverage")
	}

	// Validate symbols
	if len(c.Trading.SupportedSymbols) == 0 {
		return fmt.Errorf("at least one supported symbol is required")
	}
	if c.Trading.DefaultSymbol == "" {
		return fmt.Errorf("default symbol is required")
	}

	// Validate strategy config
	if c.Strategy.Grid.MinGridLevels <= 0 {
		return fmt.Errorf("min grid levels must be positive")
	}
	if c.Strategy.Grid.MaxGridLevels <= c.Strategy.Grid.MinGridLevels {
		return fmt.Errorf("max grid levels must be greater than min grid levels")
	}

	// Validate risk config
	if c.Risk.MaxPortfolioRisk <= 0 || c.Risk.MaxPortfolioRisk > 1 {
		return fmt.Errorf("max portfolio risk must be between 0 and 1")
	}
	if c.Risk.MaxPositionRisk <= 0 || c.Risk.MaxPositionRisk > 1 {
		return fmt.Errorf("max position risk must be between 0 and 1")
	}

	// Validate logging config
	validLevels := []string{"debug", "info", "warn", "error"}
	levelValid := false
	for _, level := range validLevels {
		if c.Logging.Level == level {
			levelValid = true
			break
		}
	}
	if !levelValid {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	validFormats := []string{"json", "text"}
	formatValid := false
	for _, format := range validFormats {
		if c.Logging.Format == format {
			formatValid = true
			break
		}
	}
	if !formatValid {
		return fmt.Errorf("invalid log format: %s", c.Logging.Format)
	}

	return nil
}

// GetEnv returns environment variable with default value
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvBool returns boolean environment variable with default value
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

// GetEnvFloat returns float environment variable with default value
func GetEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := parseFloat(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// GetEnvInt returns integer environment variable with default value
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := parseInt(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// Helper functions for parsing
func parseFloat(s string) (float64, error) {
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	return result, err
}

func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}