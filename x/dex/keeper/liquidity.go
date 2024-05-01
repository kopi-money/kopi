package keeper

import (
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/dex/types"
	"github.com/pkg/errors"

	"cosmossdk.io/store/prefix"
)

// SetLiquidity sets a specific liquidity in the store from its index. When the index is zero, i.e. it's a new entry,
// the NextIndex is increased and updated as well.
func (k Keeper) SetLiquidity(ctx context.Context, liquidity types.Liquidity, change math.Int) {
	if liquidity.Index == 0 {
		nextIndex, _ := k.GetLiquidityNextIndex(ctx)
		nextIndex.Next += 1
		k.SetLiquidityNextIndex(ctx, nextIndex)

		liquidity.Index = nextIndex.Next
	}

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLiquidity))
	b := k.cdc.MustMarshal(&liquidity)
	store.Set(types.KeyDenomIndex(liquidity.Denom, liquidity.Index), b)

	k.updateLiquiditySum(ctx, liquidity.Denom, change)
}

func (k Keeper) updateLiquiditySum(ctx context.Context, denom string, change math.Int) {
	liqSum, found := k.GetLiquiditySum(ctx, denom)
	if !found {
		liqSum = math.ZeroInt()
	}
	liqSum = liqSum.Add(change)
	//fmt.Println(fmt.Sprintf("UL, %v: %v", denom, change.String()))
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
	liqBase, _ := k.GetLiquiditySum(ctx, utils.BaseCurrency)

	if liqBase.Equal(math.ZeroInt()) && denom != utils.BaseCurrency {
		return types.ErrBaseLiqEmpty
	}

	liq := k.addLiquidity(ctx, denom, address.String(), amount)

	// When changing actual liquidity, the virtual liquidity has to be adjusted to keep the ratio.
	if denom != utils.BaseCurrency {
		k.updatePair(ctx, denom)
	} else {
		k.updatePairs(ctx)
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

func (k Keeper) addLiquidity(ctx context.Context, denom, address string, amount math.Int) types.Liquidity {
	iterator := k.LiquidityIterator(ctx, denom)

	seen := false
	for ; iterator.Valid(); iterator.Next() {
		var liq types.Liquidity
		k.cdc.MustUnmarshal(iterator.Value(), &liq)

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
			return liq
		}
	}

	liq := types.Liquidity{Denom: denom, Address: address, Amount: amount}
	k.SetLiquidity(ctx, liq, amount)
	return liq
}

func (k Keeper) GetLiquidityEntries(ctx context.Context, denom string) []types.Liquidity {
	var entries []types.Liquidity

	iterator := k.LiquidityIterator(ctx, denom)
	for ; iterator.Valid(); iterator.Next() {
		var liq types.Liquidity
		k.cdc.MustUnmarshal(iterator.Value(), &liq)
		entries = append(entries, liq)
	}

	return entries
}

func (k Keeper) GetLiquidityByAddress(ctx context.Context, denom, address string) math.Int {
	sum := math.ZeroInt()

	iterator := k.LiquidityIterator(ctx, denom)
	for ; iterator.Valid(); iterator.Next() {
		var liq types.Liquidity
		k.cdc.MustUnmarshal(iterator.Value(), &liq)

		if liq.Address == address {
			sum = sum.Add(liq.Amount)
		}
	}

	return sum
}

func (k Keeper) GetLiquidityEntriesByAddress(ctx context.Context, denom, address string) int {
	num := 0

	iterator := k.LiquidityIterator(ctx, denom)
	for ; iterator.Valid(); iterator.Next() {
		var liq types.Liquidity
		k.cdc.MustUnmarshal(iterator.Value(), &liq)

		if liq.Address == address {
			num++
		}
	}

	return num
}

func (k Keeper) GetAllLiquidity(ctx context.Context) (list []types.Liquidity) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLiquidity))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	for ; iterator.Valid(); iterator.Next() {
		liq := k.LiquidityUnmarshal(iterator.Value())
		list = append(list, liq)
	}

	return
}

// RemoveLiquidity removes a liquidity from the store
func (k Keeper) RemoveLiquidity(ctx context.Context, denom string, index uint64, change math.Int) {
	if change.LT(math.ZeroInt()) {
		panic("cannot be negative amount, positive amount is assumed")
	}

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLiquidity))
	store.Delete(types.KeyDenomIndex(denom, index))
	k.updateLiquiditySum(ctx, denom, change.Neg())
}

func (k Keeper) LiquidityStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixLiquidity))
}

func (k Keeper) LiquidityIterator(ctx context.Context, denom string) storetypes.Iterator {
	store := k.LiquidityStore(ctx)
	return storetypes.KVStorePrefixIterator(store, types.KeyString(denom))
}

func (k Keeper) LiquidityUnmarshal(raw []byte) types.Liquidity {
	var liq types.Liquidity
	k.cdc.MustUnmarshal(raw, &liq)
	return liq
}

// UpdateVirtualLiquidities updates the virtual liquidity for each pair. This method is called at the end of each block.
// The virtual liquidty is only updated when there is no actual liquidity for that denom. When the virtual liquidity is
// 0, it means the pair probably just have been created and will be set to the initial virtual amount. If the amount
// of actual liquidity is zero and the amount of virtual liquidity is not zero, we slowly decrease the amount of virtual
// liquidity to increase that denom's price.
func (k Keeper) UpdateVirtualLiquidities(ctx context.Context) {
	decay := k.GetParams(ctx).VirtualLiquidityDecay
	liqBase, _ := k.GetLiquiditySum(ctx, utils.BaseCurrency)

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		if denom != utils.BaseCurrency {
			liq, _ := k.GetLiquiditySum(ctx, denom)
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
					k.Logger().Info(fmt.Sprintf("Lowering virtual liquidity for %v: %v", denom, pair.VirtualOther))
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
		liq, _ := k.GetLiquiditySum(ctx, utils.BaseCurrency)
		return liq.ToLegacyDec(), nil
	}

	liq := k.GetFullLiquidityOther(ctx, denom)
	price, err := k.CalculatePrice(ctx, denom, utils.BaseCurrency)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return liq.Mul(price), nil
}
