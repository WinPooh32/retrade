package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/WinPooh32/retrade/candle"
	"github.com/WinPooh32/retrade/platform"
	"github.com/WinPooh32/retrade/provider/binance"
	"github.com/WinPooh32/retrade/provider/file"
)

const intervalTicks = 30
const intervalLetter = binance.IntervalMinute

func main() {
	offline := flag.Bool("offline", false, "offline mode")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	const symbol = "BTCUSDT"
	const window = 2 * 60
	const buffer = 60000

	var interval = binance.IntervalFromLetter(intervalTicks, intervalLetter)

	candles := candle.NewCandle(interval, buffer)

	err := fetch(ctx, symbol, window, interval, candles, *offline)
	if errors.Is(err, context.Canceled) {
		fmt.Println("interrupted.")
		return
	}
	if err != nil {
		fmt.Println("fetch:", err)
		return
	}

	fmt.Println("exit.")
}

func fetch(ctx context.Context, symbol string, window int, interval int64, candles *candle.Candle, offline bool) error {
	err := fetchHistoryFile(ctx, symbol, candles)
	if err != nil {
		return fmt.Errorf("fetch history file: %w", err)
	}

	err = fetchHistoryBinance(ctx, symbol, window, interval, candles)
	if err != nil {
		return fmt.Errorf("fetch history binance: %w", err)
	}

	if !offline {
		err = fetchBinance(ctx, symbol, candles)
		if err != nil {
			return fmt.Errorf("fetch binance trades: %w", err)
		}
	}

	return nil
}

func fetchHistoryFile(ctx context.Context, symbol string, candles *candle.Candle) error {
	fmt.Println("fetch file history.")

	f, err := file.Open(symbol + ".csv")
	if err != nil {
		return fmt.Errorf("open history file: %w", err)
	}

	historyEvents := f.Subscribe(ctx, symbol)

	for e := range historyEvents {
		switch e.Type {
		case platform.EventErr:
			return fmt.Errorf("event: %w", e.Error)
		case platform.EventCandle:
			c := e.Event.Candle
			candles.AppendRaw(
				c.Time,
				c.Open,
				c.High,
				c.Low,
				c.Close,
				c.Volume,
			)
		}
	}
	return nil
}

func fetchHistoryBinance(ctx context.Context, symbol string, window int, interval int64, candles *candle.Candle) error {
	fmt.Println("fetch binance history.")

	binanceHistory := binance.NewHistory(false, intervalTicks, intervalLetter)
	binanceHistoryEvents := binanceHistory.Subscribe(ctx, symbol)

	fileLastTs, _, _, _, _, _ := candles.Last()

	for e := range binanceHistoryEvents {
		switch e.Type {
		case platform.EventErr:
			return fmt.Errorf("event: %w", e.Error)
		case platform.EventCandle:
			c := e.Event.Candle
			if c.Time <= fileLastTs {
				continue
			}
			candles.AppendRaw(
				c.Time,
				c.Open,
				c.High,
				c.Low,
				c.Close,
				c.Volume,
			)
			time, open, high, low, close, volume := candles.Last()
			fmt.Println("binance candle:", time, open, high, low, close, volume)
		}
	}

	return nil
}

func fetchBinance(ctx context.Context, symbol string, candles *candle.Candle) error {
	fmt.Println("fetch binance trades.")

reconnect:
	for i := 0; i < 5; i++ {
		fmt.Println("fetch binance.")

		binance, err := binance.New(false, "", "")
		if err != nil {
			return fmt.Errorf("binance: new instance: %w", err)
		}

		binanceEvents := binance.Subscribe(ctx, symbol)

		for e := range binanceEvents {
			switch e.Type {
			case platform.EventErr:
				if e.Error == context.Canceled {
					return fmt.Errorf("event: %w", e.Error)
				} else {
					fmt.Println(e.Error)
					s := 5 * time.Second
					fmt.Printf("reconnect in %s\n", s)
					time.Sleep(s)
					continue reconnect
				}
			case platform.EventTrade:
				t := e.Event.Trade

				fmt.Printf("%+v\n", t)

				if candles.Add(t) {
					// Print filled candlestick.
					time, open, high, low, close, volume := candles.Last()
					fmt.Println("last filled candle:", time, open, high, low, close, volume)
				}
			case platform.EventBookTicker:
				b := e.Event.BookTicker

				fmt.Printf("%+v\n", b)
			}
			// Reset retry counter.
			if i != 0 {
				i = 0
			}
		}
	}
	return nil
}
