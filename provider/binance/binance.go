package binance

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/WinPooh32/fixed"
	"github.com/WinPooh32/retrade/platform"
	"github.com/adshao/go-binance/v2"
)

var ErrClosed = fmt.Errorf("connection closed")

type control struct {
	stopC chan struct{}
	doneC chan struct{}
}

type Binance struct {
	once sync.Once

	trades control
	books  control

	client *binance.Client
}

func New(testnet bool, apiKey, secretKey string) (*Binance, error) {
	binance.UseTestnet = testnet

	var b = Binance{
		client: binance.NewClient(apiKey, secretKey),
	}

	b.client.UserAgent = "retrade/1.0"

	_, err := b.client.NewSetServerTimeService().Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("client: set server time: %w", err)
	}

	return &b, nil
}

func (b *Binance) Subscribe(ctx context.Context, symbol platform.Symbol) <-chan platform.EventContainer {
	events := make(chan platform.EventContainer, 1024)

	sink := make(chan platform.EventContainer, 1024)

	go func() {
		defer func() {
			var err = b.Close()
			if err != nil {
				events <- platform.MakeError(fmt.Errorf("close: %w", err))
			}
			close(events)
		}()

		for {
			select {
			case e := <-sink:
				events <- e
			case <-b.books.doneC:
				events <- platform.MakeError(ErrClosed)
				return
			case <-b.trades.doneC:
				events <- platform.MakeError(ErrClosed)
				return
			case <-ctx.Done():
				events <- platform.MakeError(ctx.Err())
				return
			}
		}
	}()

	errHandler := func(err error) {
		sink <- platform.MakeError(fmt.Errorf("go-binance: %w", err))
	}

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

	wsBookTickerHandler := func(event *binance.WsBookTickerEvent) {
		b := platform.BookTicker{
			Time:         time.Now().UnixNano()/int64(time.Millisecond) - b.client.TimeOffset,
			UpdateID:     strconv.FormatInt(event.UpdateID, 10),
			BestBidPrice: fixed.NewS(event.BestBidPrice),
			BestBidQty:   fixed.NewS(event.BestBidQty),
			BestAskPrice: fixed.NewS(event.BestAskPrice),
			BestAskQty:   fixed.NewS(event.BestAskQty),
		}
		sink <- platform.MakeBookTicker(b)
	}

	var err error

	b.trades.doneC, b.trades.stopC, err = binance.WsAggTradeServe(string(symbol), wsAggTradeHandler, errHandler)
	if err != nil {
		sink <- platform.MakeError(fmt.Errorf("go-binance: %w", err))
		return events
	}

	b.books.doneC, b.books.stopC, err = binance.WsBookTickerServe(string(symbol), wsBookTickerHandler, errHandler)
	if err != nil {
		sink <- platform.MakeError(fmt.Errorf("go-binance: %w", err))
		return events
	}

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

func (b *Binance) OrderMarket(ctx context.Context, symbol platform.Symbol, side platform.OrderSide, quantity platform.Fixed) (orderID string, err error) {
	req := b.client.NewCreateOrderService().
		Symbol(string(symbol)).
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

func (b *Binance) OrderOCO(ctx context.Context, symbol platform.Symbol, side platform.OrderSide, opt platform.OptionsOCO) (orderID string, err error) {
	req := b.client.NewCreateOCOService().
		Symbol(string(symbol)).
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

	return strconv.FormatInt(res.Orders[0].OrderID, 10), nil
}

func (b *Binance) Cancel(ctx context.Context, symbol platform.Symbol, orderID string) (status string, err error) {
	id, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("orderID=%s: parse int: %w", orderID, err)
	}

	req := b.client.NewCancelOrderService().
		Symbol(string(symbol)).
		OrderID(id)

	res, err := req.Do(ctx)
	if err != nil {
		return "", fmt.Errorf("cancel order=%s: %w", orderID, err)
	}

	return string(res.Status), nil
}

func (b *Binance) CancelAll(ctx context.Context, symbol platform.Symbol) (err error) {
	req := b.client.NewCancelOpenOrdersService().
		Symbol(string(symbol))

	_, err = req.Do(ctx)
	if err != nil {
		return fmt.Errorf("cancel all orders: %w", err)
	}
	return nil
}

func (b *Binance) QueryOrder(ctx context.Context, symbol platform.Symbol, orderID string) (err error) {
	orderIDInt64, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return fmt.Errorf("parse orderID: parse int: %w", err)
	}

	req := b.client.NewGetOrderService().
		Symbol(string(symbol)).
		OrderID(orderIDInt64)

	_, err = req.Do(ctx)
	if err != nil {
		return fmt.Errorf("query order: %w", err)
	}
	return nil
}

func (b *Binance) ListOrders(ctx context.Context, symbol platform.Symbol) (orders []platform.Order, err error) {
	req := b.client.NewListOpenOrdersService().
		Symbol(string(symbol))

	res, err := req.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("list orders orders: %w", err)
	}

	orders = make([]platform.Order, 0, len(res))
	for _, o := range res {
		orders = append(orders, platform.Order{
			Symbol:                   o.Symbol,
			OrderID:                  strconv.FormatInt(o.OrderID, 10),
			Price:                    o.Price,
			OrigQuantity:             o.OrigQuantity,
			ExecutedQuantity:         o.ExecutedQuantity,
			CummulativeQuoteQuantity: o.CummulativeQuoteQuantity,
			Status:                   string(o.Status),
			Type:                     string(o.Type),
			Side:                     string(o.Side),
			StopPrice:                o.StopPrice,
			Time:                     o.Time,
		})
	}

	return orders, nil
}

func (b *Binance) Close() error {
	b.once.Do(func() {
		close(b.trades.stopC)
		close(b.books.stopC)
	})
	<-b.trades.doneC
	<-b.books.stopC
	return nil
}

func init() {
	binance.WebsocketKeepalive = true
}
