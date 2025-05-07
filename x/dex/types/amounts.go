package types

import (
	"cosmossdk.io/math"
	"fmt"
)

type AmountsMapEntry struct {
	denom  string
	amount math.LegacyDec
}
type AmountsMap struct {
	cm []AmountsMapEntry
}

func (am AmountsMap) AmountOf(denom string) math.LegacyDec {
	for _, entry := range am.cm {
		if entry.denom == denom {
			return entry.amount
		}
	}

	return math.LegacyZeroDec()
}

func (am *AmountsMap) Add(denom string, addAmount math.LegacyDec) {
	for index, entry := range am.cm {
		if entry.denom == denom {
			entry.amount = entry.amount.Add(addAmount)
			am.cm[index] = entry
			return
		}
	}

	am.cm = append(am.cm, AmountsMapEntry{
		denom:  denom,
		amount: addAmount,
	})
}

func (am *AmountsMap) Sub(denom string, subAmount math.LegacyDec) {
	am.sub(denom, subAmount, false)
}

func (am *AmountsMap) SubIgnore(denom string, subAmount math.LegacyDec) {
	am.sub(denom, subAmount, true)
}

func (am *AmountsMap) sub(denom string, subAmount math.LegacyDec, ignoreNegative bool) {
	for index, entry := range am.cm {
		if entry.denom == denom {
			entry.amount = entry.amount.Sub(subAmount)
			if !ignoreNegative && entry.amount.IsNegative() {
				panic(fmt.Sprintf("negative coin amount for %v", denom))
			}

			if entry.amount.IsPositive() {
				am.cm[index] = entry
			} else {
				am.cm = append(am.cm[:index], am.cm[index+1:]...)
			}

			return
		}
	}

	panic(fmt.Sprintf("cannot sub denom that does not exist (%v)", denom))
}
