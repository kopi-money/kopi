package keeper

import (
	"context"
	"fmt"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"sort"
	"strconv"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/cache"
	"github.com/kopi-money/kopi/constants"
	"github.com/kopi-money/kopi/x/dex/types"
)

// SetLiquidity sets a specific liquidity in the store from its index. When the index is zero, i.e. it's a new entry,
// the NextIndex is increased and updated as well.
func (k Keeper) SetLiquidity(ctx context.Context, denom string, liquidity types.Liquidity) types.Liquidity {
	if liquidity.Index == 0 {
		nextIndex, _ := k.liquidityEntriesNextIndex.Get(ctx)
		nextIndex++
		liquidity.Index = nextIndex

		k.SetLiquidityEntryNextIndex(ctx, nextIndex)
	}

	k.liquidityEntries.Set(ctx, denom, liquidity.Index, liquidity)
	return liquidity
}

func (k Keeper) GetLiquidityEntryNextIndex(ctx context.Context) (uint64, bool) {
	return k.liquidityEntriesNextIndex.Get(ctx)
}

func (k Keeper) SetLiquidityEntryNextIndex(ctx context.Context, nextIndex uint64) {
	k.liquidityEntriesNextIndex.Set(ctx, nextIndex)
}

// AddLiquidity adds liquidity to the dex for a given amount and address. The address is used to keep track which user
// has added how much.
func (k Keeper) AddLiquidity(ctx context.Context, address sdk.AccAddress, denom string, amount math.Int) (math.Int, error) {
	if !k.DenomKeeper.IsValidDenom(ctx, denom) {
		return math.Int{}, types.ErrDenomNotFound
	}

	if k.BankKeeper.SpendableCoin(ctx, address, denom).Amount.LT(amount) {
		return math.Int{}, types.ErrNotEnoughFunds
	}

	coins := sdk.NewCoins(sdk.NewCoin(denom, amount))
	if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, address, types.PoolLiquidity, coins); err != nil {
		return math.Int{}, fmt.Errorf("could not send coins to module: %w", err)
	}

	// The dex works by routing all trades via XKP. The chain is initialized with funds for the reserve, which adds
	// those funds to the dex. When no liquidity for XKP has been added, we refuse new liquidity as long as no
	// liquidity for XKP is added.

	liquidityPool := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	liqBase := k.BankKeeper.SpendableCoins(ctx, liquidityPool.GetAddress()).AmountOf(constants.BaseCurrency)
	if liqBase.IsZero() && denom != constants.BaseCurrency {
		return math.Int{}, types.ErrBaseLiqEmpty
	}

	_, liq := k.addLiquidity(ctx, denom, address.String(), amount, nil)

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			"liquidity_added",
			sdk.Attribute{Key: "denom", Value: denom},
			sdk.Attribute{Key: "amount", Value: amount.String()},
			sdk.Attribute{Key: "address", Value: address.String()},
			sdk.Attribute{Key: "index", Value: strconv.Itoa(int(liq.Index))},
		),
	)

	return liq.Amount, nil
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
			k.SetLiquidity(ctx, denom, liq)
			liquidityEntries[index] = liq
			return liquidityEntries, liq
		}
	}

	liq := types.Liquidity{Address: address, Amount: amount}
	liq = k.SetLiquidity(ctx, denom, liq)
	liquidityEntries = append(liquidityEntries, liq)

	return liquidityEntries, liq
}

func (k Keeper) LiquidityIterator(ctx context.Context, denom string) cache.Iterator[uint64, types.Liquidity] {
	rng := collections.NewPrefixedPairRange[string, uint64](denom)
	return k.liquidityEntries.Iterator(ctx, rng, denom)
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

func (k Keeper) GetAllLiquidity(ctx context.Context) (list []types.DenomLiquidity) {
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		iterator := k.LiquidityIterator(ctx, denom)

		denomLiquidity := types.DenomLiquidity{Denom: denom}
		for iterator.Valid() {
			liq := iterator.GetNext()
			denomLiquidity.Entries = append(denomLiquidity.Entries, &liq)
		}

		sort.SliceStable(denomLiquidity.Entries, func(i, j int) bool {
			return denomLiquidity.Entries[i].Index < denomLiquidity.Entries[j].Index
		})
	}

	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Denom < list[j].Denom
	})

	return
}

// RemoveLiquidity removes a liquidity from the store
func (k Keeper) RemoveLiquidity(ctx context.Context, denom string, index uint64) {
	k.liquidityEntries.Remove(ctx, denom, index)
}

// UpdateVirtualLiquidities updates the virtual liquidity for each pair. This method is called at the end of each block.
// The virtual liquidty is only updated when there is no actual liquidity for that denom. When the virtual liquidity is
// 0, it means the pair probably just have been created and will be set to the initial virtual amount. If the amount
// of actual liquidity is zero and the amount of virtual liquidity is not zero, we slowly decrease the amount of virtual
// liquidity to increase that denom's price.
func (k Keeper) UpdateVirtualLiquidities(ctx context.Context) error {
	decay := k.getVirtualLiquidityDecay(ctx)
	liquidityPool := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	poolBalance := k.BankKeeper.SpendableCoins(ctx, liquidityPool.GetAddress())

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		if denom != constants.BaseCurrency {
			liq := poolBalance.AmountOf(denom)
			if liq.LT(k.DenomKeeper.MinLiquidity(ctx, denom)) {
				ratio, err := k.DenomKeeper.GetRatio(ctx, denom)
				if err != nil {
					return fmt.Errorf("could not get ratio for %v: %w", denom, err)
				}

				ratio.Ratio = ratio.Ratio.Mul(decay)
				k.DenomKeeper.SetRatio(ctx, ratio)
			}
		}
	}

	return nil
}

func (k Keeper) GetDenomValue(ctx context.Context, denom string) (math.LegacyDec, error) {
	if denom == constants.BaseCurrency {
		liq := k.GetLiquiditySum(ctx, constants.BaseCurrency)
		return liq.ToLegacyDec(), nil
	}

	liq := k.GetFullLiquidityOther(ctx, denom)
	price, err := k.CalculatePrice(ctx, denom, constants.BaseCurrency)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return liq.Mul(price), nil
}

func (k Keeper) PrepareAdditionalLiquidity(ctx *types.TradeContext) {
	if ctx.HasOneStep() {
		return
	}

	liqFrom := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.TradeDenomGiving)
	liqTo := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.TradeDenomReceiving)
	liqBase := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(constants.BaseCurrency)

	var (
		maxValue     = liqBase.ToLegacyDec()
		ratioFrom    denomtypes.Ratio
		ratioTo      denomtypes.Ratio
		liqValueFrom math.LegacyDec
		liqValueTo   math.LegacyDec
	)

	if ctx.TradeDenomGiving != constants.BaseCurrency {
		ratioFrom, _ = k.DenomKeeper.GetRatio(ctx, ctx.TradeDenomGiving)
		liqValueFrom = liqFrom.ToLegacyDec().Quo(ratioFrom.Ratio) // C
		maxValue = math.LegacyMaxDec(liqValueFrom, maxValue)
	}

	if ctx.TradeDenomReceiving != constants.BaseCurrency {
		ratioTo, _ = k.DenomKeeper.GetRatio(ctx, ctx.TradeDenomReceiving)
		liqValueTo = liqTo.ToLegacyDec().Quo(ratioTo.Ratio) // C
		maxValue = math.LegacyMaxDec(liqValueTo, maxValue)
	}

	ctx.SetAdditionalLiquidity(constants.BaseCurrency, maxValue.Sub(liqBase.ToLegacyDec()))

	if ctx.TradeDenomGiving != constants.BaseCurrency {
		if liqValueFrom.LT(maxValue) {
			missing := maxValue.Sub(liqValueFrom).Mul(ratioFrom.Ratio)
			ctx.SetAdditionalLiquidity(ctx.TradeDenomGiving, missing)
		} else {
			ctx.SetAdditionalLiquidity(ctx.TradeDenomGiving, math.LegacyZeroDec())
		}
	}

	if ctx.TradeDenomReceiving != constants.BaseCurrency {
		if liqValueTo.LT(maxValue) {
			missing := maxValue.Sub(liqValueTo).Mul(ratioTo.Ratio)
			ctx.SetAdditionalLiquidity(ctx.TradeDenomReceiving, missing)
		} else {
			ctx.SetAdditionalLiquidity(ctx.TradeDenomReceiving, math.LegacyZeroDec())
		}
	}
}
