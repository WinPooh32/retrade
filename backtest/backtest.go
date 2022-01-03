package backtest

import (
	"context"
	"fmt"
	"math"

	"github.com/WinPooh32/fixed"
	"github.com/WinPooh32/retrade/candle"
	"github.com/WinPooh32/retrade/platform"
	"github.com/WinPooh32/retrade/ring/ringmapf64"
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

	VolumeClusters []map[float64]float64
}

type Strategy interface {
	Name() string
	BuySignal(snap HistorySnaphsot) bool
	SellSignal(snap HistorySnaphsot) bool
}

type Options struct {
	Symbol            platform.Symbol
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

type Provider interface {
	platform.Public
	platform.Spot
	platform.Account
}

type eventHandler struct {
	opt    Options
	runner *Runner
	state  runstate
}

type stats struct {
	buyTime   []int64
	buyPrice  []float32
	sellTime  []int64
	sellPrice []float32
	account   []float32
}

type Runner struct {
	provider Provider
	stats    stats
}

func NewRunner(provider Provider) *Runner {
	const defaultCap = 1024
	return &Runner{
		provider: provider,
		stats: stats{
			buyTime:   make([]int64, 0, defaultCap),
			buyPrice:  make([]float32, 0, defaultCap),
			sellTime:  make([]int64, 0, defaultCap),
			sellPrice: make([]float32, 0, defaultCap),
			account:   make([]float32, 0, defaultCap),
		},
	}
}

func (runner *Runner) Run(ctx context.Context, strategy Strategy, opt Options) (result Result, err error) {
	var handler = eventHandler{
		opt:    opt,
		runner: runner,
		state: runstate{
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

			volumeClusters:       ringmapf64.MakeRing(int(opt.HistoryWindowSize)),
			volumeClustersMoment: map[float64]float64{},

			clusters: make([]map[float64]float64, 0, int(opt.HistoryWindowSize)),
		},
	}

	for event := range runner.provider.Subscribe(ctx, opt.Symbol) {
		var err error

		switch event.Type {
		case platform.EventErr:
			err = fmt.Errorf("provider: event: %w", event.Error)

		case platform.EventCandle:
			if err = handler.onCandle(ctx, event.Event.Candle); err != nil {
				err = fmt.Errorf("handler: on Candle event: %w", err)
			}

		case platform.EventTrade:
			if err = handler.onTrade(ctx, event.Event.Trade); err != nil {
				err = fmt.Errorf("handler: on Trade event: %w", err)
			}

		case platform.EventBookTicker:
			if err = handler.onBookTicker(ctx, event.Event.BookTicker); err != nil {
				err = fmt.Errorf("handler: on Book Ticker event: %w", err)
			}
		}

		if err != nil {
			return result, err
		}

		if state := &handler.state; state.next && state.tick > state.finishedTick {
			runner.DoSideAction(state, strategy, opt)
		}
	}

	result = Result{
		Buy:     series.MakeData(1, runner.stats.buyTime, runner.stats.buyPrice),
		Sell:    series.MakeData(1, runner.stats.sellTime, runner.stats.sellPrice),
		Account: series.MakeData(1, runner.stats.sellTime, runner.stats.account),
		Pool:    handler.state.pool,
	}

	return result, nil
}

func (runner *Runner) DoSideAction(state *runstate, strategy Strategy, opt Options) {
	state.finishedTick = state.tick

	state.volumeClusters.ForcePushBack(state.volumeClustersMoment)
	n := state.volumeClusters.CopyTo(state.clusters[:state.volumeClusters.Len()])

	state.volumeClustersMoment = map[float64]float64{}

	var snap = HistorySnaphsot{
		Price:          state.price.HistoryFloat32(),
		BuyBestCount:   state.buyBestCount.HistoryFloat32(),
		BuyBestVolume:  state.buyBestVolume.HistoryFloat32(),
		SellBestCount:  state.sellBestCount.HistoryFloat32(),
		SellBestVolume: state.sellBestVolume.HistoryFloat32(),
		BestAsk:        state.bestAsk.HistoryFloat32(),
		BestBid:        state.bestBid.HistoryFloat32(),
		VolumeClusters: state.clusters[:n],
	}

	switch state.side {
	case buy:
		var ts, price, ok = state.doBuy(strategy, snap, opt)
		if ok {
			runner.stats.buyTime = append(runner.stats.buyTime, ts)
			runner.stats.buyPrice = append(runner.stats.buyPrice, price)
		}
	case sell:
		var ts, price, account, ok = state.doSell(strategy, snap, opt)
		if ok {
			runner.stats.sellTime = append(runner.stats.sellTime, ts)
			runner.stats.sellPrice = append(runner.stats.sellPrice, price)
			runner.stats.account = append(runner.stats.account, account)
		}
	}
}

func (handler *eventHandler) onCandle(ctx context.Context, candle platform.Candle) error {
	state := &handler.state

	state.price.AppendRaw(
		candle.Time,
		candle.Open,
		candle.High,
		candle.Low,
		candle.Close,
		candle.Volume,
	)
	state.tick = candle.Time / handler.opt.FramePeriod
	state.next = true

	return nil
}

func (handler *eventHandler) onTrade(ctx context.Context, trade platform.Trade) error {
	state := &handler.state

	state.tick = trade.Time / handler.opt.FramePeriod

	var (
		tSellCount  platform.Trade
		tSellVolume platform.Trade

		tBuyCount  platform.Trade
		tBuyVolume platform.Trade
	)

	if trade.IsBuyerMaker {
		// Solt by market.
		tSellVolume = platform.Trade{
			Time:     trade.Time,
			Quantity: trade.Quantity,
		}
		tSellCount = platform.Trade{
			Time:     trade.Time,
			Quantity: fixed.NewI(1, 0),
		}

		tBuyVolume = platform.Trade{
			Time: trade.Time,
		}
		tBuyCount = platform.Trade{
			Time: trade.Time,
		}
	} else {
		// Buyed by market.
		tSellVolume = platform.Trade{
			Time: trade.Time,
		}
		tSellCount = platform.Trade{
			Time: trade.Time,
		}

		tBuyVolume = platform.Trade{
			Time:     trade.Time,
			Quantity: trade.Quantity,
		}
		tBuyCount = platform.Trade{
			Time:     trade.Time,
			Quantity: fixed.NewI(1, 0),
		}
	}

	state.sellBestCount.Add(tSellCount)
	state.sellBestVolume.Add(tSellVolume)

	state.buyBestCount.Add(tBuyCount)
	state.buyBestVolume.Add(tBuyVolume)

	const step = 1.0
	price := trade.Price.Float()
	priceGroup := math.Floor(price/step) * step

	state.volumeClustersMoment[float64(priceGroup)] += float64(trade.Quantity.Float())

	state.next = state.price.Add(trade)

	return nil
}

func (handler *eventHandler) onBookTicker(ctx context.Context, bookticker platform.BookTicker) error {
	state := &handler.state

	state.tick = bookticker.Time / handler.opt.FramePeriod

	state.bestAsk.Add(platform.Trade{
		Time:     bookticker.Time,
		Price:    bookticker.BestAskPrice,
		Quantity: bookticker.BestAskQty,
	})

	state.next = state.bestBid.Add(platform.Trade{
		Time:     bookticker.Time,
		Price:    bookticker.BestBidPrice,
		Quantity: bookticker.BestBidQty,
	})

	return nil
}
