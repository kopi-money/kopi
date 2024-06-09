package keeper

import (
	"context"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
	"strconv"
)

type LiquidityProvider struct {
	index          uint64
	address        string
	amountProvided math.LegacyDec
	shareProvided  math.LegacyDec
}

type LiquidityProviders []*LiquidityProvider

func (lps *LiquidityProviders) sumProvided() math.LegacyDec {
	sum := math.LegacyZeroDec()
	for _, lp := range *lps {
		sum = sum.Add(lp.amountProvided)
	}

	return sum
}

func (lps *LiquidityProviders) provided() *LiquidityProviders {
	sumProvided := lps.sumProvided()
	for _, lp := range *lps {
		if sumProvided.GT(math.LegacyZeroDec()) {
			lp.shareProvided = lp.amountProvided.Quo(sumProvided)
		} else {
			lp.shareProvided = math.LegacyZeroDec()
		}
	}

	return lps
}

func (k Keeper) determineLiquidityProviders(ctx context.Context, eventManager sdk.EventManagerI, ordersCaches *types.OrdersCaches, amountToReceiveLeft math.Int, denomFrom, denomTo string) (*LiquidityProviders, math.Int, error) {
	var (
		liquidityProviders LiquidityProviders
		liquidityUsed      math.Int
		deleteIndexes      []uint64
		sumUsed            = math.ZeroInt()
	)

	// Iterate over the existing liquidity entries for this currency
	liquidityList := ordersCaches.LiquidityMap.Get(denomTo)
	for _, liq := range liquidityList {
		if amountToReceiveLeft.LTE(math.ZeroInt()) {
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

		lp := LiquidityProvider{index: liq.Index, address: liq.Address, amountProvided: liquidityUsed.ToLegacyDec()}
		liquidityProviders = append(liquidityProviders, &lp)

		liq.Amount = liq.Amount.Sub(liquidityUsed)
		if liq.Amount.Equal(math.ZeroInt()) {
			k.RemoveLiquidity(ctx, denomTo, liq.Index, liquidityUsed)
			deleteIndexes = append(deleteIndexes, liq.Index)
		} else {
			k.SetLiquidity(ctx, liq, liquidityUsed.Neg())
		}

		sumUsed = sumUsed.Add(liquidityUsed)

		eventManager.EmitEvent(
			sdk.NewEvent("liquidity_used",
				sdk.Attribute{Key: "address", Value: liq.Address},
				sdk.Attribute{Key: "denom", Value: denomFrom},
				sdk.Attribute{Key: "amount", Value: liquidityUsed.String()},
				sdk.Attribute{Key: "index", Value: strconv.Itoa(int(liq.Index))},
			),
		)
	}

	coins := sdk.NewCoins(sdk.NewCoin(denomTo, sumUsed))
	//poolBalance := ordersCaches.DexPool.Get().Sub(coins...)
	//ordersCaches.DexPool.Set(poolBalance)

	liquidityList = removeIndexes(liquidityList, deleteIndexes)
	ordersCaches.LiquidityMap.Set(denomTo, liquidityList)

	if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolLiquidity, types.PoolTrade, coins); err != nil {
		return nil, math.Int{}, err
	}

	return &liquidityProviders, amountToReceiveLeft, nil
}

func removeIndexes(liquidityList []types.Liquidity, indexes []uint64) (list []types.Liquidity) {
	if len(indexes) == 0 {
		return liquidityList
	}

	for _, entry := range liquidityList {
		if len(indexes) > 0 && entry.Index == indexes[0] {
			indexes = indexes[1:]
			continue
		}

		list = append(list, entry)
	}

	return
}

func (k Keeper) distributeFeeForLiquidityProviders(ctx context.Context, ordersCaches *types.OrdersCaches, liquidityProviders *LiquidityProviders, feeForLiquidityProvidersLeft math.Int, denom string) error {
	providerFee := k.getProviderFee(ctx)
	liquidityEntries := ordersCaches.LiquidityMap.Get(denom)

	liquidityProviderIndex := 0
	for feeForLiquidityProvidersLeft.GT(math.ZeroInt()) {
		liquidityProvider := (*liquidityProviders)[liquidityProviderIndex]
		liquidityProviderIndex += 1

		amount := math.MinInt(feeForLiquidityProvidersLeft, liquidityProvider.amountProvided.RoundInt())
		feeForLiquidityProvidersLeft = feeForLiquidityProvidersLeft.Sub(amount)
		liquidityProvider.amountProvided = liquidityProvider.amountProvided.Mul(providerFee)
		liquidityEntries, _ = k.addLiquidity(ctx, denom, liquidityProvider.address, amount, liquidityEntries)

		coins := sdk.NewCoins(sdk.NewCoin(denom, amount))
		//poolBalance := ordersCaches.DexPool.Get().Add(coins...)
		//ordersCaches.DexPool.Set(poolBalance)

		if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolTrade, types.PoolLiquidity, coins); err != nil {
			return err
		}
	}

	ordersCaches.LiquidityMap.Set(denom, liquidityEntries)

	return nil
}

func (k Keeper) distributeGivenFunds(ctx context.Context, ordersCaches *types.OrdersCaches, liquidityProviders *LiquidityProviders, fundsToDistribute math.Int, denom string) error {
	liquidityEntries := ordersCaches.LiquidityMap.Get(denom)
	provided := liquidityProviders.provided()

	for _, liquidityProvider := range *provided {
		eligable := liquidityProvider.shareProvided.Mul(fundsToDistribute.ToLegacyDec()).RoundInt()
		liquidityEntries, _ = k.addLiquidity(ctx, denom, liquidityProvider.address, eligable, liquidityEntries)
	}

	coins := sdk.NewCoins(sdk.NewCoin(denom, fundsToDistribute))
	//poolBalance := ordersCaches.DexPool.Get().Add(coins...)
	//ordersCaches.DexPool.Set(poolBalance)

	if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolTrade, types.PoolLiquidity, coins); err != nil {
		return err
	}

	return nil
}
