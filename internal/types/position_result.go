package types

import (
	"time"
)

// PositionResult represents the result of a position operation
type PositionResult struct {
	// Trade details
	Symbol      string    `json:"symbol"`
	Action      string    `json:"action"`      // "open", "close", "modify"
	Direction   string    `json:"direction"`   // "long", "short"
	Quantity    float64   `json:"quantity"`
	Price       float64   `json:"price"`
	Value       float64   `json:"value"`       // quantity * price

	// Execution details
	OrderID     string    `json:"order_id"`
	FilledQty   float64   `json:"filled_qty"`
	FilledPrice float64   `json:"filled_price"`
	Commission  float64   `json:"commission"`
	Slippage    float64   `json:"slippage"`

	// Timing
	CreateTime  time.Time `json:"create_time"`
	ExecuteTime time.Time `json:"execute_time"`
	Duration    time.Duration `json:"duration"`

	// Position information
	EntryPrice  float64   `json:"entry_price,omitempty"`
	ExitPrice   float64   `json:"exit_price,omitempty"`
	PnL         float64   `json:"pnl,omitempty"`
	ROI         float64   `json:"roi,omitempty"`

	// Status
	Status      string    `json:"status"`      // "success", "partial", "failed"
	Reason      string    `json:"reason,omitempty"`
	Error       string    `json:"error,omitempty"`

	// Risk metrics
	RiskAmount  float64   `json:"risk_amount,omitempty"`
	StopLoss    float64   `json:"stop_loss,omitempty"`
	TakeProfit  float64   `json:"take_profit,omitempty"`

	// Metadata
	Strategy    string    `json:"strategy,omitempty"`
	Confidence  float64   `json:"confidence,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewPositionResult creates a new position result
func NewPositionResult(symbol, action, direction string) *PositionResult {
	now := time.Now()
	return &PositionResult{
		Symbol:     symbol,
		Action:     action,
		Direction:  direction,
		CreateTime: now,
		Status:     "pending",
	}
}

// SetSuccess marks the result as successful
func (pr *PositionResult) SetSuccess() {
	pr.Status = "success"
	pr.ExecuteTime = time.Now()
	pr.Duration = pr.ExecuteTime.Sub(pr.CreateTime)
}

// SetPartial marks the result as partially filled
func (pr *PositionResult) SetPartial() {
	pr.Status = "partial"
	pr.ExecuteTime = time.Now()
	pr.Duration = pr.ExecuteTime.Sub(pr.CreateTime)
}

// SetFailed marks the result as failed
func (pr *PositionResult) SetFailed(reason, err string) {
	pr.Status = "failed"
	pr.Reason = reason
	pr.Error = err
	pr.ExecuteTime = time.Now()
	pr.Duration = pr.ExecuteTime.Sub(pr.CreateTime)
}

// IsSuccessful returns true if the operation was successful
func (pr *PositionResult) IsSuccessful() bool {
	return pr.Status == "success"
}

// IsPartial returns true if the operation was partially filled
func (pr *PositionResult) IsPartial() bool {
	return pr.Status == "partial"
}

// IsFailed returns true if the operation failed
func (pr *PositionResult) IsFailed() bool {
	return pr.Status == "failed"
}

// GetEffectivePrice returns the effective execution price including slippage
func (pr *PositionResult) GetEffectivePrice() float64 {
	if pr.FilledPrice > 0 {
		return pr.FilledPrice
	}
	return pr.Price
}

// GetTotalCost returns the total cost including commission
func (pr *PositionResult) GetTotalCost() float64 {
	return pr.Value + pr.Commission
}

// GetNetPnL returns the net PnL after commission
func (pr *PositionResult) GetNetPnL() float64 {
	return pr.PnL - pr.Commission
}

// SetPositionInfo sets position-related information
func (pr *PositionResult) SetPositionInfo(entryPrice, exitPrice, pnl float64) {
	pr.EntryPrice = entryPrice
	pr.ExitPrice = exitPrice
	pr.PnL = pnl

	// Calculate ROI
	if entryPrice > 0 && pr.Quantity != 0 {
		investment := entryPrice * pr.Quantity
		if investment != 0 {
			pr.ROI = (pnl / investment) * 100
		}
	}
}

// SetRiskInfo sets risk-related information
func (pr *PositionResult) SetRiskInfo(riskAmount, stopLoss, takeProfit float64) {
	pr.RiskAmount = riskAmount
	pr.StopLoss = stopLoss
	pr.TakeProfit = takeProfit
}

// SetMetadata sets metadata for the result
func (pr *PositionResult) SetMetadata(key string, value interface{}) {
	if pr.Metadata == nil {
		pr.Metadata = make(map[string]interface{})
	}
	pr.Metadata[key] = value
}

// Clone creates a copy of the position result
func (pr *PositionResult) Clone() *PositionResult {
	clone := *pr

	// Deep copy metadata
	if pr.Metadata != nil {
		clone.Metadata = make(map[string]interface{})
		for k, v := range pr.Metadata {
			clone.Metadata[k] = v
		}
	}

	return &clone
}