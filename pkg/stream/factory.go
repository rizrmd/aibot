package stream

import (
	"fmt"
)

// StreamProviderFactory creates stream providers based on configuration
type StreamProviderFactory struct{}

// NewStreamProviderFactory creates a new factory
func NewStreamProviderFactory() *StreamProviderFactory {
	return &StreamProviderFactory{}
}

// CreateStreamProvider creates a stream provider based on the configuration
func (f *StreamProviderFactory) CreateStreamProvider(config interface{}) (StreamProvider, error) {
	var providerType string

	// Type assert to get provider type
	switch c := config.(type) {
	case StreamConfig:
		providerType = c.ProviderType
	case SimulationConfig:
		providerType = c.ProviderType
	case RealStreamConfig:
		providerType = c.ProviderType
	case ReplayConfig:
		providerType = c.ProviderType
	default:
		return nil, fmt.Errorf("unknown configuration type")
	}

	switch providerType {
	case "simulation":
		simConfig, ok := config.(SimulationConfig)
		if !ok {
			return nil, fmt.Errorf("invalid configuration for simulation provider")
		}
		return NewSimulationProvider(simConfig), nil

	case "real":
		realConfig, ok := config.(RealStreamConfig)
		if !ok {
			return nil, fmt.Errorf("invalid configuration for real provider")
		}
		return f.createRealProvider(realConfig)

	case "replay":
		replayConfig, ok := config.(ReplayConfig)
		if !ok {
			return nil, fmt.Errorf("invalid configuration for replay provider")
		}
		return f.createReplayProvider(replayConfig)

	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// createRealProvider creates a real WebSocket provider
func (f *StreamProviderFactory) createRealProvider(config RealStreamConfig) (StreamProvider, error) {
	// This would be implemented for real exchanges like Binance, Bybit, etc.
	// For now, we'll return a simulation provider as a placeholder
	return nil, fmt.Errorf("real providers not implemented yet - use simulation for now")
}

// createReplayProvider creates a historical replay provider
func (f *StreamProviderFactory) createReplayProvider(config ReplayConfig) (StreamProvider, error) {
	// This would be implemented for historical data replay
	// For now, we'll return a simulation provider as a placeholder
	return nil, fmt.Errorf("replay providers not implemented yet - use simulation for now")
}