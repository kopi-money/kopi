package types

import "cosmossdk.io/math"

func (lp LiquidityPair) GetActualVirtualBase() math.LegacyDec {
	return lp.ActualBase.Add(lp.VirtualBase)
}

func (lp LiquidityPair) GetActualVirtualOther() math.LegacyDec {
	return lp.ActualOther.Add(lp.VirtualOther)
}

func (lp LiquidityPair) GetFullBase() math.LegacyDec {
	return lp.ActualBase.Add(lp.VirtualBase).Add(lp.ExtraBase)
}

func (lp LiquidityPair) GetFullOther() math.LegacyDec {
	return lp.ActualOther.Add(lp.VirtualOther).Add(lp.ExtraOther)
}
