package platform

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
	Symbol                   string
	OrderID                  string
	Price                    string
	OrigQuantity             string
	ExecutedQuantity         string
	CummulativeQuoteQuantity string
	Status                   string
	Type                     string
	Side                     string
	StopPrice                string
	Time                     int64
}

type OptionsOCO struct {
	Price    Fixed
	Stop     Fixed
	Limit    Fixed
	Quantity Fixed
}
