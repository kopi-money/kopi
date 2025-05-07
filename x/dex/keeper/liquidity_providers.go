package keeper

import (
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

type LiquidityProvider struct {
	index         uint64
	positionIndex uint64
	address       string
	amount        math.Int
}

type LiquidityProviders []*LiquidityProvider

func (lps LiquidityProviders) amountSum() math.Int {
	sum := math.ZeroInt()
	for _, lp := range lps {
		sum = sum.Add(lp.amount)
	}

	return sum
}

// determineLiquidityProviders is called when executing a trade: When, for example, a user trades kUSD to XKP, this
// function determines which XKP liquidity entries to use. The liquidity providers will in return receive the kUSD given
// into the global pool.
func (k Keeper) determineLiquidityProviders(ctx types.TradeContext, amountToReceiveLeft math.Int, denomTo string, receivingAddress string, protocolTrade bool) (LiquidityProviders, math.Int, error) {
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

		// If the trade is not a protocol trade, we don't use liquidity coming from the same address as the address
		// trading to prevent self-trades.
		if !protocolTrade && liq.Address == receivingAddress {
			continue
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

		liquidityProviders = append(liquidityProviders, &LiquidityProvider{
			index:         liq.Index,
			address:       liq.Address,
			amount:        liquidityUsed,
			positionIndex: liq.PositionIndex,
		})
		sumUsed = sumUsed.Add(liquidityUsed)
		liq.Amount = liq.Amount.Sub(liquidityUsed)

		k.AddLiquidityAddressSum(ctx, liq.Address, denomTo, liquidityUsed.Neg())
		if liq.Amount.IsZero() {
			k.RemoveLiquidity(ctx, denomTo, liq.Index)
			deleteIndexes = append(deleteIndexes, index)
		} else {
			k.SetLiquidity(ctx, denomTo, liq)
			liquidityList[index] = liq
		}
	}

	// The data stored in cache has to be be updated too the be aligned with what is stored in storage.
	ctx.OrdersCaches.SubtractLiquidity(denomTo, sumUsed)
	liquidityList = removeIndexes(liquidityList, deleteIndexes)
	ctx.OrdersCaches.LiquidityMap.Set(denomTo, liquidityList)
	ctx.TradeBalances.AddTransfer(
		ctx.OrdersCaches.AccPoolLiquidity.Get().String(),
		ctx.OrdersCaches.AccPoolTrade.Get().String(),
		denomTo, sumUsed,
	)

	movLiq := k.capMovingLiquidity(ctx, denomTo)
	ctx.OrdersCaches.MovingLiquidity.Set(denomTo, movLiq)

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

// distributeGivenFunds is called when executing a trade. When, for example, a user trades kUSD to XKP and XKP has been
// removed from the pool, the liquidity providers whose liquidity is used are given kUSD in exchange in this function.
func (k Keeper) distributeGivenFunds(ctx types.TradeContext, ordersCaches *types.OrdersCaches, liquidityProviders LiquidityProviders, fundsToDistribute, fundsTaken math.Int, denom string) error {
	var (
		liquidityEntries           = ordersCaches.LiquidityMap.Get(denom)
		fundsToDistributeRemaining = fundsToDistribute
		fundsTakenRemaining        = fundsTaken
		eligable                   math.Int
		taken                      math.Int
	)

	sum := liquidityProviders.amountSum().ToLegacyDec()
	if !sum.IsPositive() {
		return fmt.Errorf("provided sum is not positive: %v", sum)
	}

	for index, liquidityProvider := range liquidityProviders {
		if index+1 == len(liquidityProviders) {
			// If this is the last liquidity provider, we use the remaining funds to make sure there are no leftovers
			// (cause by potential rounding issues)
			eligable = fundsToDistributeRemaining
			taken = fundsTakenRemaining
		} else {
			share := liquidityProvider.amount.ToLegacyDec().Quo(sum) // C
			eligable = share.Mul(fundsToDistribute.ToLegacyDec()).RoundInt()
			taken = share.Mul(fundsTakenRemaining.ToLegacyDec()).RoundInt()
		}

		if eligable.IsPositive() {
			positionIndex := liquidityProvider.positionIndex
			if positionIndex > 0 {
				sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
					sdk.NewEvent("liquidity_swap",
						sdk.Attribute{Key: "position_index", Value: strconv.Itoa(int(liquidityProvider.positionIndex))},
						sdk.Attribute{Key: "denom_received", Value: ctx.TradeDenomReceiving},
						sdk.Attribute{Key: "denom_used", Value: ctx.TradeDenomGiving},
						sdk.Attribute{Key: "liquidity_used", Value: taken.String()},
						sdk.Attribute{Key: "liquidity_received", Value: eligable.String()},
					),
				)
			}

			liquidityEntries, _ = k.addLiquidity(ctx.Context, denom, liquidityProvider.address, eligable, liquidityEntries, positionIndex)
			fundsToDistributeRemaining = fundsToDistributeRemaining.Sub(eligable)
		}
	}

	ordersCaches.LiquidityMap.Set(denom, liquidityEntries)
	ctx.OrdersCaches.AddLiquidity(denom, fundsToDistribute)
	ctx.TradeBalances.AddTransfer(
		ordersCaches.AccPoolTrade.Get().String(),
		ordersCaches.AccPoolLiquidity.Get().String(),
		denom, fundsToDistribute,
	)

	return nil
}

func (k Keeper) getLiquidityAddressSum(ctx context.Context, address, denom string) math.Int {
	sum, has := k.liquidityAddressSum.Get(ctx, address, denom)
	if !has {
		return math.ZeroInt()
	}

	return sum.Sum
}

func (k Keeper) getLiquidityAddressSumDec(ctx context.Context, address, denom string) math.LegacyDec {
	return k.getLiquidityAddressSum(ctx, address, denom).ToLegacyDec()
}

func (k Keeper) GetLiquidityAddressSums(ctx context.Context, address string) (coins sdk.Coins) {
	iterator := k.liquidityAddressSum.Iterator(ctx, nil, address)
	for iterator.Valid() {
		keyValue := iterator.GetNextKeyValue()
		coins = coins.Add(sdk.NewCoin(keyValue.Key(), keyValue.Value().Value().Sum))
	}

	return
}

func (k Keeper) AddLiquidityAddressSum(ctx context.Context, address, denom string, amount math.Int) {
	sum := k.getLiquidityAddressSum(ctx, address, denom)
	sum = math.MaxInt(sum.Add(amount), math.ZeroInt())
	k.liquidityAddressSum.Set(ctx, address, denom, types.LiquiditySum{Sum: sum})
}
