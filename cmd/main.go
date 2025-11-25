package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"aibot/internal/bot"
	"aibot/internal/config"
	"aibot/internal/logging"
	"aibot/internal/strategy"
	"aibot/pkg/stream"
	"aibot/pkg/trading"

	"github.com/sirupsen/logrus"
)

const (
	// Application constants
	AppName    = "AI Trading Bot"
	AppVersion = "1.0.0"
	DefaultConfigPath = "./config.json"
)

var (
	// Command line flags
	configPath = flag.String("config", DefaultConfigPath, "Path to configuration file")
	debugMode  = flag.Bool("debug", false, "Enable debug mode")
	version    = flag.Bool("version", false, "Show version information")
	help       = flag.Bool("help", false, "Show help information")

	// Global variables
	cfg        *config.Config
	logger     *logging.Logger
	orchestrator *bot.Orchestrator
	streamProvider stream.StreamProvider
	tradingExecutor trading.TradingExecutor
)

// Application represents the main application
type Application struct {
	ctx        context.Context
	cancel     context.CancelFunc
	shutdownCh chan struct{}
}

func init() {
	// Set up command line parsing
	flag.Usage = printUsage

	// Set runtime optimizations
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	// Parse command line flags
	flag.Parse()

	// Handle version flag
	if *version {
		printVersion()
		os.Exit(0)
	}

	// Handle help flag
	if *help {
		printUsage()
		os.Exit(0)
	}

	// Initialize application
	app, err := initializeApplication()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	// Run application
	if err := app.run(); err != nil {
		logger.Fatalf("Application failed: %v", err)
	}

	logger.Info("Application shutdown completed")
}

// initializeApplication initializes the application
func initializeApplication() (*Application, error) {
	// Create application context
	ctx, cancel := context.WithCancel(context.Background())
	app := &Application{
		ctx:        ctx,
		cancel:     cancel,
		shutdownCh: make(chan struct{}),
	}

	// Load configuration
	var err error
	cfg, err = config.LoadConfig(*configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override debug mode if specified
	if *debugMode {
		cfg.App.Debug = true
		cfg.Logging.Level = "debug"
	}

	// Ensure required directories exist
	if err := ensureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	// Initialize logging
	logger = logging.NewLogger(cfg.Logging)
	logging.InitGlobalLogger(cfg.Logging)

	// Log application startup
	logger.WithFields(logrus.Fields{
		"version":     AppVersion,
		"environment": cfg.App.Environment,
		"config_path": *configPath,
		"debug_mode":  cfg.App.Debug,
	}).Info("Starting AI Trading Bot")

	// Validate configuration
	if err := validateConfiguration(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Initialize components
	if err := app.initializeComponents(cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	// Set up signal handling
	app.setupSignalHandling()

	return app, nil
}

// run runs the main application loop
func (app *Application) run() error {
	logger.Info("Application initialized successfully")

	// Run trading mode (now handles all modes through streaming config)
	logger.Info("Starting trading bot")
	return app.runLiveTrading()
}

// runLiveTrading runs the live trading mode
func (app *Application) runLiveTrading() error {
	// Create bot configuration for orchestrator
	botConfig := convertToBotConfig(cfg)

	// Create orchestrator
	var err error
	orchestrator, err = bot.NewOrchestrator(botConfig)
	if err != nil {
		return fmt.Errorf("failed to create orchestrator: %w", err)
	}

	// Start orchestrator
	if err := orchestrator.Start(streamProvider, tradingExecutor); err != nil {
		return fmt.Errorf("failed to start orchestrator: %w", err)
	}

	logger.Info("Trading bot started successfully")

	// Wait for shutdown signal
	select {
	case <-app.shutdownCh:
		logger.Info("Shutdown signal received")
	case <-app.ctx.Done():
		logger.Info("Context cancelled")
	}

	// Graceful shutdown
	return app.shutdown()
}


// initializeComponents initializes all application components
func (app *Application) initializeComponents(cfg *config.Config) error {
	logger.Info("Initializing application components")

	// Initialize stream provider
	var err error
	streamProvider, err = createStreamProvider(cfg.Stream)
	if err != nil {
		return fmt.Errorf("failed to create stream provider: %w", err)
	}

	// Initialize trading executor
	tradingExecutor, err = createTradingExecutor(cfg.Trading)
	if err != nil {
		return fmt.Errorf("failed to create trading executor: %w", err)
	}

	logger.Info("Components initialized successfully")
	return nil
}

// createStreamProvider creates the appropriate stream provider
func createStreamProvider(cfg config.StreamConfig) (stream.StreamProvider, error) {
	factory := stream.NewStreamProviderFactory()

	// Create live config
	liveConfig := stream.RealStreamConfig{
		StreamConfig: stream.StreamConfig{
			ProviderType: "live",
			Symbols:      []string{"BTCUSDT"},
		},
		WSSURL:          "wss://api.binance.com/ws/btcusdt@ticker", // Default example
		PingInterval:     20 * time.Second,
		Timeout:          30 * time.Second,
		Compression:      true,
		RateLimitPerSec:  10,
	}

	return factory.CreateStreamProvider(liveConfig)
}

// createTradingExecutor creates the appropriate trading executor
func createTradingExecutor(cfg config.TradingConfig) (trading.TradingExecutor, error) {
	factory := trading.NewTradingExecutorFactory()

	// Create live config
	liveConfig := trading.LiveConfig{
		ExecutionConfig: trading.ExecutionConfig{
			ProviderType:    "live",
			InitialBalance:  cfg.InitialBalance,
			DefaultLeverage: cfg.DefaultLeverage,
			Commission:      cfg.MakerFee + cfg.TakerFee,
		},
		WSSURL:          "wss://api.binance.com/ws/btcusdt@trade",
		RESTURL:         "https://api.binance.com/api/v3",
		Timeout:         30 * time.Second,
		RateLimitPerSec: 10,
		UseTestNet:      true,
		EnableHedging:   false,
	}

	return factory.CreateTradingExecutor(liveConfig)
}

// convertToBotConfig converts app config to bot orchestrator config
func convertToBotConfig(cfg *config.Config) *bot.BotConfig {
	return &bot.BotConfig{
		InitialBalance:    cfg.Trading.InitialBalance,
		MaxSymbols:        1, // Simplified
		DefaultSymbol:     cfg.Trading.DefaultSymbol,
		GridSetupConfig: strategy.GridSetupConfig{
			MinHistoryCandles: 100,
			AnalysisTimeframe: "3s",
			DefaultGridLevels: 20,
			MinGridLevels:     10,
			MaxGridLevels:     30,
			MinPriceRange:     0.03, // 3%
			MaxPriceRange:     0.10, // 10%
			RangeMultiplier:   1.2,
			ATRMultiplier:     2.0,
			MakerFee:          0.0002, // 0.02%
			TakerFee:          0.0006, // 0.06%
			MinProfitPerLevel: 0.0015, // 0.15%
		},
		BreakoutConfig: strategy.BreakoutConfig{
			ConfirmationPeriod: 3,
			MinBreakoutStrength: 0.5,
			VolumeMultiplier:    1.2,
			RSIOverbought:       70,
			RSIOversold:         30,
			ATRMultiple:         1.5,
			MomentumThreshold:   0.3,
		},
		FalseBreakoutConfig: strategy.FalseBreakoutConfig{
			PriceReversionThreshold: 0.005, // 0.5%
			StrongReversalThreshold: 0.01,  // 1.0%
			ConfirmationCandles:     3,
			VolumeDeclineThreshold:  0.5,   // 50%
			MomentumReversalMs:      900,
			StdDevMultiplier:        2.0,
		},
		StabilityConfig: strategy.StabilityConfig{
			AnalysisWindow:       10,
			VolatilityThreshold:  0.005, // 0.5%
			MomentumThreshold:    0.002, // 0.2%
			PriceConformity:      0.8,   // 80%
			RangeContraction:     0.7,   // 70%
			MinStabilityPeriods:  3,
			PrimaryTimeframe:     "3s",
			SecondaryTimeframe:   "15s",
		},
		RiskManagerConfig: strategy.RiskManagerConfig{
			MaxPortfolioRisk:     0.05,  // 5%
			MaxPositionRisk:      0.02,  // 2%
			MaxCorrelation:       0.7,
			MaxDrawdown:          0.10,  // 10%
			MinRiskRewardRatio:   1.5,
			DefaultLeverage:      5.0,
			MaxLeverage:          10.0,
			ConcentrationLimit:   0.3,   // 30%
			VolatilityMultiplier: 1.5,
		},
		StreamConfig: stream.StreamConfig{
			ProviderType: "live",
			Symbols:      []string{cfg.Trading.DefaultSymbol},
		},
		TradingConfig: trading.ExecutionConfig{
			ProviderType:    "live",
			InitialBalance:  cfg.Trading.InitialBalance,
			DefaultLeverage: cfg.Trading.DefaultLeverage,
			Commission:      cfg.Trading.MakerFee + cfg.Trading.TakerFee,
		},
		UpdateInterval:      1 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		MaxDailyLoss:        cfg.Trading.MaxDailyLoss,
		MaxConsecutiveLosses: cfg.Trading.MaxConsecutiveLosses,
	}
}

// setupSignalHandling sets up signal handling for graceful shutdown
func (app *Application) setupSignalHandling() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-sigCh
		logger.WithField("signal", sig.String()).Info("Signal received, initiating shutdown")

		// Cancel context to trigger shutdown
		app.cancel()
	}()
}

// shutdown performs graceful shutdown
func (app *Application) shutdown() error {
	logger.Info("Starting graceful shutdown")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownErrors := make(chan error, 1)

	// Perform shutdown in goroutine
	go func() {
		defer close(shutdownErrors)

		// Stop orchestrator
		if orchestrator != nil {
			logger.Info("Stopping orchestrator")
			if err := orchestrator.Stop(); err != nil {
				shutdownErrors <- fmt.Errorf("failed to stop orchestrator: %w", err)
				return
			}
		}

		// Stop stream provider
		if streamProvider != nil {
			logger.Info("Stopping stream provider")
			if err := streamProvider.Stop(); err != nil {
				shutdownErrors <- fmt.Errorf("failed to stop stream provider: %w", err)
				return
			}
		}

		// Disconnect trading executor
		if tradingExecutor != nil {
			logger.Info("Disconnecting trading executor")
			if err := tradingExecutor.Disconnect(); err != nil {
				shutdownErrors <- fmt.Errorf("failed to disconnect trading executor: %w", err)
				return
			}
		}

		logger.Info("Shutdown completed successfully")
	}()

	// Wait for shutdown or timeout
	select {
	case err := <-shutdownErrors:
		return err
	case <-shutdownCtx.Done():
		logger.Warn("Shutdown timeout reached")
		return fmt.Errorf("shutdown timeout")
	}
}

// ensureDirectories ensures required directories exist
func ensureDirectories() error {
	directories := []string{
		"./logs",
		"./data",
		"./config",
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// validateConfiguration performs additional configuration validation
func validateConfiguration(cfg *config.Config) error {
	// Validate that default symbol is in supported symbols
	found := false
	for _, symbol := range cfg.Trading.SupportedSymbols {
		if symbol == cfg.Trading.DefaultSymbol {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("default symbol %s is not in supported symbols", cfg.Trading.DefaultSymbol)
	}

	// Validate that config directory is accessible
	configDir := filepath.Dir(*configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return fmt.Errorf("config directory does not exist: %s", configDir)
	}

	// Validate log directory accessibility
	if cfg.Logging.Output == "file" || cfg.Logging.Output == "both" {
		if err := os.MkdirAll(cfg.Logging.Directory, 0755); err != nil {
			return fmt.Errorf("cannot create log directory: %w", err)
		}
	}

	return nil
}

// printUsage prints command line usage information
func printUsage() {
	fmt.Printf(`%s - %s

Usage: %s [options]

Options:
`, AppName, AppVersion, os.Args[0])
	flag.PrintDefaults()
	fmt.Printf(`
Examples:
  %s                                    # Run with default config
  %s -config ./myconfig.json            # Run with custom config
  %s -debug                            # Run in debug mode
  %s -version                          # Show version
  %s -help                             # Show this help

Environment Variables:
  TRADING_BOT_CONFIG_PATH    Path to configuration file (overrides -config flag)
  TRADING_BOT_DEBUG          Enable debug mode (overrides -debug flag)
  TRADING_BOT_LOG_LEVEL      Override log level (debug, info, warn, error)
  TRADING_BOT_ENVIRONMENT   Override environment setting

Configuration:
  A configuration file will be created with default values if it doesn't exist.
  The default configuration file location is: %s

For more information, see the documentation.
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], DefaultConfigPath)
}

// printVersion prints version information
func printVersion() {
	fmt.Printf(`%s %s

Go Version: %s
GOOS: %s
GOARCH: %s
Built: %s
`, AppName, AppVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH, time.Now().Format(time.RFC3339))
}