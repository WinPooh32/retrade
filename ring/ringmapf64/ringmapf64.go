package ringmapf64

import (
	_ "github.com/WinPooh32/gotemplate/ring"
)

type mapf64 = map[float64]float64

//go:generate gotemplate "github.com/WinPooh32/gotemplate/ring" Ring(mapf64)
