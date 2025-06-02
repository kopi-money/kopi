package types

import (
	"cosmossdk.io/math"
)

var (
	DefaultMaxGasWantedPerTx  = uint64(30 * 1000 * 1000)
	DefaultHighGasTxThreshold = uint64(2.5 * 1000 * 1000)
	ConsensusMinFee           = math.LegacyNewDecWithPrec(25, 4) // 0.0025 XKP
)

type MempoolFeeOptions struct {
	MaxGasWantedPerTx  uint64
	HighGasTxThreshold uint64
}

func NewDefaultMempoolFeeOptions() MempoolFeeOptions {
	return MempoolFeeOptions{
		MaxGasWantedPerTx:  DefaultMaxGasWantedPerTx,
		HighGasTxThreshold: DefaultHighGasTxThreshold,
	}
}
