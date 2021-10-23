package binance

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/WinPooh32/fixed"
	"github.com/WinPooh32/retrade/platform"
	"github.com/adshao/go-binance/v2"
)

var ErrClosed = fmt.Errorf("connection closed")

type Binance struct {
	once  sync.Once
	stopC chan struct{}
	doneC chan struct{}

	client *binance.Client
}

func New(testnet bool, apiKey, secretKey string) *Binance {
	binance.UseTestnet = testnet

	var b = Binance{
		client: binance.NewClient(apiKey, secretKey),
	}

	b.client.UserAgent = "retrade/1.0"

	return &b
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

	var err error

	b.doneC, b.stopC, err = binance.WsAggTradeServe(symbol, wsAggTradeHandler, errHandler)
	if err != nil {
		sink <- platform.MakeError(fmt.Errorf("go-binance: %w", err))
	}

	go func() {
		for {
			select {
			case e := <-sink:
				events <- e
			case <-b.doneC:
				events <- platform.MakeError(ErrClosed)
				close(events)
				return
			case <-ctx.Done():
				events <- platform.MakeError(ctx.Err())
				<-b.doneC
				close(events)
				return
			}
		}
	}()

	return events
}

func (b *Binance) Wallet(ctx context.Context) (wallet map[string]platform.Fixed, err error) {
	res, err := b.client.NewGetAccountService().Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	wallet = make(map[string]fixed.Fixed, len(res.Balances))

	for _, b := range res.Balances {
		free, err := fixed.Parse(b.Free)
		if err != nil {
			return nil, fmt.Errorf("parse value=%s: %w", b.Free, err)
		}
		wallet[b.Asset] = free
	}

	return wallet, nil
}

func (b *Binance) OrderMarket(ctx context.Context, symbol string, side platform.OrderSide, quantity platform.Fixed) (orderID string, err error) {
	req := b.client.NewCreateOrderService().
		Symbol(symbol).
		Type(binance.OrderTypeMarket)

	switch side {
	case platform.OrderSideBuy:
		req.
			Side(binance.SideTypeBuy).
			QuoteOrderQty(quantity.String())

	case platform.OrderSideSell:
		req.
			Side(binance.SideTypeSell).
			Quantity(quantity.String())
	}

	res, err := req.Do(ctx)
	if err != nil {
		return "", fmt.Errorf("post market order: %w", err)
	}

	return strconv.FormatInt(res.OrderID, 10), nil
}

func (b *Binance) OrderOCO(ctx context.Context, symbol string, side platform.OrderSide, opt platform.OptionsOCO) (orderID string, err error) {
	req := b.client.NewCreateOCOService().
		Symbol(symbol).
		Price(opt.Price.String()).
		StopPrice(opt.Stop.String()).
		StopLimitPrice(opt.Limit.String()).
		StopLimitTimeInForce(binance.TimeInForceTypeGTC).
		Quantity(opt.Quantity.String())

	switch side {
	case platform.OrderSideBuy:
		req.Side(binance.SideTypeBuy)
	case platform.OrderSideSell:
		req.Side(binance.SideTypeSell)
	}

	res, err := req.Do(ctx)
	if err != nil {
		return "", fmt.Errorf("post OCO order: %w", err)
	}

	return strconv.FormatInt(res.OrderListID, 10), nil
}

func (b *Binance) Close() error {
	b.once.Do(func() {
		close(b.stopC)
	})
	<-b.doneC
	return nil
}

func init() {
	binance.WebsocketKeepalive = true
}
