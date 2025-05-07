package keeper

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/cache"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/trading"
	"strconv"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/constants"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
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

func (k Keeper) LiquidityIterator(ctx context.Context, denom string) cache.Iterator[uint64, types.Liquidity] {
	return k.liquidityEntries.Iterator(ctx, nil, denom)
}

func (k Keeper) AddLiquidity(ctx context.Context, address sdk.AccAddress, denom string, amount math.Int) error {
	return k.AddLiquidityWithCompound(ctx, address, denom, amount, false)
}

// AddLiquidityWithCompound adds liquidity to the dex for a given amount and address. The address is used to keep track which user
// has added how much.
func (k Keeper) AddLiquidityWithCompound(ctx context.Context, address sdk.AccAddress, denom string, amount math.Int, autoCompound bool) error {
	if !k.DenomKeeper.IsValidDenom(ctx, denom) {
		return denomtypes.ErrInvalidDexAsset
	}

	if k.BankKeeper.SpendableCoin(ctx, address, denom).Amount.LT(amount) {
		return types.ErrNotEnoughFunds
	}

	coins := sdk.NewCoins(sdk.NewCoin(denom, amount))
	if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, address, types.PoolLiquidity, coins); err != nil {
		return fmt.Errorf("send coins to module: %w", err)
	}

	// The dex works by giving each denom a price in relation to XKP. When there is no price and no liquidity, the
	// prices are not meaningful. Thus we force adding liquidity first to XKP before to other denoms. This case is only
	// relevant for the genesis and is pretty much obsolete in later stages.

	liquidityPool := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	liqBase := k.BankKeeper.SpendableCoins(ctx, liquidityPool.GetAddress()).AmountOf(constants.BaseCurrency)
	if liqBase.IsZero() && denom != constants.BaseCurrency {
		return types.ErrBaseLiqEmpty
	}

	// Liquidity added by protocol addresses won't get a dedicated liquidity index.

	var positionIndex uint64
	if !k.addressIsExcluded(ctx, address.String()) {
		var has bool
		positionIndex, has = k.liquidityPositionNextIndex.Get(ctx)
		if !has {
			positionIndex = 0
		}

		positionIndex++
		k.liquidityPositionNextIndex.Set(ctx, positionIndex)
	}

	_, liq := k.addLiquidity(ctx, denom, address.String(), amount, nil, positionIndex)

	if positionIndex > 0 {
		amountUSD, _ := k.DenomKeeper.GetValueInUSD(ctx, denom, amount.ToLegacyDec())

		sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
			sdk.NewEvent(
				"liquidity_added",
				sdk.Attribute{Key: "denom", Value: denom},
				sdk.Attribute{Key: "amount", Value: amount.String()},
				sdk.Attribute{Key: "amount_usd", Value: amountUSD.String()},
				sdk.Attribute{Key: "address", Value: address.String()},
				sdk.Attribute{Key: "index", Value: strconv.Itoa(int(liq.Index))},
				sdk.Attribute{Key: "position_index", Value: strconv.Itoa(int(positionIndex))},
			),
		)

		k.liquidityPositions.Set(ctx, address.String(), positionIndex, types.LiquidityPosition{
			AutoCompound: autoCompound,
			CreatedAt:    sdk.UnwrapSDKContext(ctx).BlockHeight(),
		})
	}

	return nil
}

// addLiquidity adds liquidity to the DEX. The liquidity is defined by address and position index. When being used for
// trades, liquidity is taken from the beinning of the list. If liquidity would be added to the first found occurrence,
// liquidity added by whales would be used more often compared to smaller liquidity entries. To make this more fair,
// liquidity is added to the second entry of an address or in a new entry at the end.
func (k Keeper) addLiquidity(ctx context.Context, denom, address string, amount math.Int, liquidityEntries []types.Liquidity, positionIndex uint64) ([]types.Liquidity, types.Liquidity) {
	if liquidityEntries == nil {
		liquidityEntries = k.liquidityEntries.Iterator(ctx, nil, denom).GetAll()
	}

	k.AddLiquidityAddressSum(ctx, address, denom, amount)

	seen := false
	for index, liq := range liquidityEntries {
		if liq.Address == address && liq.PositionIndex == positionIndex {
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

	liq := types.Liquidity{Address: address, Amount: amount, PositionIndex: positionIndex}
	liq = k.SetLiquidity(ctx, denom, liq)
	liquidityEntries = append(liquidityEntries, liq)

	return liquidityEntries, liq
}

func (k Keeper) GetLiquidityByAddress(ctx context.Context, denom, address string) math.Int {
	sum := math.ZeroInt()

	iterator := k.liquidityEntries.Iterator(ctx, nil, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()

		if liq.Address == address {
			sum = sum.Add(liq.Amount)
		}
	}

	return sum
}

func (k Keeper) GetLiquidityByPositionIndex(ctx context.Context, denom string, positionIndex uint64) math.Int {
	sum := math.ZeroInt()

	iterator := k.liquidityEntries.Iterator(ctx, nil, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()

		if liq.PositionIndex == positionIndex {
			sum = sum.Add(liq.Amount)
		}
	}

	return sum
}

func (k Keeper) GetLiquidityEntriesByAddress(ctx context.Context, denom, address string) int {
	num := 0

	iterator := k.liquidityEntries.Iterator(ctx, nil, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()
		if liq.Address == address {
			num++
		}
	}

	return num
}

func (k Keeper) exportLiquidityEntries(ctx context.Context) (list []types.GenesisLiquidityEntry) {
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		iterator := k.liquidityEntries.Iterator(ctx, nil, denom)

		for iterator.Valid() {
			liq := iterator.GetNext()
			list = append(list, types.GenesisLiquidityEntry{
				Index:         liq.Index,
				Address:       liq.Address,
				Amount:        liq.Amount,
				PositionIndex: liq.GetPositionIndex(),
				Denom:         denom,
			})
		}
	}

	return
}

func (k Keeper) RemoveLiquidity(ctx context.Context, denom string, index uint64) {
	k.liquidityEntries.Remove(ctx, denom, index)
}

// UpdateRatios updates the ratio of each DEX denom. This method is called at the end of each block. The ratio is only
// updated when there is no or only little liquidity for that denom. The ratio is slowly changed to increase the price
// to slowly incentivize users adding liquidity.
func (k Keeper) UpdateRatios(ctx context.Context) error {
	factor := k.getPriceIncreasingFactor(ctx)
	liquidityPool := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	poolBalance := k.BankKeeper.SpendableCoins(ctx, liquidityPool.GetAddress())

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		if denom != constants.BaseCurrency {
			liq := poolBalance.AmountOf(denom)
			if liq.LT(k.DenomKeeper.MinLiquidity(ctx, denom)) {
				// If a kCoin is above parity, the protocol mints+sells and thereby already adds liquidity.
				if k.skipKCoin(ctx, denom) {
					continue
				}

				ratio, _ := k.DenomKeeper.GetRatio(ctx, denom)
				ratio.Ratio = ratio.Ratio.Mul(factor)
				k.DenomKeeper.SetRatio(ctx, ratio)
			}
		}
	}

	return nil
}

func (k Keeper) skipKCoin(ctx context.Context, denom string) bool {
	if !k.DenomKeeper.IsKCoin(ctx, denom) {
		return false
	}

	aboveParity, err := k.DenomKeeper.IsAboveParity(ctx, denom)
	if err != nil {
		k.Logger().Error(fmt.Sprintf("aboveParity: %v", err))
		return false
	}

	return aboveParity
}

func (k Keeper) GetDenomLiquidityValueInBase(ctx context.Context, denom string) (math.LegacyDec, error) {
	liq := k.GetPoolLiquidity(ctx, denom).ToLegacyDec()
	return k.DenomKeeper.GetValueInBase(ctx, denom, liq)
}

// CalculateTradeLiquidity calculates the liquidity for the two denoms affected by a trade. It uses the effective
// liquidity of the two denoms and their value in the base denom, the trade value is the minimum of the two denoms'
// effective liquidity. That means, if one denom has a lot of liquidity and the other one has only little, the trade
// will only use little liquidity.
func (k Keeper) CalculateTradeLiquidity(ctx context.Context, tradeDenomGiving, tradeDenomReceiving string) (trading.Liquidity, trading.Liquidity, error) {
	liqFrom, liqFromBase, minimumTradeValueBaseFrom := k.getEffectiveLiquidity(ctx, tradeDenomGiving)
	if !liqFrom.IsPositive() {
		return trading.Liquidity{}, trading.Liquidity{}, types.ErrNoLiquidityGiving
	}

	liqTo, liqToBase, minimumTradeValueBaseTo := k.getEffectiveLiquidity(ctx, tradeDenomReceiving)
	if !liqTo.IsPositive() {
		return trading.Liquidity{}, trading.Liquidity{}, types.ErrNoLiquidityReceiving
	}

	minimumTradeValue := math.LegacyMaxDec(minimumTradeValueBaseFrom, minimumTradeValueBaseTo)
	tradeValue := math.LegacyMinDec(liqFromBase, liqToBase)
	tradeValue = math.LegacyMaxDec(tradeValue, minimumTradeValue)

	liqShareFrom := tradeValue.Quo(liqFromBase)
	liqShareTo := tradeValue.Quo(liqToBase)

	return trading.Liquidity{
			Actual: liqFrom.Mul(liqShareFrom),
		}, trading.Liquidity{
			Actual: liqTo.Mul(liqShareTo),
		}, nil
}

// CalculateTradeLiquidityFromCache is the same as CalculateTradeLiquidity except that the liquidity values are read
// from the cache.
func (k Keeper) CalculateTradeLiquidityFromCache(ctx types.TradeContext, tradeDenomGiving, tradeDenomReceiving string) (trading.Liquidity, trading.Liquidity, error) {
	liqFrom := k.getEffectiveLiquidityFromCache(ctx, tradeDenomGiving)
	if !liqFrom.GetFull().IsPositive() {
		return trading.Liquidity{}, trading.Liquidity{}, types.ErrNoLiquidityGiving
	}

	liqTo := k.getEffectiveLiquidityFromCache(ctx, tradeDenomReceiving)
	if !liqTo.GetFull().IsPositive() {
		return trading.Liquidity{}, trading.Liquidity{}, types.ErrNoLiquidityReceiving
	}

	minimumTradeValue := math.LegacyMaxDec(liqFrom.MinimumTradeValueBase, liqTo.MinimumTradeValueBase)
	tradeValue := math.LegacyMinDec(liqFrom.TradeValueBase, liqTo.TradeValueBase)
	tradeValue = math.LegacyMaxDec(tradeValue, minimumTradeValue)

	liqFromT := liqFrom.AdjustToTradeValue(tradeValue)
	liqToT := liqTo.AdjustToTradeValue(tradeValue)

	return liqFromT, liqToT, nil
}

// RemoveAllLiquidityForDenom is called when a denom is removed from the DEX and sends all provided liquidity to the
// providers' addresses.
func (k Keeper) RemoveAllLiquidityForDenom(ctx context.Context, denom string) error {
	iterator := k.liquidityEntries.Iterator(ctx, nil, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()

		acc, _ := sdk.AccAddressFromBech32(liq.Address)
		coins := sdk.NewCoins(sdk.NewCoin(denom, liq.Amount))
		if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolLiquidity, acc, coins); err != nil {
			return fmt.Errorf("send coins from module to account: %w", err)
		}

		k.RemoveLiquidity(ctx, denom, liq.Index)
	}

	return nil
}

func (k Keeper) getLiquidityForAddress(ctx context.Context, address string) (coins sdk.Coins) {
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		coins = coins.Add(sdk.NewCoin(denom, k.GetLiquidityByAddress(ctx, denom, address)))
	}

	return
}

// canUnlock checks if liquidity can be removed from a given liquidity position. The function will return falls if not
// enough blocks have been created since creating of this position.
func (k Keeper) canUnlock(ctx context.Context, address string, positionIndex uint64) (bool, error) {
	position, has := k.liquidityPositions.Get(ctx, address, positionIndex)
	if !has {
		return false, fmt.Errorf("unable to find liquidity position: %v / %v", address, positionIndex)
	}

	height := sdk.UnwrapSDKContext(ctx).BlockHeight()
	canUnlockAfter := position.CreatedAt + k.GetParams(ctx).MinimumLiquidityLockInBlocks
	return height >= canUnlockAfter, nil
}

func (k Keeper) getWithdrawableLiquidityForAddressForDenom(ctx context.Context, address, denom string) (math.Int, error) {
	sum := math.ZeroInt()

	iterator := k.liquidityEntries.Iterator(ctx, nil, denom)
	for iterator.Valid() {
		value := iterator.GetNext()
		if value.Address != address {
			continue
		}

		canUnlock, err := k.canUnlock(ctx, address, value.PositionIndex)
		if err != nil {
			return math.Int{}, fmt.Errorf("can unlock: %w", err)
		}

		if !canUnlock {
			continue
		}

		sum = sum.Add(value.Amount)
	}

	return sum, nil
}
