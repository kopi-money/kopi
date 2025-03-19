package types

import (
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AmountsMapEntry struct {
	denom  string
	amount math.LegacyDec
}
type AmountsMap struct {
	cm []AmountsMapEntry
}

func NewAmountsMap() *AmountsMap {
	return &AmountsMap{}
}

func NewAmountsMapFromCoins(coins sdk.Coins) *AmountsMap {
	coinMap := AmountsMap{}
	for _, coin := range coins {
		coinMap.Add(coin.Denom, coin.Amount.ToLegacyDec())
	}

	return &coinMap
}

func (am *AmountsMap) AmountOf(denom string) math.LegacyDec {
	for _, entry := range am.cm {
		if entry.denom == denom {
			return entry.amount
		}
	}

	return math.LegacyZeroDec()
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

func (am *AmountsMap) Coins() (coins sdk.Coins) {
	for _, entry := range am.cm {
		coins = coins.Add(sdk.NewCoin(entry.denom, entry.amount.TruncateInt()))
	}

	return
}
