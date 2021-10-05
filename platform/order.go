package platform

import "encoding/json"

type OrderSide string

const (
	OrderSideBuy  = "buy"
	OrderSideSell = "sell"
)

type OrderType string

const (
	OrderTypeMarket = "MARKET"
	OrderTypeLimit  = "LIMIT"
)

type Status string

const (
	StatusFilled = "FILLED"
)

type Order struct {
	Raw json.RawMessage
}
