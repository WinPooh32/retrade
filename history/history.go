package history

import (
	"github.com/WinPooh32/fixed"
)

const recordLen = 6

type OHLCV struct {
	Time int64
	Open,
	High,
	Low,
	Close,
	Volume fixed.Fixed
}

type Writer interface {
	Write(t OHLCV) (err error)
}

type Reader interface {
	Read() (t OHLCV, err error)
}
