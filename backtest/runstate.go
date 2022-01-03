package backtest

import (
	"github.com/WinPooh32/fixed"
	"github.com/WinPooh32/retrade/candle"
	"github.com/WinPooh32/retrade/ring/ringmapf64"
)

type runstate struct {
	side         int
	next         bool
	finishedTick int64
	tick         int64

	account float32
	pool    float32

	price *candle.Candle

	buyBestCount  *candle.Candle
	sellBestCount *candle.Candle

	buyBestVolume  *candle.Candle
	sellBestVolume *candle.Candle

	bestAsk *candle.Candle
	bestBid *candle.Candle

	volumeClusters       ringmapf64.Ring
	volumeClustersMoment map[float64]float64

	clusters []map[float64]float64
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
