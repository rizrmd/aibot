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
	case RealStreamConfig:
		providerType = c.ProviderType
	case ReplayConfig:
		providerType = c.ProviderType
	default:
		return nil, fmt.Errorf("unknown configuration type")
	}

	switch providerType {
	case "live":
		realConfig, ok := config.(RealStreamConfig)
		if !ok {
			return nil, fmt.Errorf("invalid configuration for live provider")
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

// createRealProvider creates a live WebSocket provider
func (f *StreamProviderFactory) createRealProvider(config RealStreamConfig) (StreamProvider, error) {
	// This would be implemented for real exchanges like Binance, Bybit, etc.
	return nil, fmt.Errorf("live providers not implemented yet - use replay for now")
}

// createReplayProvider creates a historical replay provider
func (f *StreamProviderFactory) createReplayProvider(config ReplayConfig) (StreamProvider, error) {
	// This would be implemented for historical data replay
	return nil, fmt.Errorf("replay providers not implemented yet")
}