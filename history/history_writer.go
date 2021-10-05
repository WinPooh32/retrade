package history

import (
	"fmt"
	"io"
)

type HistoryWriter struct {
	w      io.Writer
	record []string
}

func NewWriter(w io.Writer) (*HistoryWriter, error) {
	return &HistoryWriter{
		w:      w,
		record: make([]string, recordLen),
	}, nil
}

func (fw *HistoryWriter) Write(t OHLCV) (err error) {
	_, err = fmt.Fprintf(fw.w, "%d,%s,%s,%s,%s,%s\n", t.Time, t.Open, t.High, t.Low, t.Close, t.Volume)
	return err
}
