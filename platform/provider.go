package platform

import (
	"context"

	"github.com/WinPooh32/fixed"
)

type Fixed = fixed.Fixed

type Spot interface {
	OrderMarket(ctx context.Context, symbol string, side OrderSide, quantity Fixed) (orderID string, err error)
	OrderOCO(ctx context.Context, symbol string, side OrderSide, opt OptionsOCO) (orderID string, err error)
	Cancel(ctx context.Context, symbol string, orderID string) (err error)
	CancelAll(ctx context.Context, symbol string) (err error)
	QueryOrder(ctx context.Context) (err error)
	ListOrders(ctx context.Context, symbol string) (orders []Order, err error)
}

type Account interface {
	Wallet(ctx context.Context) (wallet map[string]Fixed, err error)
}

type Public interface {
	Subscribe(ctx context.Context, symbol string) (events <-chan EventContainer)
}
