package stream

import (
	"aibot/internal/types"
	"context"
	"time"
)

// StreamProvider defines the interface for data streaming providers
type StreamProvider interface {
	// Start begins streaming data for the given symbols
	Start(ctx context.Context, symbols []string) error

	// Stop stops the streaming provider
	Stop() error

	// Subscribe adds symbols to the subscription list
	Subscribe(symbols []string) error

	// Unsubscribe removes symbols from the subscription list
	Unsubscribe(symbols []string) error

	// GetOHLCVChannel returns the channel for OHLCV data
	GetOHLCVChannel() <-chan types.OHLCV

	// GetTickerChannel returns the channel for ticker data
	GetTickerChannel() <-chan types.Ticker

	// IsConnected returns true if the provider is connected
	IsConnected() bool

	// GetSubscribedSymbols returns the list of currently subscribed symbols
	GetSubscribedSymbols() []string

	// GetLastError returns the last error that occurred
	GetLastError() error
}

// StreamConfig holds configuration for stream providers
type StreamConfig struct {
	ProviderType    string        `json:"provider_type"`    // "live", "replay"
	Exchange        string        `json:"exchange"`         // "binance", "bybit", etc.
	APIKey          string        `json:"api_key"`
	APISecret       string        `json:"api_secret"`
	Testnet         bool          `json:"testnet"`
	Symbols         []string      `json:"symbols"`
	ReconnectDelay  time.Duration `json:"reconnect_delay"`
	MaxRetries      int           `json:"max_retries"`
	BufferSize      int           `json:"buffer_size"`
}


// RealStreamConfig holds specific configuration for real WebSocket providers
type RealStreamConfig struct {
	StreamConfig
	WSSURL          string        `json:"ws_url"`
	PingInterval     time.Duration `json:"ping_interval"`
	Timeout         time.Duration `json:"timeout"`
	Compression     bool          `json:"compression"`
	RateLimitPerSec int           `json:"rate_limit_per_sec"`
}

// ReplayConfig holds specific configuration for historical replay
type ReplayConfig struct {
	StreamConfig
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
	DataFiles       []string      `json:"data_files"`
	PlaybackSpeed   float64       `json:"playback_speed"`   // 1.0 = normal speed
	PausePoints     []time.Time   `json:"pause_points"`     // Specific times to pause
}

// StreamEvent represents various events from the stream
type StreamEvent struct {
	Type      string      `json:"type"`      // "connect", "disconnect", "error", "subscribe", "unsubscribe"
	Symbol    string      `json:"symbol"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
	Error     error       `json:"error,omitempty"`
}

// StreamStats provides statistics about the stream
type StreamStats struct {
	MessagesReceived    int64     `json:"messages_received"`
	MessagesPerSecond   float64   `json:"messages_per_second"`
	LastMessageTime     time.Time `json:"last_message_time"`
	ConnectionUptime    time.Duration `json:"connection_uptime"`
	ReconnectCount      int       `json:"reconnect_count"`
	ErrorCount          int       `json:"error_count"`
	SymbolsSubscribed   int       `json:"symbols_subscribed"`
	BytesReceived       int64     `json:"bytes_received"`
	LatencyMs           float64   `json:"latency_ms"`      // Average latency in milliseconds
}