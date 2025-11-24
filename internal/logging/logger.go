package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"aibot/internal/config"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger wraps logrus logger with additional functionality
type Logger struct {
	*logrus.Logger
	component string
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level       string
	Format      string
	Output      string
	Directory   string
	MaxSize     int
	MaxBackups  int
	MaxAge      int
	Compress    bool
	FlushInterval time.Duration
	BufferSize  int
	EnableStructured bool
	Fields      []string
}

// Field represents a log field
type Field struct {
	Key   string
	Value interface{}
}

// Log levels
const (
	DebugLevel = logrus.DebugLevel
	InfoLevel  = logrus.InfoLevel
	WarnLevel  = logrus.WarnLevel
	ErrorLevel = logrus.ErrorLevel
	FatalLevel = logrus.FatalLevel
	PanicLevel = logrus.PanicLevel
)

// Global logger instance
var globalLogger *Logger

// NewLogger creates a new logger with the given configuration
func NewLogger(cfg config.LoggingConfig) *Logger {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set formatter
	if cfg.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	// Set output
	var output io.Writer
	switch cfg.Output {
	case "stdout":
		output = os.Stdout
	case "file":
		output = createFileWriter(cfg)
	case "both":
		output = io.MultiWriter(os.Stdout, createFileWriter(cfg))
	default:
		output = os.Stdout
	}

	logger.SetOutput(output)

	return &Logger{
		Logger: logger,
	}
}

// createFileWriter creates a rotating file writer
func createFileWriter(cfg config.LoggingConfig) io.Writer {
	// Ensure log directory exists
	if err := os.MkdirAll(cfg.Directory, 0755); err != nil {
		fmt.Printf("Warning: Failed to create log directory: %v\n", err)
		return os.Stdout
	}

	logFile := filepath.Join(cfg.Directory, "trading_bot.log")

	return &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    cfg.MaxSize,    // MB
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,     // days
		Compress:   cfg.Compress,
	}
}

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(cfg config.LoggingConfig) {
	globalLogger = NewLogger(cfg)
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		// Create default logger if not initialized
		globalLogger = NewLogger(config.LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		})
	}
	return globalLogger
}

// NewComponentLogger creates a logger for a specific component
func NewComponentLogger(component string) *Logger {
	baseLogger := GetGlobalLogger()
	return &Logger{
		Logger:    baseLogger.Logger,
		component: component,
	}
}

// Logging methods with component awareness

// Debug logs a debug message
func (l *Logger) Debug(args ...interface{}) {
	if l.component != "" {
		l.WithField("component", l.component).Debug(args...)
	} else {
		l.Logger.Debug(args...)
	}
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.component != "" {
		l.WithField("component", l.component).Debugf(format, args...)
	} else {
		l.Logger.Debugf(format, args...)
	}
}

// Info logs an info message
func (l *Logger) Info(args ...interface{}) {
	if l.component != "" {
		l.WithField("component", l.component).Info(args...)
	} else {
		l.Logger.Info(args...)
	}
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	if l.component != "" {
		l.WithField("component", l.component).Infof(format, args...)
	} else {
		l.Logger.Infof(format, args...)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(args ...interface{}) {
	if l.component != "" {
		l.WithField("component", l.component).Warn(args...)
	} else {
		l.Logger.Warn(args...)
	}
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	if l.component != "" {
		l.WithField("component", l.component).Warnf(format, args...)
	} else {
		l.Logger.Warnf(format, args...)
	}
}

// Error logs an error message
func (l *Logger) Error(args ...interface{}) {
	if l.component != "" {
		l.WithField("component", l.component).Error(args...)
	} else {
		l.Logger.Error(args...)
	}
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	if l.component != "" {
		l.WithField("component", l.component).Errorf(format, args...)
	} else {
		l.Logger.Errorf(format, args...)
	}
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(args ...interface{}) {
	if l.component != "" {
		l.WithField("component", l.component).Fatal(args...)
	} else {
		l.Logger.Fatal(args...)
	}
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	if l.component != "" {
		l.WithField("component", l.component).Fatalf(format, args...)
	} else {
		l.Logger.Fatalf(format, args...)
	}
}

// Panic logs a panic message and panics
func (l *Logger) Panic(args ...interface{}) {
	if l.component != "" {
		l.WithField("component", l.component).Panic(args...)
	} else {
		l.Logger.Panic(args...)
	}
}

// Panicf logs a formatted panic message and panics
func (l *Logger) Panicf(format string, args ...interface{}) {
	if l.component != "" {
		l.WithField("component", l.component).Panicf(format, args...)
	} else {
		l.Logger.Panicf(format, args...)
	}
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	return &Logger{
		Logger:    l.Logger.WithFields(fields).Logger,
		component: l.component,
	}
}

// WithField adds a single field to the logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		Logger:    l.Logger.WithField(key, value).Logger,
		component: l.component,
	}
}

// WithError adds an error field to the logger
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger:    l.Logger.WithError(err).Logger,
		component: l.component,
	}
}

// WithCaller adds caller information to the logger
func (l *Logger) WithCaller() *Logger {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return l
	}

	return l.WithFields(logrus.Fields{
		"file": file,
		"line": line,
	})
}

// Trading-specific logging methods

// LogTrade logs a trade execution
func (l *Logger) LogTrade(symbol string, side string, quantity float64, price float64, commission float64) {
	l.WithFields(logrus.Fields{
		"event":     "trade",
		"symbol":    symbol,
		"side":      side,
		"quantity":  quantity,
		"price":     price,
		"commission": commission,
		"value":     quantity * price,
	}).Info("Trade executed")
}

// LogPosition logs a position update
func (l *Logger) LogPosition(symbol string, size float64, entryPrice float64, unrealizedPnL float64, realizedPnL float64) {
	l.WithFields(logrus.Fields{
		"event":          "position_update",
		"symbol":         symbol,
		"size":           size,
		"entry_price":    entryPrice,
		"unrealized_pnl": unrealizedPnL,
		"realized_pnl":   realizedPnL,
	}).Info("Position updated")
}

// LogSignal logs a trading signal
func (l *Logger) LogSignal(signalType string, symbol string, action string, confidence float64, reason string) {
	l.WithFields(logrus.Fields{
		"event":      "trading_signal",
		"signal_type": signalType,
		"symbol":     symbol,
		"action":     action,
		"confidence": confidence,
		"reason":     reason,
	}).Info("Trading signal generated")
}

// LogRisk logs a risk management event
func (l *Logger) LogRisk(riskType string, level string, message string, value float64, threshold float64) {
	l.WithFields(logrus.Fields{
		"event":     "risk_alert",
		"risk_type": riskType,
		"level":     level,
		"value":     value,
		"threshold": threshold,
	}).Warn(message)
}

// LogMode logs a mode transition
func (l *Logger) LogMode(fromMode string, toMode string, reason string) {
	l.WithFields(logrus.Fields{
		"event":    "mode_transition",
		"from_mode": fromMode,
		"to_mode":  toMode,
		"reason":   reason,
	}).Info("Mode transition")
}

// LogBreakout logs a breakout event
func (l *Logger) LogBreakout(symbol string, breakoutType string, price float64, confidence float64, strength float64) {
	l.WithFields(logrus.Fields{
		"event":         "breakout",
		"symbol":        symbol,
		"breakout_type": breakoutType,
		"price":         price,
		"confidence":    confidence,
		"strength":      strength,
	}).Info("Breakout detected")
}

// LogFalseBreakout logs a false breakout event
func (l *Logger) LogFalseBreakout(symbol string, falseBreakoutType string, price float64, confidence float64, recoveryAction string) {
	l.WithFields(logrus.Fields{
		"event":             "false_breakout",
		"symbol":            symbol,
		"false_breakout_type": falseBreakoutType,
		"price":             price,
		"confidence":        confidence,
		"recovery_action":   recoveryAction,
	}).Warn("False breakout detected")
}

// LogStability logs a stability detection event
func (l *Logger) LogStability(symbol string, isStable bool, confidence float64, reason string) {
	level := logrus.InfoLevel
	if !isStable {
		level = logrus.WarnLevel
	}

	l.WithFields(logrus.Fields{
		"event":      "stability_check",
		"symbol":     symbol,
		"is_stable":  isStable,
		"confidence": confidence,
		"reason":     reason,
	}).Log(level, "Price stability analysis")
}

// LogPerformance logs performance metrics
func (l *Logger) LogPerformance(totalPnL float64, winRate float64, tradeCount int, sharpeRatio float64) {
	l.WithFields(logrus.Fields{
		"event":       "performance_update",
		"total_pnl":   totalPnL,
		"win_rate":    winRate,
		"trade_count": tradeCount,
		"sharpe_ratio": sharpeRatio,
	}).Info("Performance metrics updated")
}

// LogGrid logs grid-related events
func (l *Logger) LogGrid(symbol string, event string, gridLevels int, spacing float64, bounds map[string]float64) {
	l.WithFields(logrus.Fields{
		"event":       "grid_event",
		"symbol":      symbol,
		"grid_action": event,
		"grid_levels": gridLevels,
		"spacing":     spacing,
		"upper_bound": bounds["upper"],
		"lower_bound": bounds["lower"],
		"center":      bounds["center"],
	}).Info("Grid operation")
}

// LogError logs an error with context
func (l *Logger) LogError(operation string, err error, context map[string]interface{}) {
	fields := logrus.Fields{
		"event":     "error",
		"operation": operation,
		"error":     err.Error(),
	}

	// Add context fields
	for k, v := range context {
		fields[k] = v
	}

	l.WithFields(fields).Error("Operation failed")
}

// LogSystem logs system-level events
func (l *Logger) LogSystem(event string, message string, details map[string]interface{}) {
	fields := logrus.Fields{
		"event": "system_event",
		"system_event": event,
	}

	// Add detail fields
	for k, v := range details {
		fields[k] = v
	}

	l.WithFields(fields).Info(message)
}

// Global convenience functions

// Debug logs a debug message using the global logger
func Debug(args ...interface{}) {
	GetGlobalLogger().Debug(args...)
}

// Debugf logs a formatted debug message using the global logger
func Debugf(format string, args ...interface{}) {
	GetGlobalLogger().Debugf(format, args...)
}

// Info logs an info message using the global logger
func Info(args ...interface{}) {
	GetGlobalLogger().Info(args...)
}

// Infof logs a formatted info message using the global logger
func Infof(format string, args ...interface{}) {
	GetGlobalLogger().Infof(format, args...)
}

// Warn logs a warning message using the global logger
func Warn(args ...interface{}) {
	GetGlobalLogger().Warn(args...)
}

// Warnf logs a formatted warning message using the global logger
func Warnf(format string, args ...interface{}) {
	GetGlobalLogger().Warnf(format, args...)
}

// Error logs an error message using the global logger
func Error(args ...interface{}) {
	GetGlobalLogger().Error(args...)
}

// Errorf logs a formatted error message using the global logger
func Errorf(format string, args ...interface{}) {
	GetGlobalLogger().Errorf(format, args...)
}

// Fatal logs a fatal message using the global logger
func Fatal(args ...interface{}) {
	GetGlobalLogger().Fatal(args...)
}

// Fatalf logs a formatted fatal message using the global logger
func Fatalf(format string, args ...interface{}) {
	GetGlobalLogger().Fatalf(format, args...)
}

// WithFields adds fields to the global logger
func WithFields(fields map[string]interface{}) *Logger {
	return GetGlobalLogger().WithFields(fields)
}

// WithField adds a field to the global logger
func WithField(key string, value interface{}) *Logger {
	return GetGlobalLogger().WithField(key, value)
}

// WithError adds an error field to the global logger
func WithError(err error) *Logger {
	return GetGlobalLogger().WithError(err)
}

// CreatePerformanceLogger creates a logger specifically for performance tracking
func CreatePerformanceLogger() *Logger {
	return NewComponentLogger("performance")
}

// CreateTradingLogger creates a logger specifically for trading operations
func CreateTradingLogger() *Logger {
	return NewComponentLogger("trading")
}

// CreateRiskLogger creates a logger specifically for risk management
func CreateRiskLogger() *Logger {
	return NewComponentLogger("risk")
}

// CreateStrategyLogger creates a logger specifically for strategy operations
func CreateStrategyLogger() *Logger {
	return NewComponentLogger("strategy")
}

// CreateDataLogger creates a logger specifically for data operations
func CreateDataLogger() *Logger {
	return NewComponentLogger("data")
}