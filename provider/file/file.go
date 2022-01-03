package file

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/WinPooh32/fixed"
	"github.com/WinPooh32/retrade/history"
	"github.com/WinPooh32/retrade/platform"
)

type File struct{ file *os.File }

func Open(name string) (f File, err error) {
	f.file, err = os.OpenFile(name, os.O_CREATE|os.O_RDONLY, 0666)
	if err != nil {
		return f, fmt.Errorf("failed to open: %w", err)
	}
	return f, nil
}

func (f File) Close() error {
	return f.file.Close()
}

func (f File) Subscribe(ctx context.Context, symbol platform.Symbol) <-chan platform.EventContainer {
	events := make(chan platform.EventContainer, 1024)

	r, err := history.NewReader(f.file)
	if err != nil {
		events <- platform.MakeError(fmt.Errorf("history new reader: %w", err))
		return nil
	}

	go func() {
		defer close(events)

		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				events <- platform.MakeError(ctx.Err())
				return
			default:
			}

			t, err := r.Read()
			if err == io.EOF {
				return
			}
			if err != nil {
				events <- platform.MakeError(fmt.Errorf("read slow candles: %w", err))
				return
			}
			c := platform.MakeCandle(
				platform.Candle{
					Time:                t.Time * 1000,
					Open:                t.Open,
					High:                t.High,
					Low:                 t.Low,
					Close:               t.Close,
					Volume:              t.Volume,
					TimeClose:           0,
					VolumeQuote:         fixed.ZERO,
					CountTrades:         0,
					VolumeTakerBuyBase:  fixed.ZERO,
					VolumeTakerBuyQuote: fixed.ZERO,
				},
			)
			events <- c
		}
	}()

	return events
}
