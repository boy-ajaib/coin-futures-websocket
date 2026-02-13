package types

const (
	// TopicUserMargin is the Kafka topic for user margin updates
	TopicUserMargin = "com.ajaib.coin.cfx.streamer.futures.message.UserMargin"

	// TopicUserPosition is the Kafka topic for user position updates
	TopicUserPosition = "com.ajaib.coin.cfx.streamer.futures.message.UserPosition"

	// ChannelMarginSuffix is the WebSocket channel suffix for margin data
	ChannelMarginSuffix = "margin"

	// ChannelPositionSuffix is the WebSocket channel suffix for position data
	ChannelPositionSuffix = "position"
)

// UserMargin represents a user's margin account state from Kafka
type UserMargin struct {
	Timestamp          int64   `json:"timestamp"`
	CFXUserID          string  `json:"cfx_user_id"`
	Asset              string  `json:"asset"`
	TotalPositionValue float64 `json:"total_position_value"`
	MarginBalance      float64 `json:"margin_balance"`
	OrderMargin        float64 `json:"order_margin"`
	EffectiveLeverage  float64 `json:"effective_leverage"`
	MaintenanceMargin  float64 `json:"maintenance_margin"`
	UnrealizedPnl      float64 `json:"unrealized_pnl"`
	AvailableMargin    float64 `json:"available_margin"`
	WalletBalance      float64 `json:"wallet_balance"`
	MarginRatio        float64 `json:"margin_ratio"`
	WithdrawableMargin float64 `json:"withdrawable_margin"`
}

// UserPosition represents a user's futures position from Kafka
type UserPosition struct {
	Timestamp                int64   `json:"timestamp"`
	CFXUserID                string  `json:"cfx_user_id"`
	Symbol                   string  `json:"symbol"`
	Size                     float64 `json:"size"`
	Value                    float64 `json:"value"`
	Leverage                 int     `json:"leverage"`
	EntryPrice               float64 `json:"entry_price"`
	MarkPrice                float64 `json:"mark_price"`
	LiquidationPrice         float64 `json:"liquidation_price"`
	MaintenanceMargin        float64 `json:"maintenance_margin"`
	RealisedPnl              float64 `json:"realised_pnl"`
	UnrealisedPnl            float64 `json:"unrealised_pnl"`
	DeleveragePercentile     float64 `json:"deleverage_percentile"`
	RiskLimit                int64   `json:"risk_limit"`
	OpenOrderBuyCost         float64 `json:"open_order_buy_cost"`
	OpenOrderSellCost        float64 `json:"open_order_sell_cost"`
	InitialMarginRequirement float64 `json:"initial_margin_requirement"`
	UpdatedTime              int64   `json:"updated_time"`
	OpenOrderBuyQuantity     float64 `json:"open_order_buy_quantity"`
	OpenOrderSellQuantity    float64 `json:"open_order_sell_quantity"`
	OrderMargin              float64 `json:"order_margin"`
}

// GetCFXUserID returns the CFX user ID for this margin data
func (m *UserMargin) GetCFXUserID() string {
	return m.CFXUserID
}

// GetCFXUserID returns the CFX user ID for this position data
func (p *UserPosition) GetCFXUserID() string {
	return p.CFXUserID
}
