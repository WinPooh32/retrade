package backtest

import (
	"context"

	"github.com/WinPooh32/retrade/platform"
)

type NopProvider struct{}

var _ Provider = &NopProvider{}

func (*NopProvider) Subscribe(ctx context.Context, symbol platform.Symbol) (events <-chan platform.EventContainer) {
	return nil
}

func (*NopProvider) OrderMarket(ctx context.Context, symbol platform.Symbol, side platform.OrderSide, quantity platform.Fixed) (orderID string, err error) {
	return
}

func (*NopProvider) OrderOCO(ctx context.Context, symbol platform.Symbol, side platform.OrderSide, opt platform.OptionsOCO) (orderID string, err error) {
	return
}

func (*NopProvider) Cancel(ctx context.Context, symbol platform.Symbol, orderID string) (status string, err error) {
	return
}

func (*NopProvider) CancelAll(ctx context.Context, symbol platform.Symbol) (err error) { return }

func (*NopProvider) QueryOrder(ctx context.Context) (err error) { return }

func (*NopProvider) ListOrders(ctx context.Context, symbol platform.Symbol) (orders []platform.Order, err error) {
	return
}

func (*NopProvider) Wallet(ctx context.Context) (wallet map[platform.Symbol]platform.Fixed, err error) {
	return
}
