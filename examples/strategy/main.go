package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/WinPooh32/fta"
	"github.com/WinPooh32/retrade/backtest"
	"github.com/WinPooh32/retrade/candle"
	"github.com/WinPooh32/retrade/platform"
	"github.com/WinPooh32/retrade/provider/binance"
	"github.com/WinPooh32/series"
)

type Options struct {
	PeriodFast int
	PeriodSlow int
}

type MacdStrategy struct {
	OptBuy  Options
	OptSell Options
	Signal  int
}

func (ms *MacdStrategy) Name() string {
	return "Cross Moving Averages"
}

func (ms *MacdStrategy) calc(price candle.HistoryFloat32, opt Options) (val, sig float32) {
	var (
		close = series.MakeData(1, price.Time, price.Close)
		n     = close.Len()
	)

	if n < ms.OptBuy.PeriodSlow {
		return
	}

	var macd, macdSignal = fta.MACD(close, float32(opt.PeriodFast), float32(opt.PeriodSlow), float32(ms.Signal), true)

	val = macd.Data()[n-1]
	sig = macdSignal.Data()[n-1]
	return
}

func (ms *MacdStrategy) BuySignal(snap backtest.HistorySnaphsot) bool {
	var val, sig = ms.calc(snap.Price, ms.OptBuy)
	return val > sig
}

func (ms *MacdStrategy) SellSignal(snap backtest.HistorySnaphsot) bool {
	var val, sig = ms.calc(snap.Price, ms.OptSell)
	return val < sig
}

type Provider struct {
	backtest.NopProvider
	platform.Public
}

func (prov *Provider) Subscribe(ctx context.Context, symbol platform.Symbol) <-chan platform.EventContainer {
	return prov.Public.Subscribe(ctx, symbol)
}

func main() {
	const intervalTicks = 1
	const intervalLetter = binance.IntervalDay
	const symbol = "BTCUSDT"
	const window = 1000

	var ctx, cancel = signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var interval = binance.IntervalFromLetter(intervalTicks, intervalLetter)

	var publicProvider = binance.NewHistory(false, intervalTicks, intervalLetter)

	var runner = backtest.NewRunner(&Provider{Public: publicProvider})

	var optBuy = Options{
		PeriodFast: 12,
		PeriodSlow: 26,
	}

	var optSell = Options{
		PeriodFast: 8,
		PeriodSlow: 17,
	}

	var opt = backtest.Options{
		Symbol:            symbol,
		FeeBuy:            0.001,
		FeeSell:           0.001,
		Account:           1000.0,
		FramePeriod:       interval,
		HistoryWindowSize: window,
		Limit:             1000.0,
	}

	result, err := runner.Run(ctx, &MacdStrategy{optBuy, optSell, 9}, opt)
	if err != nil {
		fmt.Printf("runner: method Test: %s\n", err)
		return
	}

	fmt.Println("wallet USDT:")
	for _, a := range result.Account.Data() {
		fmt.Println(a)
	}

	fmt.Println("pool value:", result.Pool)
	fmt.Println("exit.")
}
