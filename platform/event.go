package platform

type EventType int

const (
	EventErr EventType = iota
	EventCandle
	EventTrade
)

type EventContainer struct {
	Type  EventType
	Event Event
	Error error
}

type Event struct {
	Trade  Trade
	Candle Candle
}

type Trade struct {
	TradeID      int64
	Time         int64
	Historic     bool
	Symbol       string
	Price        Fixed
	Quantity     Fixed
	IsBuyerMaker bool
}

type Candle struct {
	Time                int64
	Open                Fixed
	High                Fixed
	Low                 Fixed
	Close               Fixed
	Volume              Fixed
	TimeClose           int64
	VolumeQuote         Fixed
	CountTrades         int64
	VolumeTakerBuyBase  Fixed
	VolumeTakerBuyQuote Fixed
}

func MakeTrade(t Trade) EventContainer {
	return EventContainer{
		Type: EventTrade,
		Event: Event{
			Trade: t,
		},
	}
}

func MakeCandle(c Candle) EventContainer {
	return EventContainer{
		Type: EventCandle,
		Event: Event{
			Candle: c,
		},
	}
}

func MakeError(err error) EventContainer {
	return EventContainer{
		Type:  EventErr,
		Error: err,
	}
}
