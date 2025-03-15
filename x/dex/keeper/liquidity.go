package keeper

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/cache"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"sort"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

func (k Keeper) LiquidityIterator(ctx context.Context, denom string) cache.Iterator[uint64, types.Liquidity] {
	return k.liquidityEntries.Iterator(ctx, nil, denom)
}

// AddLiquidity adds liquidity to the dex for a given amount and address. The address is used to keep track which user
// has added how much.
func (k Keeper) AddLiquidity(ctx context.Context, address sdk.AccAddress, denom string, amount math.Int) (math.Int, error) {
	return k.AddLiquidityWithCompound(ctx, address, denom, amount, false)
}

func (k Keeper) AddLiquidityWithCompound(ctx context.Context, address sdk.AccAddress, denom string, amount math.Int, autoCompound bool) (math.Int, error) {
	if !k.DenomKeeper.IsValidDenom(ctx, denom) {
		return math.Int{}, denomtypes.ErrInvalidDexAsset
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
		})
	}

	return liq.Amount, nil
}

func (k Keeper) addLiquidity(ctx context.Context, denom, address string, amount math.Int, liquidityEntries []types.Liquidity, positionIndex uint64) ([]types.Liquidity, types.Liquidity) {
	if liquidityEntries == nil {
		liquidityEntries = k.liquidityEntries.Iterator(ctx, nil, denom).GetAll()
	}

	seen := false
	for index, liq := range liquidityEntries {
		if liq.Address == address && liq.PositionIndex == positionIndex {
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

func (k Keeper) GetAllLiquidity(ctx context.Context) (list []types.GenesisLiquidity) {
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		iterator := k.liquidityEntries.Iterator(ctx, nil, denom)

		for iterator.Valid() {
			liq := iterator.GetNext()
			list = append(list, types.GenesisLiquidity{
				Index:         liq.Index,
				Address:       liq.Address,
				Amount:        liq.Amount,
				PositionIndex: liq.GetPositionIndex(),
				Denom:         denom,
			})
		}

	}

	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Index < list[j].Index
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
				// If a kCoin is above parity, the protocol mints+sells and thereby adds already liquidity.
				if k.skipKCoin(ctx, denom) {
					continue
				}

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

func (k Keeper) skipKCoin(ctx context.Context, denom string) bool {
	if !k.DenomKeeper.IsKCoin(ctx, denom) {
		return false
	}

	aboveParity, err := k.isAboveParity(ctx, denom)
	if err != nil {
		k.Logger().Error(fmt.Sprintf("aboveParity: %v", err))
		return false
	}

	return aboveParity
}

func (k Keeper) GetDenomValue(ctx context.Context, denom string) (math.LegacyDec, error) {
	if denom == constants.BaseCurrency {
		liq := k.GetLiquiditySum(ctx, constants.BaseCurrency)
		return liq.ToLegacyDec(), nil
	}

	liq := k.GetFullLiquidityOther(ctx, denom)
	price, err := k.DenomKeeper.CalculatePrice(ctx, denom, constants.BaseCurrency)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return liq.Mul(price), nil
}

func (k Keeper) PrepareCutLiquidity(ctx *types.TradeContext) {
	liqFrom := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.TradeDenomGiving)
	liqTo := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(ctx.TradeDenomReceiving)
	liqBase := ctx.OrdersCaches.LiquidityPool.Get().AmountOf(constants.BaseCurrency).ToLegacyDec()

	if ctx.TradeDenomGiving != constants.BaseCurrency {
		cutLiquidity := k.createCutLiquidity(ctx, liqBase, liqFrom.ToLegacyDec(), ctx.TradeDenomGiving)

		switch ctx.TradeType {
		case types.TradeTypeSell:
			ctx.CutLiquidities.Step1 = &cutLiquidity
		case types.TradeTypeBuy:
			ctx.CutLiquidities.Step2 = &cutLiquidity
		default:
			panic("unknown trade type")
		}
	}

	if ctx.TradeDenomReceiving != constants.BaseCurrency {
		cutLiquidity := k.createCutLiquidity(ctx, liqBase, liqTo.ToLegacyDec(), ctx.TradeDenomReceiving)

		switch ctx.TradeType {
		case types.TradeTypeSell:
			ctx.CutLiquidities.Step2 = &cutLiquidity
		case types.TradeTypeBuy:
			ctx.CutLiquidities.Step1 = &cutLiquidity
		default:
			panic("unknown trade type")
		}
	}
}

func (k Keeper) createCutLiquidity(ctx context.Context, liqBase, liqOther math.LegacyDec, denom string) types.CutLiquidity {
	ratio, _ := k.DenomKeeper.GetRatio(ctx, denom)
	liqValue := liqOther.Quo(ratio.Ratio) // C

	tradeValue := math.LegacyMinDec(liqValue, liqBase)
	unusedLiqBase := liqBase.Sub(tradeValue)
	tradeValueOther := tradeValue.Mul(ratio.Ratio)
	unusedLiqOther := liqOther.Sub(tradeValueOther)

	cutLiquidity := types.CutLiquidity{}
	cutLiquidity.CutBase = math.LegacyMinDec(liqBase, tradeValue)
	cutLiquidity.CutOther = tradeValue.Mul(ratio.Ratio)

	if liqBase.LT(tradeValue) {
		cutLiquidity.VirtualBase = tradeValue.Sub(liqBase)
	}

	if tradeValue.LT(tradeValueOther) {
		cutLiquidity.VirtualOther = tradeValueOther.Sub(cutLiquidity.CutOther)
	}

	extraVirtualLiquidity := k.DenomKeeper.ExtraVirtualLiquidity(ctx, denom)
	if cutLiquidity.VirtualBase.IsNil() {
		cutLiquidity.VirtualBase = math.LegacyZeroDec()
	}

	if cutLiquidity.VirtualOther.IsNil() {
		cutLiquidity.VirtualOther = math.LegacyZeroDec()
	}

	extraVirtualLiquidityBase := extraVirtualLiquidity.ToLegacyDec().Quo(ratio.Ratio)
	cutLiquidity.VirtualBase = cutLiquidity.VirtualBase.Add(extraVirtualLiquidityBase)
	cutLiquidity.VirtualOther = cutLiquidity.VirtualOther.Add(extraVirtualLiquidity.ToLegacyDec())

	if cutLiquidity.VirtualBase.IsPositive() && unusedLiqBase.IsPositive() {
		usable := math.LegacyMinDec(cutLiquidity.VirtualBase, unusedLiqBase)
		cutLiquidity.CutBase = cutLiquidity.CutBase.Add(usable)
		cutLiquidity.VirtualBase = cutLiquidity.VirtualBase.Sub(usable)
	}

	if cutLiquidity.VirtualOther.IsPositive() && unusedLiqOther.IsPositive() {
		usable := math.LegacyMinDec(cutLiquidity.VirtualOther, unusedLiqOther)
		cutLiquidity.CutOther = cutLiquidity.CutOther.Add(usable)
		cutLiquidity.VirtualOther = cutLiquidity.VirtualOther.Sub(usable)
	}

	return cutLiquidity
}

func (k Keeper) CalcTradeBaseValue(ctx context.Context) (math.LegacyDec, error) {
	tradeBaseValueUSD := k.getTradeBaseValue(ctx)
	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("highest USD reference: %w", err)
	}

	tradeValueBase, err := k.DenomKeeper.GetValueInBase(ctx, referenceDenom, tradeBaseValueUSD)
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("convert to base: %w", err)
	}

	return tradeValueBase, nil
}

func (k Keeper) RemoveAllLiquidityForDenom(ctx context.Context, denom string) error {
	iterator := k.liquidityEntries.Iterator(ctx, nil, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()

		acc, _ := sdk.AccAddressFromBech32(liq.Address)
		coins := sdk.NewCoins(sdk.NewCoin(denom, liq.Amount))
		if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolLiquidity, acc, coins); err != nil {
			return fmt.Errorf("could not send coins from module to account: %w", err)
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
