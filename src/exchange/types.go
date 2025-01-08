package exchange

type ExchangeResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
}

type Candle struct {
	OpenTime  int64   `json:"open_time"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
	CloseTime int64   `json:"close_time"`
}

type TradeSignal struct {
	SignalType string  `json:"signal_type"` // "buy" or "sell"
	Price      float64 `json:"price"`
	Risk       float64 `json:"risk"` // Risk assessment for the trade
}
