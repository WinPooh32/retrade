package backtest

import (
	"context"
	"fmt"

	"github.com/WinPooh32/fixed"
	"github.com/WinPooh32/retrade/candle"
	"github.com/WinPooh32/retrade/platform"
	"github.com/WinPooh32/series"
)

type HistorySnaphsot struct {
	Price candle.HistoryFloat32

	BuyBestCount  candle.HistoryFloat32
	SellBestCount candle.HistoryFloat32

	BuyBestVolume  candle.HistoryFloat32
	SellBestVolume candle.HistoryFloat32

	BestAsk candle.HistoryFloat32
	BestBid candle.HistoryFloat32
}

type Strategy interface {
	Name() string
	BuySignal(snap HistorySnaphsot) bool
	SellSignal(snap HistorySnaphsot) bool
}

type Options struct {
	Symbol            string
	FeeBuy            float32
	FeeSell           float32
	Account           float32
	FramePeriod       int64
	HistoryWindowSize int64
	Limit             float32
}

type Result struct {
	Account series.Data
	Buy     series.Data
	Sell    series.Data
	Pool    float32
}

const (
	buy  = 0
	sell = 1
)

type runstate struct {
	side    int
	account float32
	pool    float32

	price *candle.Candle

	buyBestCount  *candle.Candle
	sellBestCount *candle.Candle

	buyBestVolume  *candle.Candle
	sellBestVolume *candle.Candle

	bestAsk *candle.Candle
	bestBid *candle.Candle
}

func (state *runstate) doBuy(strategy Strategy, snap HistorySnaphsot, opt Options) (buyTime int64, price float32, ok bool) {
	var closePrice fixed.Fixed

	buyTime, _, _, _, closePrice, _ = state.price.Last()
	price = float32(closePrice.Float())

	if strategy.BuySignal(snap) {
		state.account = state.buy(state.account, price, opt.FeeBuy)
		state.side = sell

		ok = true
		return
	}

	ok = false
	return
}

func (state *runstate) doSell(strategy Strategy, snap HistorySnaphsot, opt Options) (buyTime int64, price float32, account float32, ok bool) {
	var closePrice fixed.Fixed

	buyTime, _, _, _, closePrice, _ = state.price.Last()
	price = float32(closePrice.Float())

	if strategy.SellSignal(snap) {
		state.account = state.sell(state.account, price, opt.FeeSell)
		state.side = buy

		if opt.Limit != 0 && state.account > opt.Limit {
			state.pool += state.account - opt.Limit
			state.account = opt.Limit
		}

		account = state.account
		ok = true
		return
	}

	ok = false
	return
}

func (*runstate) buy(account float32, price float32, fee float32) float32 {
	fee = account * fee
	return (account - fee) / price
}

func (*runstate) sell(account float32, price float32, fee float32) float32 {
	fee = account * fee
	return (account - fee) * price
}

type Runner struct {
	buyTime   []int64
	buyPrice  []float32
	sellTime  []int64
	sellPrice []float32
	account   []float32
}

func NewRunner() *Runner {
	return &Runner{
		buyTime:   make([]int64, 0, 1024),
		buyPrice:  make([]float32, 0, 1024),
		sellTime:  make([]int64, 0, 1024),
		sellPrice: make([]float32, 0, 1024),
		account:   make([]float32, 0, 1024),
	}
}

func (runner *Runner) Test(ctx context.Context, provider platform.Public, strategy Strategy, opt Options) (result Result, err error) {
	var (
		next         bool
		finishedTick int64
		tick         int64

		state = runstate{
			side:    buy,
			account: opt.Account,
			pool:    0.0,

			price:          candle.NewCandle(opt.FramePeriod, int(opt.HistoryWindowSize)),
			buyBestCount:   candle.NewCandle(opt.FramePeriod, int(opt.HistoryWindowSize)),
			sellBestCount:  candle.NewCandle(opt.FramePeriod, int(opt.HistoryWindowSize)),
			buyBestVolume:  candle.NewCandle(opt.FramePeriod, int(opt.HistoryWindowSize)),
			sellBestVolume: candle.NewCandle(opt.FramePeriod, int(opt.HistoryWindowSize)),
			bestAsk:        candle.NewCandle(opt.FramePeriod, int(opt.HistoryWindowSize)),
			bestBid:        candle.NewCandle(opt.FramePeriod, int(opt.HistoryWindowSize)),
		}
	)

	for event := range provider.Subscribe(ctx, opt.Symbol) {
		switch event.Type {
		case platform.EventErr:
			err = fmt.Errorf("provider: event: %w", event.Error)
			return

		case platform.EventCandle:
			c := event.Event.Candle
			state.price.AppendRaw(
				c.Time,
				c.Open,
				c.High,
				c.Low,
				c.Close,
				c.Volume,
			)
			tick = c.Time / opt.FramePeriod
			next = true

		case platform.EventTrade:
			t := event.Event.Trade
			tick = t.Time / opt.FramePeriod

			var (
				tSellCount  platform.Trade
				tSellVolume platform.Trade

				tBuyCount  platform.Trade
				tBuyVolume platform.Trade
			)

			if t.IsBuyerMaker {
				// Solt by market.
				tSellVolume = platform.Trade{
					Time:     t.Time,
					Quantity: t.Quantity,
				}
				tSellCount = platform.Trade{
					Time:     t.Time,
					Quantity: fixed.NewI(1, 0),
				}

				tBuyVolume = platform.Trade{
					Time: t.Time,
				}
				tBuyCount = platform.Trade{
					Time: t.Time,
				}
			} else {
				// Buyed by market.
				tSellVolume = platform.Trade{
					Time: t.Time,
				}
				tSellCount = platform.Trade{
					Time: t.Time,
				}

				tBuyVolume = platform.Trade{
					Time:     t.Time,
					Quantity: t.Quantity,
				}
				tBuyCount = platform.Trade{
					Time:     t.Time,
					Quantity: fixed.NewI(1, 0),
				}
			}

			state.sellBestCount.Add(tSellCount)
			state.sellBestVolume.Add(tSellVolume)

			state.buyBestCount.Add(tBuyCount)
			state.buyBestVolume.Add(tBuyVolume)

			next = state.price.Add(t)

		case platform.EventBookTicker:
			b := event.Event.BookTicker
			tick = b.Time / opt.FramePeriod

			state.bestAsk.Add(platform.Trade{
				Time:     b.Time,
				Price:    b.BestAskPrice,
				Quantity: b.BestAskQty,
			})

			next = state.bestBid.Add(platform.Trade{
				Time:     b.Time,
				Price:    b.BestBidPrice,
				Quantity: b.BestBidQty,
			})
		}

		if next && tick > finishedTick {
			finishedTick = tick

			var snap = HistorySnaphsot{
				Price:          state.price.HistoryFloat32(),
				BuyBestCount:   state.buyBestCount.HistoryFloat32(),
				BuyBestVolume:  state.buyBestVolume.HistoryFloat32(),
				SellBestCount:  state.sellBestCount.HistoryFloat32(),
				SellBestVolume: state.sellBestVolume.HistoryFloat32(),
				BestAsk:        state.bestAsk.HistoryFloat32(),
				BestBid:        state.bestBid.HistoryFloat32(),
			}

			switch state.side {
			case buy:
				var ts, price, ok = state.doBuy(strategy, snap, opt)
				if ok {
					runner.buyTime = append(runner.buyTime, ts)
					runner.buyPrice = append(runner.buyPrice, price)
				}
			case sell:
				var ts, price, account, ok = state.doSell(strategy, snap, opt)
				if ok {
					runner.sellTime = append(runner.sellTime, ts)
					runner.sellPrice = append(runner.sellPrice, price)
					runner.account = append(runner.account, account)
				}
			}
		}
	}

	if state.side == sell {
		var snap = HistorySnaphsot{
			Price:          state.price.HistoryFloat32(),
			BuyBestCount:   state.buyBestCount.HistoryFloat32(),
			BuyBestVolume:  state.buyBestVolume.HistoryFloat32(),
			SellBestCount:  state.sellBestCount.HistoryFloat32(),
			SellBestVolume: state.sellBestVolume.HistoryFloat32(),
			BestAsk:        state.bestAsk.HistoryFloat32(),
			BestBid:        state.bestBid.HistoryFloat32(),
		}

		var ts, price, account, ok = state.doSell(strategy, snap, opt)
		if ok {
			runner.sellTime = append(runner.sellTime, ts)
			runner.sellPrice = append(runner.sellPrice, price)
			runner.account = append(runner.account, account)
		}
	}

	return Result{
		Buy:     series.MakeData(1, runner.buyTime, runner.buyPrice),
		Sell:    series.MakeData(1, runner.sellTime, runner.sellPrice),
		Account: series.MakeData(1, runner.sellTime, runner.account),
		Pool:    state.pool,
	}, nil
}
