package types

import (
	"time"
)

// OrderResult represents the result of an order execution
type OrderResult struct {
	OrderID        string    `json:"order_id"`
	Symbol         string    `json:"symbol"`
	Side           string    `json:"side"`
	PositionType   string    `json:"position_type"`
	Quantity       float64   `json:"quantity"`
	Price          float64   `json:"price"`
	FilledQty      float64   `json:"filled_qty"`
	FilledPrice    float64   `json:"filled_price"`
	Fee            float64   `json:"fee"`
	Timestamp      time.Time `json:"timestamp"`
	Status         string    `json:"status"` // "filled", "partial", "rejected", "pending"
	ExecutedTime   time.Time `json:"executed_time"`
	Commission     float64   `json:"commission"`
	CommissionAsset string   `json:"commission_asset"`
}