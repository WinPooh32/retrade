package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"retrade/platform"

	"github.com/WinPooh32/fixed"
)

type IntervalLetter string

const (
	IntervalSecond = "s"
	IntervalMinute = "m"
	IntervalHour   = "h"
	IntervalDay    = "d"
)

type BinanceHistory struct {
	testnet bool
	inteval int
	letter  IntervalLetter
}

func NewHistory(testnet bool, inteval int, letter IntervalLetter) *BinanceHistory {
	return &BinanceHistory{
		testnet: testnet,
		inteval: inteval,
		letter:  letter,
	}
}

func (bh *BinanceHistory) Subscribe(ctx context.Context, symbol string) <-chan platform.EventContainer {
	const (
		Time                = 0
		Open                = 1
		High                = 2
		Low                 = 3
		Close               = 4
		Volume              = 5
		TimeClose           = 6
		VolumeQuote         = 7
		CountTrades         = 8
		VolumeTakerBuyBase  = 9
		VolumeTakerBuyQuote = 10
	)

	events := make(chan platform.EventContainer, 32)

	go func() {
		defer close(events)

		resp, err := http.Get(fmt.Sprintf("%s/v3/klines?symbol=%s&interval=%d%s&limit=10000", bh.api(), symbol, bh.inteval, bh.letter))
		if err != nil {
			events <- platform.MakeError(fmt.Errorf("failed to prefetch: %w", err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			events <- platform.MakeError(fmt.Errorf("unexpected status: %d", resp.StatusCode))
			return
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			events <- platform.MakeError(fmt.Errorf("read resp body: %w", err))
			return
		}

		type ohlcv [][]interface{}
		var vals ohlcv

		err = json.Unmarshal(data, &vals)
		if err != nil {
			events <- platform.MakeError(fmt.Errorf("json unmarshal: %w", err))
			return
		}

		for _, row := range vals {
			select {
			case <-ctx.Done():
				events <- platform.MakeError(ctx.Err())
				return
			default:
			}

			tsF64, _ := row[Time].(float64)
			openStr, _ := row[Open].(string)
			highStr, _ := row[High].(string)
			lowStr, _ := row[Low].(string)
			closeStr, _ := row[Close].(string)
			volumeStr, _ := row[Volume].(string)
			tsCloseF64, _ := row[TimeClose].(float64)
			volumeQuoteStr, _ := row[VolumeQuote].(string)
			countTradesF64, _ := row[CountTrades].(float64)
			volumeTakerBuyBaseStr, _ := row[VolumeTakerBuyBase].(string)
			volumeTakerBuyQuoteStr, _ := row[VolumeTakerBuyQuote].(string)

			c := platform.Candle{
				Time:                int64(tsF64),
				Open:                fixed.NewS(openStr),
				High:                fixed.NewS(highStr),
				Low:                 fixed.NewS(lowStr),
				Close:               fixed.NewS(closeStr),
				Volume:              fixed.NewS(volumeStr),
				TimeClose:           int64(tsCloseF64),
				VolumeQuote:         fixed.NewS(volumeQuoteStr),
				CountTrades:         int64(countTradesF64),
				VolumeTakerBuyBase:  fixed.NewS(volumeTakerBuyBaseStr),
				VolumeTakerBuyQuote: fixed.NewS(volumeTakerBuyQuoteStr),
			}

			events <- platform.MakeCandle(c)
		}
	}()

	return events
}

func (bh *BinanceHistory) api() string {
	if bh.testnet {
		return "https://testnet.binance.vision/api"
	} else {
		return "https://api.binance.com/api"
	}
}
