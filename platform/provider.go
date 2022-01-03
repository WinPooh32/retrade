package platform

import (
	"context"

	"github.com/WinPooh32/fixed"
)

type Fixed = fixed.Fixed

type Symbol string

type Spot interface {
	OrderMarket(ctx context.Context, symbol Symbol, side OrderSide, quantity Fixed) (orderID string, err error)
	OrderOCO(ctx context.Context, symbol Symbol, side OrderSide, opt OptionsOCO) (orderID string, err error)
	Cancel(ctx context.Context, symbol Symbol, orderID string) (status string, err error)
	CancelAll(ctx context.Context, symbol Symbol) (err error)
	QueryOrder(ctx context.Context) (err error)
	ListOrders(ctx context.Context, symbol Symbol) (orders []Order, err error)
}

type Account interface {
	Wallet(ctx context.Context) (wallet map[Symbol]Fixed, err error)
}

type Public interface {
	Subscribe(ctx context.Context, symbol Symbol) (events <-chan EventContainer)
}
