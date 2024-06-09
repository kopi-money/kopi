package data

import _ "embed"

//go:embed liq.dat
var LiquidityEntries string

//go:embed orders.dat
var Orders string
