package types

import (
	"time"
)

// OrderSide represents the side of an order
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderType represents the type of order
type OrderType string

const (
	OrderTypeMarket OrderType = "market"
	OrderTypeLimit  OrderType = "limit"
	OrderTypeStop   OrderType = "stop"
	OrderTypeStopLimit OrderType = "stop_limit"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusNew       OrderStatus = "new"
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPartial   OrderStatus = "partial"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusRejected  OrderStatus = "rejected"
)

// Order represents a trading order
type Order struct {
	ID            string        `json:"id"`
	Symbol        string        `json:"symbol"`
	Side          OrderSide     `json:"side"`
	Type          OrderType     `json:"type"`
	Quantity      float64       `json:"quantity"`
	Price         float64       `json:"price"`
	FilledQty     float64       `json:"filled_qty"`
	FilledPrice   float64       `json:"filled_price"`
	AvgFillPrice  float64       `json:"avg_fill_price"`
	Fee           float64       `json:"fee"`
	Status        OrderStatus   `json:"status"`
	CreateTime    time.Time     `json:"create_time"`
	UpdateTime    time.Time     `json:"update_time"`
	FillTime      *time.Time    `json:"fill_time,omitempty"`
	PositionType  PositionType  `json:"position_type"` // "long" or "short"
	StopPrice     float64       `json:"stop_price,omitempty"` // For stop orders
	TimeInForce   string        `json:"time_in_force"` // "GTC", "IOC", "FOK"
	ReduceOnly    bool          `json:"reduce_only"`
	ClientOrderID string        `json:"client_order_id,omitempty"`
}

// NewOrder creates a new order
func NewOrder(id, symbol string, side OrderSide, orderType OrderType, quantity, price float64, positionType PositionType) *Order {
	now := time.Now()
	return &Order{
		ID:           id,
		Symbol:       symbol,
		Side:         side,
		Type:         orderType,
		Quantity:     quantity,
		Price:        price,
		FilledQty:    0,
		FilledPrice:  0,
		AvgFillPrice: 0,
		Fee:          0,
		Status:       OrderStatusNew,
		CreateTime:   now,
		UpdateTime:   now,
		PositionType: positionType,
		TimeInForce:  "GTC", // Good Till Cancelled
		ReduceOnly:   false,
	}
}

// NewMarketOrder creates a new market order
func NewMarketOrder(id, symbol string, side OrderSide, quantity float64, positionType PositionType) *Order {
	return NewOrder(id, symbol, side, OrderTypeMarket, quantity, 0, positionType)
}

// NewLimitOrder creates a new limit order
func NewLimitOrder(id, symbol string, side OrderSide, quantity, price float64, positionType PositionType) *Order {
	return NewOrder(id, symbol, side, OrderTypeLimit, quantity, price, positionType)
}

// IsBuy returns true if this is a buy order
func (o *Order) IsBuy() bool {
	return o.Side == OrderSideBuy
}

// IsSell returns true if this is a sell order
func (o *Order) IsSell() bool {
	return o.Side == OrderSideSell
}

// IsFilled returns true if order is completely filled
func (o *Order) IsFilled() bool {
	return o.Status == OrderStatusFilled || o.FilledQty >= o.Quantity
}

// IsPartiallyFilled returns true if order is partially filled
func (o *Order) IsPartiallyFilled() bool {
	return o.FilledQty > 0 && o.FilledQty < o.Quantity
}

// IsActive returns true if order is still active
func (o *Order) IsActive() bool {
	return o.Status == OrderStatusNew || o.Status == OrderStatusPending || o.Status == OrderStatusPartial
}

// GetRemainingQty returns the remaining quantity to fill
func (o *Order) GetRemainingQty() float64 {
	return o.Quantity - o.FilledQty
}

// Fill fills the order with given quantity and price
func (o *Order) Fill(fillQty, fillPrice, fillFee float64) {
	totalFillQty := o.FilledQty + fillQty
	totalFillCost := (o.FilledQty * o.AvgFillPrice) + (fillQty * fillPrice)

	o.FilledQty = totalFillQty
	o.AvgFillPrice = totalFillCost / totalFillQty
	o.FilledPrice = fillPrice
	o.Fee += fillFee
	o.UpdateTime = time.Now()

	if totalFillQty >= o.Quantity {
		o.Status = OrderStatusFilled
		now := time.Now()
		o.FillTime = &now
	} else if totalFillQty > 0 {
		o.Status = OrderStatusPartial
	}
}

// Cancel cancels the order
func (o *Order) Cancel() {
	if o.IsActive() {
		o.Status = OrderStatusCancelled
		o.UpdateTime = time.Now()
	}
}

// GetFillPercentage returns what percentage of the order has been filled
func (o *Order) GetFillPercentage() float64 {
	if o.Quantity == 0 {
		return 0
	}
	return (o.FilledQty / o.Quantity) * 100
}

// GetNotionalValue returns the total value of the order (quantity × price)
func (o *Order) GetNotionalValue() float64 {
	return o.Quantity * o.Price
}

// GetFilledNotionalValue returns the value of filled portion
func (o *Order) GetFilledNotionalValue() float64 {
	return o.FilledQty * o.AvgFillPrice
}

// SetStopPrice sets the stop price for stop orders
func (o *Order) SetStopPrice(stopPrice float64) {
	o.StopPrice = stopPrice
}

// SetReduceOnly sets the reduce-only flag
func (o *Order) SetReduceOnly(reduceOnly bool) {
	o.ReduceOnly = reduceOnly
}

// GetEffectivePrice returns the effective price including fees
func (o *Order) GetEffectivePrice() float64 {
	if o.AvgFillPrice == 0 {
		return o.Price
	}

	feeRate := o.Fee / o.GetFilledNotionalValue()
	if o.IsBuy() {
		return o.AvgFillPrice * (1 + feeRate)
	}
	return o.AvgFillPrice * (1 - feeRate)
}