package candle

import (
	"retrade/platform"
	"retrade/ring/ringfixed"
	"retrade/ring/ringi64"

	"github.com/WinPooh32/fixed"
)

type Fixed = fixed.Fixed

type OHLCV struct {
	Date   ringi64.Ring
	Open   ringfixed.Ring
	High   ringfixed.Ring
	Low    ringfixed.Ring
	Close  ringfixed.Ring
	Volume ringfixed.Ring
}

func NewOHLCV(cap int) *OHLCV {
	return &OHLCV{
		Date:   ringi64.MakeRing(cap),
		Open:   ringfixed.MakeRing(cap),
		High:   ringfixed.MakeRing(cap),
		Low:    ringfixed.MakeRing(cap),
		Close:  ringfixed.MakeRing(cap),
		Volume: ringfixed.MakeRing(cap),
	}
}

type History struct {
	Time   []int64
	Open   []Fixed
	High   []Fixed
	Low    []Fixed
	Close  []Fixed
	Volume []Fixed
}

func (h History) Slice(l, r int) History {
	return History{
		Time:   h.Time[l:r],
		Open:   h.Open[l:r],
		High:   h.High[l:r],
		Low:    h.Low[l:r],
		Close:  h.Close[l:r],
		Volume: h.Volume[l:r],
	}
}

func MakeHistory(cap int) History {
	return History{
		Time:   make([]int64, cap),
		Open:   make([]Fixed, cap),
		High:   make([]Fixed, cap),
		Low:    make([]Fixed, cap),
		Close:  make([]Fixed, cap),
		Volume: make([]Fixed, cap),
	}
}

type HistoryFloat32 struct {
	Time   []int64
	Open   []float32
	High   []float32
	Low    []float32
	Close  []float32
	Volume []float32
}

func (h HistoryFloat32) Slice(l, r int) HistoryFloat32 {
	return HistoryFloat32{
		Time:   h.Time[l:r],
		Open:   h.Open[l:r],
		High:   h.High[l:r],
		Low:    h.Low[l:r],
		Close:  h.Close[l:r],
		Volume: h.Volume[l:r],
	}
}

func MakeHistoryFloat32(cap int) HistoryFloat32 {
	return HistoryFloat32{
		Time:   make([]int64, cap),
		Open:   make([]float32, cap),
		High:   make([]float32, cap),
		Low:    make([]float32, cap),
		Close:  make([]float32, cap),
		Volume: make([]float32, cap),
	}
}

type Candle struct {
	count       int64
	period      int64
	ts          int64
	countPeriod int64
	records     []platform.Trade
	ohlcv       *OHLCV
	history     History
	historyF32  HistoryFloat32
}

func NewCandle(period int64, cap int) *Candle {
	return &Candle{
		period:     period,
		ts:         0,
		records:    make([]platform.Trade, 0, 1000),
		ohlcv:      NewOHLCV(cap),
		history:    MakeHistory(cap),
		historyF32: MakeHistoryFloat32(cap),
	}
}

func (c *Candle) AppendRaw(time int64, open, high, low, close, volume Fixed) {
	c.countPeriod = (time / c.period)
	c.ts = c.countPeriod * c.period
	c.ohlcv.Date.ForcePushBack(time)
	c.ohlcv.Open.ForcePushBack(open)
	c.ohlcv.High.ForcePushBack(high)
	c.ohlcv.Low.ForcePushBack(low)
	c.ohlcv.Close.ForcePushBack(close)
	c.ohlcv.Volume.ForcePushBack(volume)
	c.count++
}

func (c *Candle) Add(t platform.Trade) (filled bool) {
	c.records = append(c.records, t)

	countPeriod := t.Time / c.period

	if countPeriod > c.countPeriod {
		c.flush()
		c.ts = countPeriod * c.period
		c.countPeriod = countPeriod
		return true
	}
	return false
}

func (c *Candle) History() History {
	n := c.ohlcv.Date.CopyTo(c.history.Time)
	c.ohlcv.Open.CopyTo(c.history.Open)
	c.ohlcv.High.CopyTo(c.history.High)
	c.ohlcv.Low.CopyTo(c.history.Low)
	c.ohlcv.Close.CopyTo(c.history.Close)
	c.ohlcv.Volume.CopyTo(c.history.Volume)
	return c.history.Slice(0, n)
}

func (c *Candle) HistoryFloat32() HistoryFloat32 {
	h := c.History()
	n := len(h.Time)

	copy(c.historyF32.Time[:n], h.Time)
	c.fixedToFloat32(c.historyF32.Open[:n], h.Open)
	c.fixedToFloat32(c.historyF32.High[:n], h.High)
	c.fixedToFloat32(c.historyF32.Low[:n], h.Low)
	c.fixedToFloat32(c.historyF32.Close[:n], h.Close)
	c.fixedToFloat32(c.historyF32.Volume[:n], h.Volume)

	return c.historyF32.Slice(0, n)
}

func (c *Candle) Last() (time int64, open, high, low, close, volume Fixed) {
	if c.ohlcv.Date.Len() == 0 {
		return
	}
	time = c.ohlcv.Date.Back()
	open = c.ohlcv.Open.Back()
	high = c.ohlcv.High.Back()
	low = c.ohlcv.Low.Back()
	close = c.ohlcv.Close.Back()
	volume = c.ohlcv.Volume.Back()
	return
}

func (c *Candle) Count() int64 {
	return c.count
}

func (c *Candle) BufLen() int {
	return c.ohlcv.Date.Len()
}

func (c *Candle) lastPartial() (date int64, open, high, low, close, volume Fixed) {
	if len(c.records) == 0 {
		return
	}

	date = c.ts
	open = c.records[0].Price
	high = open
	low = open
	close = c.records[len(c.records)-1].Price
	volume = fixed.ZERO

	if date == 0 {
		date = c.records[0].Time
	}

	for _, r := range c.records {
		if r.Price.GreaterThan(high) {
			high = r.Price
		} else if r.Price.LessThan(low) {
			low = r.Price
		}
		volume = volume.Add(r.Quantity)
	}

	return
}

func (c *Candle) flush() error {
	var date, open, high, low, close, volume = c.lastPartial()

	c.ohlcv.Date.ForcePushBack(date)
	c.ohlcv.Open.ForcePushBack(open)
	c.ohlcv.High.ForcePushBack(high)
	c.ohlcv.Low.ForcePushBack(low)
	c.ohlcv.Close.ForcePushBack(close)
	c.ohlcv.Volume.ForcePushBack(volume)

	c.records = c.records[:0]
	c.count++

	return nil
}

func (c *Candle) fixedToFloat32(dst []float32, src []fixed.Fixed) {
	if len(dst) != len(src) {
		panic("dst and src len must be equal")
	}
	for i, f := range src {
		dst[i] = float32(f.Float())
	}
}
