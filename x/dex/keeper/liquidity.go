package keeper

import (
	"context"
	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/cache"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/pkg/errors"
	"sort"
	"strconv"
)

// SetLiquidity sets a specific liquidity in the store from its index. When the index is zero, i.e. it's a new entry,
// the NextIndex is increased and updated as well.
func (k Keeper) SetLiquidity(ctx context.Context, liquidity types.Liquidity, change math.Int) types.Liquidity {
	if liquidity.Index == 0 {
		nextIndex, _ := k.liquidityEntriesNextIndex.Get(ctx)
		nextIndex++
		liquidity.Index = nextIndex

		k.SetLiquidityEntryNextIndex(ctx, nextIndex)
	}

	k.liquidityEntries.Set(ctx, collections.Join(liquidity.Denom, liquidity.Index), liquidity)
	k.updateLiquiditySum(ctx, liquidity.Denom, change)
	return liquidity
}

func (k Keeper) GetLiquidityEntryNextIndex(ctx context.Context) (uint64, bool) {
	return k.liquidityEntriesNextIndex.Get(ctx)
}

func (k Keeper) SetLiquidityEntryNextIndex(ctx context.Context, nextIndex uint64) {
	k.liquidityEntriesNextIndex.Set(ctx, nextIndex)
}

func (k Keeper) updateLiquiditySum(ctx context.Context, denom string, change math.Int) {
	liqSum := k.GetLiquiditySum(ctx, denom).Add(change)
	k.SetLiquiditySum(ctx, types.LiquiditySum{Denom: denom, Amount: liqSum})
}

// AddLiquidity adds liquidity to the dex for a given amount and address. The address is used to keep track which user
// has added how much.
func (k Keeper) AddLiquidity(ctx context.Context, eventManager sdk.EventManagerI, address sdk.AccAddress, denom string, amount math.Int) error {
	if !k.DenomKeeper.IsValidDenom(ctx, denom) {
		return types.ErrDenomNotFound
	}

	coins := sdk.NewCoins(sdk.NewCoin(denom, amount))
	if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, address, types.PoolLiquidity, coins); err != nil {
		return errors.Wrap(err, "could not send coins to module")
	}

	// The dex works by routing all trades via XKP. The chain is initialized with funds for the reserve, which adds
	// those funds to the dex. When no liquidity for XKP has been added, we refuse new liquidity as long as no
	// liquidity for XKP is added.
	liqBase := k.GetLiquiditySum(ctx, utils.BaseCurrency)
	if liqBase.Equal(math.ZeroInt()) && denom != utils.BaseCurrency {
		return types.ErrBaseLiqEmpty
	}

	_, liq := k.addLiquidity(ctx, denom, address.String(), amount, nil)

	// When changing actual liquidity, the virtual liquidity has to be adjusted to keep the ratio.
	if denom != utils.BaseCurrency {
		k.updatePair(ctx, nil, denom)
	} else {
		k.updatePairs(ctx, nil)
	}

	eventManager.EmitEvent(
		sdk.NewEvent(
			"liquidity_added",
			sdk.Attribute{Key: "denom", Value: denom},
			sdk.Attribute{Key: "amount", Value: amount.String()},
			sdk.Attribute{Key: "address", Value: address.String()},
			sdk.Attribute{Key: "index", Value: strconv.Itoa(int(liq.Index))},
		),
	)

	return nil
}

func (k Keeper) addLiquidity(ctx context.Context, denom, address string, amount math.Int, liquidityEntries []types.Liquidity) ([]types.Liquidity, types.Liquidity) {
	if liquidityEntries == nil {
		liquidityEntries = k.LiquidityIterator(ctx, denom).GetAll()
	}

	seen := false
	for index, liq := range liquidityEntries {
		if liq.Address == address {
			// if liquidity would be added to the first found occurrence, liquidity added by whales would be used more
			// often compared to smaller liquidity entries. To make this more fair, liquidity is added to the second
			// entry of an address or in a new entry at the end
			if !seen {
				seen = true
				continue
			}

			liq.Amount = liq.Amount.Add(amount)
			k.SetLiquidity(ctx, liq, amount)
			liquidityEntries[index] = liq
			return liquidityEntries, liq
		}
	}

	liq := types.Liquidity{Denom: denom, Address: address, Amount: amount}
	liq = k.SetLiquidity(ctx, liq, amount)
	liquidityEntries = append(liquidityEntries, liq)

	return liquidityEntries, liq
}

func (k Keeper) LiquidityIterator(ctx context.Context, denom string) *cache.Iterator[collections.Pair[string, uint64], types.Liquidity] {
	extraFilters := []cache.Filter[collections.Pair[string, uint64]]{
		func(key collections.Pair[string, uint64]) bool {
			return key.K1() == denom
		},
	}
	return k.liquidityEntries.Iterator(ctx, extraFilters...)
}

func (k Keeper) GetLiquidityByAddress(ctx context.Context, denom, address string) math.Int {
	sum := math.ZeroInt()

	iterator := k.LiquidityIterator(ctx, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()

		if liq.Address == address {
			sum = sum.Add(liq.Amount)
		}
	}

	return sum
}

func (k Keeper) GetLiquidityEntriesByAddress(ctx context.Context, denom, address string) int {
	num := 0

	iterator := k.LiquidityIterator(ctx, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()
		if liq.Address == address {
			num++
		}
	}

	return num
}

func (k Keeper) GetAllLiquidity(ctx context.Context) (list []types.Liquidity) {
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		iterator := k.LiquidityIterator(ctx, denom)
		for iterator.Valid() {
			list = append(list, iterator.GetNext())
		}
	}

	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Index < list[j].Index
	})

	return
}

// RemoveLiquidity removes a liquidity from the store
func (k Keeper) RemoveLiquidity(ctx context.Context, denom string, index uint64, change math.Int) {
	if change.LT(math.ZeroInt()) {
		panic("cannot be negative amount, positive amount is assumed")
	}

	key := collections.Join(denom, index)
	k.liquidityEntries.Remove(ctx, key)
	k.updateLiquiditySum(ctx, denom, change.Neg())
}

// UpdateVirtualLiquidities updates the virtual liquidity for each pair. This method is called at the end of each block.
// The virtual liquidty is only updated when there is no actual liquidity for that denom. When the virtual liquidity is
// 0, it means the pair probably just have been created and will be set to the initial virtual amount. If the amount
// of actual liquidity is zero and the amount of virtual liquidity is not zero, we slowly decrease the amount of virtual
// liquidity to increase that denom's price.
func (k Keeper) UpdateVirtualLiquidities(ctx context.Context) {
	decay := k.GetParams(ctx).VirtualLiquidityDecay
	liqBase := k.GetLiquiditySum(ctx, utils.BaseCurrency)

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		if denom != utils.BaseCurrency {
			liq := k.GetLiquiditySum(ctx, denom)
			if liq.LT(k.DenomKeeper.MinLiquidity(ctx, denom)) {
				pair, found := k.GetLiquidityPair(ctx, denom)

				if !found {
					pair.Denom = denom
					pair.VirtualBase = math.LegacyZeroDec()
					pair.VirtualOther = math.LegacyZeroDec()
				}

				if pair.VirtualOther.Equal(math.LegacyZeroDec()) {
					factor := k.DenomKeeper.InitialVirtualLiquidityFactor(ctx, denom)
					pair.VirtualOther = liqBase.ToLegacyDec().Quo(factor)
				} else {
					pair.VirtualOther = pair.VirtualOther.Mul(decay)
				}

				k.SetLiquidityPair(ctx, pair)
				k.updateRatios(ctx, pair.Denom)
			}
		}
	}
}

func (k Keeper) GetDenomValue(ctx context.Context, denom string) (math.LegacyDec, error) {
	if denom == utils.BaseCurrency {
		liq := k.GetLiquiditySum(ctx, utils.BaseCurrency)
		return liq.ToLegacyDec(), nil
	}

	liq := k.GetFullLiquidityOther(ctx, denom)
	price, err := k.CalculatePrice(ctx, denom, utils.BaseCurrency)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return liq.Mul(price), nil
}

func compareLiquidity(l1, l2 types.Liquidity) bool {
	return l1.Denom == l2.Denom &&
		l1.Index == l2.Index &&
		l1.Amount.Equal(l2.Amount) &&
		l1.Address == l2.Address
}
