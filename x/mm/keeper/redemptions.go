package keeper

import (
	"context"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"fmt"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/mm/types"
	"sort"
)

func (k Keeper) SetRedemptions(ctx context.Context, redemptions types.DenomRedemption) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixRedemptions))
	b := k.cdc.MustMarshal(&redemptions)
	store.Set(types.KeyDenom(redemptions.Denom), b)
}

// GetAllDenomRedemptions returns all deposits
func (k Keeper) GetAllDenomRedemptions(ctx context.Context) (list []types.DenomRedemption) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixRedemptions))

	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.DenomRedemption
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// SetRedemption set a specific withdrawals in the store
func (k Keeper) SetRedemption(ctx context.Context, denom string, withdrawal types.Redemption) {
	if withdrawal.Amount.LTE(math.ZeroInt()) {
		k.RemoveRedemption(ctx, denom, withdrawal.Address)
		return
	}

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixRedemptions))

	b := k.cdc.MustMarshal(&withdrawal)
	store.Set(types.KeyDenomAddress(denom, withdrawal.Address), b)
}

// GetRedemption returns a withdrawals from its id
func (k Keeper) GetRedemption(ctx context.Context, denom, address string) (val types.Redemption, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixRedemptions))
	b := store.Get(types.KeyDenomAddress(denom, address))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveRedemption removes a withdrawals from the store
func (k Keeper) RemoveRedemption(ctx context.Context, denom, address string) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixRedemptions))
	store.Delete(types.KeyDenomAddress(denom, address))
}

// GetRedemptions returns all withdrawals for a given denom
func (k Keeper) GetRedemptions(ctx context.Context, denom string) (list []types.Redemption) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixRedemptions))

	iterator := storetypes.KVStorePrefixIterator(store, types.KeyDenom(denom))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Redemption
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) GetRedemptionSum(ctx context.Context, denom string) math.Int {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixRedemptions))

	iterator := storetypes.KVStorePrefixIterator(store, types.KeyDenom(denom))
	defer iterator.Close()

	sum := math.ZeroInt()
	for ; iterator.Valid(); iterator.Next() {
		var val types.Redemption
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		sum = sum.Add(val.Amount)
	}

	return sum
}

func (k Keeper) HandleRedemptions(ctx context.Context, eventManager sdk.EventManagerI) error {
	for _, CAsset := range k.DenomKeeper.GetCAssets(ctx) {
		if err := k.handleRedemptionsForCAsset(ctx, eventManager, CAsset); err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not handle withdrawals for CAsset %v", CAsset.Name))
		}
	}

	return nil
}

func (k Keeper) handleRedemptionsForCAsset(ctx context.Context, eventManager sdk.EventManagerI, cAsset *denomtypes.CAsset) error {
	redemptions := k.GetRedemptions(ctx, cAsset.Name)
	if len(redemptions) == 0 {
		return nil
	}

	moduleAccount := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
	found, coin := k.BankKeeper.SpendableCoins(ctx, moduleAccount.GetAddress()).Find(cAsset.BaseDenom)

	k.logger.Info(fmt.Sprintf("Redemptions %v: %v, funds: %v", cAsset.Name, len(redemptions), coin.Amount.String()))
	if !found || coin.Amount.Equal(math.ZeroInt()) {
		return nil
	}

	sort.SliceStable(redemptions, func(i, j int) bool {
		if redemptions[i].Fee.Equal(redemptions[j].Fee) {
			return redemptions[i].AddedAt < redemptions[j].AddedAt
		}

		return redemptions[i].Fee.GT(redemptions[j].Fee)
	})

	available := math.LegacyNewDecFromInt(coin.Amount)
	for available.GT(math.LegacyZeroDec()) && len(redemptions) > 0 {
		redemption := redemptions[0]
		redemptions = redemptions[1:]

		sentAmount, err := k.handleSingleRedemption(ctx, eventManager, cAsset, redemption, available)
		if err != nil {
			return err
		}

		available = available.Sub(sentAmount)
	}

	return nil
}

func (k Keeper) handleSingleRedemption(ctx context.Context, eventManager sdk.EventManagerI, cAsset *denomtypes.CAsset, entry types.Redemption, available math.LegacyDec) (math.LegacyDec, error) {
	grossRedemptionAmountBase, redemptionAmountCAsset := k.CalculateRedemptionAmount(ctx, cAsset, entry.Amount.ToLegacyDec(), available)

	// Update the entry and process the payout
	entry.Amount = entry.Amount.Sub(redemptionAmountCAsset.RoundInt())
	k.SetRedemption(ctx, cAsset.Name, entry)

	// subtract the priority cost set by the user to be handled with higher priority
	feeCost := grossRedemptionAmountBase.Mul(entry.Fee)
	redemptionAmount := grossRedemptionAmountBase.Sub(feeCost)
	if err := k.handleRedemptionFee(ctx, eventManager, cAsset, feeCost); err != nil {
		return math.LegacyDec{}, err
	}

	// send redeemed coins (sub fee) to user
	acc, _ := sdk.AccAddressFromBech32(entry.Address)
	coins := sdk.NewCoins(sdk.NewCoin(cAsset.BaseDenom, redemptionAmount.TruncateInt()))
	if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolVault, acc, coins); err != nil {
		return math.LegacyDec{}, err
	}

	// Burn the CAsset tokens that have been redeemed
	coins = sdk.NewCoins(sdk.NewCoin(cAsset.Name, redemptionAmountCAsset.TruncateInt()))
	if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolRedemption, types.ModuleName, coins); err != nil {
		return math.LegacyDec{}, err
	}

	if err := k.BankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
		return math.LegacyDec{}, err
	}

	eventManager.EmitEvent(
		sdk.NewEvent("redemption_request_executed",
			sdk.Attribute{Key: "address", Value: entry.Address},
			sdk.Attribute{Key: "denom", Value: cAsset.Name},
			sdk.Attribute{Key: "redeemed", Value: redemptionAmountCAsset.String()},
			sdk.Attribute{Key: "received", Value: redemptionAmount.String()},
		),
	)

	return grossRedemptionAmountBase, nil
}

func (k Keeper) CalculateRedemptionAmount(ctx context.Context, cAsset *denomtypes.CAsset, requestedCAssetAmount, available math.LegacyDec) (math.LegacyDec, math.LegacyDec) {
	// First it is calculated how much of the total share the withdrawal request's given tokens represent.
	cAssetSupply := math.LegacyNewDecFromInt(k.BankKeeper.GetSupply(ctx, cAsset.Name).Amount)
	cAssetValue := k.calculateCAssetValue(ctx, cAsset)

	// how much value of all cAssetValue does the redemption request represent
	redemptionShare := requestedCAssetAmount.Quo(cAssetSupply)
	redemptionValue := cAssetValue.Mul(redemptionShare)

	// how much of what is requested can be paid out
	redeemAmount := math.LegacyMinDec(redemptionValue, available)

	// the share of what is paid out in relation to what has been requested
	requestedShare := redeemAmount.Quo(redemptionValue)

	// how much of the given cAssets have been used
	usedCAssets := requestedCAssetAmount.Mul(requestedShare)

	return redeemAmount, usedCAssets
}

func (k Keeper) handleRedemptionFee(ctx context.Context, eventManager sdk.EventManagerI, cAsset *denomtypes.CAsset, amount math.LegacyDec) error {
	if amount.LTE(math.LegacyZeroDec()) {
		return nil
	}

	protocolShare := k.GetParams(ctx).ProtocolShare
	protocolAmount := protocolShare.Mul(amount)

	coins := sdk.NewCoins(sdk.NewCoin(cAsset.BaseDenom, protocolAmount.TruncateInt()))
	if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolVault, dextypes.PoolReserve, coins); err != nil {
		return err
	}

	eventManager.EmitEvent(
		sdk.NewEvent("redemption_fee_protocol",
			sdk.Attribute{Key: "denom", Value: cAsset.BaseDenom},
			sdk.Attribute{Key: "fee", Value: protocolAmount.TruncateInt().String()},
		),
	)

	return nil
}
