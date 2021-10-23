package binance

import "fmt"

func IntervalFromLetter(period int, letter IntervalLetter) (interval int64) {
	const (
		second = 1000
		minute = 60 * second
		hour   = 60 * minute
		day    = 24 * hour
	)
	var scale int64
	switch letter {
	case "s":
		scale = second
	case "m":
		scale = minute
	case "h":
		scale = hour
	case "d":
		scale = day
	default:
		panic(fmt.Sprintf("unexpected interval time letter: %s", letter))
	}
	return int64(period) * scale
}
