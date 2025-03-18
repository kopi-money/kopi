package types

import (
	"cosmossdk.io/math"
	"sort"
)

type EpochPayouts struct {
	PreviousEpoch EpochLeftovers
	CurrentEpoch  EpochLeftovers
	Usage         EpochLeftovers
}

func (ep EpochPayouts) ToLeftovers() EpochLeftovers {
	cm := NewCoinMap(nil)
	ep.PreviousEpoch.AddToCoinMap(cm)
	ep.CurrentEpoch.AddToCoinMap(cm)
	ep.Usage.SubFromCoinMap(cm)

	var leftovers []EpochLeftover
	for _, entry := range cm.cm {
		if entry.amount.IsPositive() {
			leftovers = append(leftovers, EpochLeftover{
				Denom:  entry.denom,
				Amount: entry.amount,
			})
		}
	}

	sort.SliceStable(leftovers, func(i, j int) bool {
		return leftovers[i].Denom < leftovers[j].Denom
	})

	return EpochLeftovers{
		Leftovers: leftovers,
	}
}

func (ep EpochPayouts) Get(denom string) math.LegacyDec {
	v1 := ep.CurrentEpoch.Get(denom)
	v2 := ep.PreviousEpoch.Get(denom)
	return v1.Add(v2)
}

func (ep EpochPayouts) GetTruncated(denom string) math.LegacyDec {
	return ep.Get(denom).TruncateDec()
}

func (ep EpochPayouts) Denoms() (list []string) {
	ep.CurrentEpoch.denoms(&list)
	ep.PreviousEpoch.denoms(&list)
	return
}

func (ep *EpochPayouts) AddUsage(denom string, amount math.LegacyDec) {
	ep.Usage = ep.Usage.Add(denom, amount)
}

func (el EpochLeftovers) Add(denom string, amount math.LegacyDec) EpochLeftovers {
	for index, po := range el.Leftovers {
		if po.Denom == denom {
			po.Amount = po.Amount.Add(amount)
			el.Leftovers[index] = po
			return el
		}
	}

	el.Leftovers = append(el.Leftovers, EpochLeftover{
		Denom:  denom,
		Amount: amount,
	})

	return el
}

func (el EpochLeftovers) AddToCoinMap(cm *CoinMap) {
	el.toCoinMap(cm, true)
}

func (el EpochLeftovers) SubFromCoinMap(cm *CoinMap) {
	el.toCoinMap(cm, false)
}

func (el EpochLeftovers) toCoinMap(cm *CoinMap, add bool) {
	for _, po := range el.Leftovers {
		if add {
			cm.Add(po.Denom, po.Amount)
		} else {
			cm.Sub(po.Denom, po.Amount)
		}
	}
}

func (el EpochLeftovers) denoms(list *[]string) {
	for _, po := range el.Leftovers {
		seen := false
		for _, denom := range *list {
			if po.Denom == denom {
				seen = true
				break
			}
		}

		if !seen {
			*list = append(*list, po.Denom)
		}
	}

	return
}

func (el EpochLeftovers) Get(denom string) math.LegacyDec {
	for _, po := range el.Leftovers {
		if po.Denom == denom {
			return po.Amount
		}
	}

	return math.LegacyZeroDec()
}

type merge func(math.LegacyDec, math.LegacyDec) math.LegacyDec

func add(v1, v2 math.LegacyDec) math.LegacyDec {
	return v1.Add(v2)
}

func sub(v1, v2 math.LegacyDec) math.LegacyDec {
	return v1.Sub(v2)
}
