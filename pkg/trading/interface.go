package trading

import (
	"aibot/internal/types"
	"context"
	"time"
)

// TradingExecutor defines the interface for trading execution providers
type TradingExecutor interface {
	// Position management
	OpenLong(symbol string, quantity float64, price float64) (*types.OrderResult, error)
	OpenShort(symbol string, quantity float64, price float64) (*types.OrderResult, error)
	CloseLong(symbol string, quantity float64, price float64) (*types.OrderResult, error)
	CloseShort(symbol string, quantity float64, price float64) (*types.OrderResult, error)

	// Order management
	PlaceOrder(order *types.Order) (*types.OrderResult, error)
	CancelOrder(orderID string) error
	GetOrder(orderID string) (*types.Order, error)
	GetOpenOrders(symbol string) ([]*types.Order, error)
	GetOrderHistory(symbol string, limit int) ([]*types.Order, error)

	// Position tracking
	GetPosition(symbol string) (*types.Position, error)
	GetAllPositions() ([]*types.Position, error)

	// Account information
	GetBalance() (float64, error)
	GetAvailableBalance() (float64, error)
	GetMarginInfo() (*MarginInfo, error)

	// Market data
	GetTicker(symbol string) (*types.Ticker, error)
	GetOrderBook(symbol string, depth int) (*OrderBook, error)

	// Connection
	IsConnected() bool
	Connect(ctx context.Context) error
	Disconnect() error

	// Configuration
	GetFeeRates() (*FeeRates, error)
	GetLeverage(symbol string) (float64, error)
	SetLeverage(symbol string, leverage float64) error
}

// ExecutionConfig holds configuration for execution providers
type ExecutionConfig struct {
	ProviderType     string  `json:"provider_type"`     // "live"
	Exchange         string  `json:"exchange"`          // "binance", "bybit", etc.
	APIKey          string  `json:"api_key"`
	APISecret       string  `json:"api_secret"`
	Testnet         bool    `json:"testnet"`
	InitialBalance  float64 `json:"initial_balance"`
	MaxLeverage     float64 `json:"max_leverage"`
	DefaultLeverage float64 `json:"default_leverage"`
	RiskPerTrade    float64 `json:"risk_per_trade"`    // Percentage per trade
	MaxOpenPositions int     `json:"max_open_positions"`
	Commission      float64 `json:"commission"`         // Default commission rate
	Slippage        float64 `json:"slippage"`          // Default slippage percentage
}


// LiveConfig holds specific configuration for live trading
type LiveConfig struct {
	ExecutionConfig
	WSSURL          string        `json:"ws_url"`
	RESTURL         string        `json:"rest_url"`
	Timeout         time.Duration `json:"timeout"`
	RateLimitPerSec int           `json:"rate_limit_per_sec"`
	UseTestNet      bool          `json:"use_testnet"`
	EnableHedging   bool          `json:"enable_hedging"`
}


// MarginInfo contains margin information
type MarginInfo struct {
	TotalBalance       float64 `json:"total_balance"`
	AvailableBalance   float64 `json:"available_balance"`
	UsedMargin         float64 `json:"used_margin"`
	FreeMargin         float64 `json:"free_margin"`
	MarginLevel        float64 `json:"margin_level"`
	MaintenanceMargin  float64 `json:"maintenance_margin"`
	Leverage           float64 `json:"leverage"`
	Currency           string  `json:"currency"`
}

// OrderBook represents the order book
type OrderBook struct {
	Symbol    string          `json:"symbol"`
	Bids      []PriceLevel    `json:"bids"`
	Asks      []PriceLevel    `json:"asks"`
	Timestamp time.Time       `json:"timestamp"`
}

// PriceLevel represents a price level in the order book
type PriceLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Orders   int     `json:"orders"`
}

// FeeRates represents fee rates for different order types
type FeeRates struct {
	MakerFee  float64 `json:"maker_fee"`   // Fee for limit orders
	TakerFee  float64 `json:"taker_fee"`   // Fee for market orders
	SettlementFee float64 `json:"settlement_fee"` // Fee for futures settlement
}

// ExecutionStats provides statistics about trading execution
type ExecutionStats struct {
	TotalOrders       int64     `json:"total_orders"`
	SuccessfulOrders  int64     `json:"successful_orders"`
	FailedOrders      int64     `json:"failed_orders"`
	TotalVolume       float64   `json:"total_volume"`
	TotalFees         float64   `json:"total_fees"`
	AvgLatency        time.Duration `json:"avg_latency"`
	LastOrderTime     time.Time `json:"last_order_time"`
	OpenPositions     int       `json:"open_positions"`
	RealizedPnL       float64   `json:"realized_pnl"`
	UnrealizedPnL     float64   `json:"unrealized_pnl"`
	WinRate           float64   `json:"win_rate"`
	MaxDrawdown       float64   `json:"max_drawdown"`
	SharpeRatio       float64   `json:"sharpe_ratio"`
}

// OrderExecutionResult contains detailed information about order execution
type OrderExecutionResult struct {
	Order          *types.Order     `json:"order"`
	ExecutionTime  time.Duration    `json:"execution_time"`
	FilledQty      float64          `json:"filled_qty"`
	AvgFillPrice   float64          `json:"avg_fill_price"`
	Fee            float64          `json:"fee"`
	Slippage       float64          `json:"slippage"`
	RejectReason   string           `json:"reject_reason,omitempty"`
	PartialFills   []types.Order    `json:"partial_fills"`
}

// Trade represents a completed trade
type Trade struct {
	ID          string        `json:"id"`
	Symbol      string        `json:"symbol"`
	Side        types.OrderSide `json:"side"`
	Quantity    float64       `json:"quantity"`
	Price       float64       `json:"price"`
	Fee         float64       `json:"fee"`
	Timestamp   time.Time     `json:"timestamp"`
	OrderID     string        `json:"order_id"`
	TradeID     string        `json:"trade_id"`
	IsMaker     bool          `json:"is_maker"`
}