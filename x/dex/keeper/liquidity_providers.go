package keeper

import (
	"cosmossdk.io/math"
	"fmt"
	"github.com/kopi-money/kopi/x/dex/types"
)

type LiquidityProvider struct {
	index   uint64
	address string
	amount  math.Int
}

type LiquidityProviders []*LiquidityProvider

func (lps LiquidityProviders) amountSum() math.Int {
	sum := math.ZeroInt()
	for _, lp := range lps {
		sum = sum.Add(lp.amount)
	}

	return sum
}

func (k Keeper) determineLiquidityProviders(ctx types.TradeStepContext, amountToReceiveLeft math.Int, denomTo string) (LiquidityProviders, math.Int, error) {
	var (
		liquidityProviders LiquidityProviders
		liquidityUsed      math.Int
		deleteIndexes      []int
		sumUsed            = math.ZeroInt()
	)

	// Iterate over the existing liquidity entries for this currency
	liquidityList := ctx.OrdersCaches.LiquidityMap.Get(denomTo)
	for index, liq := range liquidityList {
		if !amountToReceiveLeft.IsPositive() {
			break
		}

		if amountToReceiveLeft.LT(liq.Amount) {
			// the current liquidity entry will not be fully used
			liquidityUsed = amountToReceiveLeft
			amountToReceiveLeft = math.ZeroInt()
		} else {
			// the current liquidity entry will be fully used
			liquidityUsed = liq.Amount
			amountToReceiveLeft = amountToReceiveLeft.Sub(liq.Amount)
		}

		lp := LiquidityProvider{index: liq.Index, address: liq.Address, amount: liquidityUsed}
		liquidityProviders = append(liquidityProviders, &lp)
		sumUsed = sumUsed.Add(liquidityUsed)
		liq.Amount = liq.Amount.Sub(liquidityUsed)

		if liq.Amount.IsZero() {
			k.RemoveLiquidity(ctx.TradeContext.Context, denomTo, liq.Index)
			deleteIndexes = append(deleteIndexes, index)
		} else {
			k.SetLiquidity(ctx.TradeContext.Context, denomTo, liq)
			liquidityList[index] = liq
		}
	}

	ctx.OrdersCaches.LiquidityPool.Get().Sub(denomTo, sumUsed)
	liquidityList = removeIndexes(liquidityList, deleteIndexes)
	ctx.OrdersCaches.LiquidityMap.Set(denomTo, liquidityList)
	ctx.TradeBalances.AddTransfer(
		ctx.OrdersCaches.AccPoolLiquidity.Get().String(),
		ctx.OrdersCaches.AccPoolTrade.Get().String(),
		denomTo, sumUsed,
	)

	return liquidityProviders, amountToReceiveLeft, nil
}

func removeIndexes(liquidityList []types.Liquidity, indexes []int) []types.Liquidity {
	for len(indexes) > 0 {
		index := indexes[len(indexes)-1]
		indexes = indexes[:len(indexes)-1]
		liquidityList = append(liquidityList[:index], liquidityList[index+1:]...)
	}

	return liquidityList
}

// distributeSellFee distributes the sell fee among the liquidity providers. For example, when a user has given 400 XKP
// and received 100 kUSD, the user actually receives only 99 kUSD. The 1 kUSD will be given to liquidity providers.
// However, it is not split evenly to prevent micro liquidity amount in the queue. Instead, it is given to few users.
// Those users will in return get a smaller share of the given funds. The receive factor determines how much of the
// given funds each provider will receive for each received
func (k Keeper) distributeSellFee(ctx types.TradeStepContext, liquidityProviders LiquidityProviders, feeForLiquidityProviders math.Int, receiveFactor math.LegacyDec, feeDenom string) {
	liquidityEntries := ctx.TradeContext.OrdersCaches.LiquidityMap.Get(feeDenom)

	feeForLiquidityProvidersLeft := feeForLiquidityProviders
	for _, liquidityProvider := range liquidityProviders {
		// Calculate how much of the given funds this LP is eligable for
		eligable := liquidityProvider.amount.ToLegacyDec().Mul(receiveFactor)

		if feeForLiquidityProvidersLeft.IsPositive() {
			sellFeeAmount := math.MinInt(feeForLiquidityProvidersLeft, liquidityProvider.amount)
			feeForLiquidityProvidersLeft = feeForLiquidityProvidersLeft.Sub(sellFeeAmount)

			eligable = eligable.Sub(sellFeeAmount.ToLegacyDec().Mul(receiveFactor))
			liquidityEntries, _ = k.addLiquidity(ctx.TradeContext.Context, feeDenom, liquidityProvider.address, sellFeeAmount, liquidityEntries)
		}

		liquidityProvider.amount = eligable.RoundInt()
	}

	ctx.OrdersCaches.LiquidityPool.Get().Add(feeDenom, feeForLiquidityProviders)
	ctx.OrdersCaches.LiquidityMap.Set(feeDenom, liquidityEntries)

	ctx.TradeContext.TradeBalances.AddTransfer(
		ctx.OrdersCaches.AccPoolTrade.Get().String(),
		ctx.OrdersCaches.AccPoolLiquidity.Get().String(),
		feeDenom, feeForLiquidityProviders,
	)
}

func (k Keeper) distributeGivenFunds(ctx types.TradeStepContext, ordersCaches *types.OrdersCaches, liquidityProviders LiquidityProviders, fundsToDistribute math.Int, denom string) error {
	var (
		liquidityEntries           = ordersCaches.LiquidityMap.Get(denom)
		fundsToDistributeRemaining = fundsToDistribute
		eligable                   math.Int
	)

	sum := liquidityProviders.amountSum().ToLegacyDec()
	if !sum.IsPositive() {
		return fmt.Errorf("provided sum is not positive")
	}

	for index, liquidityProvider := range liquidityProviders {
		if index+1 == len(liquidityProviders) {
			// In case of the last liquidity provider, we use the remaining funds to make sure there are no leftovers
			// (cause by potential rounding issues)
			eligable = fundsToDistributeRemaining
		} else {
			share := liquidityProvider.amount.ToLegacyDec().Quo(sum) // C
			eligable = share.Mul(fundsToDistribute.ToLegacyDec()).RoundInt()
		}

		if eligable.IsPositive() {
			liquidityEntries, _ = k.addLiquidity(ctx.TradeContext.Context, denom, liquidityProvider.address, eligable, liquidityEntries)
			fundsToDistributeRemaining = fundsToDistributeRemaining.Sub(eligable)
		}
	}

	ordersCaches.LiquidityMap.Set(denom, liquidityEntries)
	ordersCaches.LiquidityPool.Get().Add(denom, fundsToDistribute)
	ctx.TradeBalances.AddTransfer(
		ordersCaches.AccPoolTrade.Get().String(),
		ordersCaches.AccPoolLiquidity.Get().String(),
		denom, fundsToDistribute,
	)

	return nil
}
