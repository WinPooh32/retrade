package binance

import (
	"context"
	"fmt"

	"github.com/WinPooh32/fixed"
	"github.com/WinPooh32/retrade/platform"
	"github.com/adshao/go-binance/v2"
)

var ErrClosed = fmt.Errorf("connection closed")

type Binance struct {
}

func New() *Binance {
	return &Binance{}
}

func (b *Binance) Subscribe(ctx context.Context, symbol string) <-chan platform.EventContainer {
	events := make(chan platform.EventContainer, 1024)

	sink := make(chan platform.EventContainer, 1024)

	wsAggTradeHandler := func(event *binance.WsAggTradeEvent) {
		t := platform.Trade{
			TradeID:      event.AggTradeID,
			Time:         event.TradeTime,
			Historic:     false,
			Symbol:       symbol,
			Price:        fixed.NewS(event.Price),
			Quantity:     fixed.NewS(event.Quantity),
			IsBuyerMaker: event.IsBuyerMaker,
		}
		sink <- platform.MakeTrade(t)
	}

	errHandler := func(err error) {
		sink <- platform.MakeError(fmt.Errorf("go-binance: %w", err))
	}

	doneC, _, err := binance.WsAggTradeServe(symbol, wsAggTradeHandler, errHandler)
	if err != nil {
		sink <- platform.MakeError(fmt.Errorf("go-binance: %w", err))
	}

	go func() {
		for {
			select {
			case e := <-sink:
				events <- e
			case <-doneC:
				events <- platform.MakeError(ErrClosed)
				close(events)
				return
			case <-ctx.Done():
				events <- platform.MakeError(ctx.Err())
				<-doneC
				close(events)
				return
			}
		}
	}()

	return events
}

func init() {
	binance.WebsocketKeepalive = true
}
