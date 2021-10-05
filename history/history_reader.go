package history

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"github.com/WinPooh32/fixed"
	"github.com/hashicorp/go-multierror"
)

type HistoryReader struct {
	r     *csv.Reader
	count int
}

func NewReader(r io.Reader) (*HistoryReader, error) {
	rcsv := csv.NewReader(r)
	rcsv.Comma = ','
	rcsv.ReuseRecord = true

	return &HistoryReader{
		r: rcsv,
	}, nil
}

func (hr *HistoryReader) Read() (t OHLCV, err error) {
	const (
		Time   = 0
		Open   = 1
		High   = 2
		Low    = 3
		Close  = 4
		Volume = 5
	)

	hr.count++

	record, err := hr.r.Read()
	if err == io.EOF {
		return t, err
	}
	if err != nil {
		return t, fmt.Errorf("read csv record: %w", err)
	}
	if len(record) < recordLen {
		return t, fmt.Errorf("record on line %d: wrong number of fields %d, expected not less than %d", hr.count, len(record), recordLen)
	}

	var merr *multierror.Error

	t.Time, err = strconv.ParseInt(record[Time], 10, 64)
	merr = multierror.Append(merr, err)

	t.Open, err = fixed.NewSErr(record[Open])
	merr = multierror.Append(merr, err)

	t.High, err = fixed.NewSErr(record[High])
	merr = multierror.Append(merr, err)

	t.Low, err = fixed.NewSErr(record[Low])
	merr = multierror.Append(merr, err)

	t.Close, err = fixed.NewSErr(record[Close])
	merr = multierror.Append(merr, err)

	t.Volume, err = fixed.NewSErr(record[Volume])
	merr = multierror.Append(merr, err)

	return t, merr.ErrorOrNil()
}
