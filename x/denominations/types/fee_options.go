package types

import (
	"cosmossdk.io/math"
)

const (
	DefaultMaxGasWantedPerTx  = uint64(30 * 1000 * 1000)
	DefaultHighGasTxThreshold = uint64(2.5 * 1000 * 1000)
)

var ConsensusMinFee = math.LegacyNewDecWithPrec(25, 4) // 0.0025

type MempoolFeeOptions struct {
	MaxGasWantedPerTx  uint64
	HighGasTxThreshold uint64
}
