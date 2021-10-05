package platform

import (
	"context"

	"github.com/WinPooh32/fixed"
)

type Fixed = fixed.Fixed

type Spot interface {
	Order(ctx context.Context, symbol string, side OrderSide, quantity Fixed) (orderID string, err error)
	Cancel(ctx context.Context, orderID string) (err error)
	CancelAll(ctx context.Context) (err error)
	QueryOrder(ctx context.Context) (err error)
	ListOrders(ctx context.Context) (err error)
}

type Account interface {
	Wallet(ctx context.Context) (wallet map[string]Fixed, err error)
}

type Public interface {
	Subscribe(ctx context.Context, symbol string) (events chan<- EventContainer)
}
