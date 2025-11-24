package trading

import (
	"fmt"
)

// TradingExecutorFactory creates trading executors based on configuration
type TradingExecutorFactory struct{}

// NewTradingExecutorFactory creates a new factory
func NewTradingExecutorFactory() *TradingExecutorFactory {
	return &TradingExecutorFactory{}
}

// CreateTradingExecutor creates a trading executor based on the configuration
func (f *TradingExecutorFactory) CreateTradingExecutor(config interface{}) (TradingExecutor, error) {
	var providerType string

	// Type assert to get provider type
	switch c := config.(type) {
	case ExecutionConfig:
		providerType = c.ProviderType
	case SimulationConfig:
		providerType = c.ProviderType
	case LiveConfig:
		providerType = c.ProviderType
	case BacktestConfig:
		providerType = c.ProviderType
	default:
		return nil, fmt.Errorf("unknown configuration type")
	}

	switch providerType {
	case "simulation":
		simConfig, ok := config.(SimulationConfig)
		if !ok {
			return nil, fmt.Errorf("invalid configuration for simulation executor")
		}
		return NewSimulationExecutor(simConfig), nil

	case "live":
		liveConfig, ok := config.(LiveConfig)
		if !ok {
			return nil, fmt.Errorf("invalid configuration for live executor")
		}
		return f.createLiveExecutor(liveConfig)

	case "backtest":
		backtestConfig, ok := config.(BacktestConfig)
		if !ok {
			return nil, fmt.Errorf("invalid configuration for backtest executor")
		}
		return f.createBacktestExecutor(backtestConfig)

	default:
		return nil, fmt.Errorf("unsupported executor type: %s", providerType)
	}
}

// createLiveExecutor creates a live trading executor
func (f *TradingExecutorFactory) createLiveExecutor(config LiveConfig) (TradingExecutor, error) {
	// This would be implemented for real exchanges like Binance, Bybit, etc.
	// For now, we'll return a simulation executor as a placeholder
	return nil, fmt.Errorf("live executors not implemented yet - use simulation for now")
}

// createBacktestExecutor creates a backtesting executor
func (f *TradingExecutorFactory) createBacktestExecutor(config BacktestConfig) (TradingExecutor, error) {
	// This would be implemented for historical backtesting
	// For now, we'll return a simulation executor as a placeholder
	return nil, fmt.Errorf("backtest executors not implemented yet - use simulation for now")
}