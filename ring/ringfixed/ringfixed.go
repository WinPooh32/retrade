package ringfixed

import (
	_ "github.com/WinPooh32/gotemplate/ring"

	fixedlib "github.com/WinPooh32/fixed"
)

type fixed = fixedlib.Fixed

//go:generate gotemplate "github.com/WinPooh32/gotemplate/ring" Ring(fixed)
