package keeper

import (
	"context"
	"cosmossdk.io/collections"
	"fmt"
	"github.com/kopi-money/kopi/cache"
	"sort"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k Keeper) LoadRedemptionRequest(ctx context.Context, denom, address string) (types.Redemption, bool) {
	return k.redemptions.Get(ctx, collections.Join(denom, address))
}

func (k Keeper) RedemptionIterator(ctx context.Context, denom string) cache.Iterator[collections.Pair[string, string], types.Redemption] {
	rng := collections.NewPrefixedPairRange[string, string](denom)
	keyPrefix := func(key collections.Pair[string, string]) bool {
		return key.K1() == denom
	}
	return k.redemptions.Iterator(ctx, rng, keyPrefix)
}

func (k Keeper) GetDenomRedemptions(ctx context.Context) (list []types.DenomRedemption) {
	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		var redemptions []*types.Redemption

		iterator := k.RedemptionIterator(ctx, cAsset.BaseDenom)
		for iterator.Valid() {
			redemption := iterator.GetNext()
			redemptions = append(redemptions, &redemption)
		}

		list = append(list, types.DenomRedemption{
			Denom:       cAsset.BaseDenom,
			Redemptions: redemptions,
		})
	}

	return
}

// SetRedemption set a specific withdrawals in the store
func (k Keeper) updateRedemption(ctx context.Context, denom string, redemption types.Redemption) error {
	if redemption.Amount.LTE(math.ZeroInt()) {
		k.removeRedemption(ctx, denom, redemption.Address)
		return nil
	} else {
		if err := k.SetRedemption(ctx, denom, redemption); err != nil {
			return errors.Wrap(err, "could not set redemption")
		}

		return nil
	}
}

func (k Keeper) SetRedemption(ctx context.Context, denom string, redemption types.Redemption) error {
	if redemption.Address == "" {
		return fmt.Errorf("redemption with empty address given")
	}
	if redemption.Amount.IsNil() {
		return fmt.Errorf("redemption with nil amount given")
	}

	key := collections.Join(denom, redemption.Address)
	k.redemptions.Set(ctx, key, redemption)
	return nil
}

func (k Keeper) removeRedemption(ctx context.Context, denom, address string) {
	key := collections.Join(denom, address)
	k.redemptions.Remove(ctx, key)
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
	redemptions := k.RedemptionIterator(ctx, cAsset.BaseDenom).GetAll()
	if len(redemptions) == 0 {
		return nil
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	k.logger.Info(fmt.Sprintf("(%v / %v): %v", sdkCtx.BlockHeight(), cAsset.BaseDenom, len(redemptions)))

	moduleAccount := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
	found, coin := k.BankKeeper.SpendableCoins(ctx, moduleAccount.GetAddress()).Find(cAsset.BaseDenom)
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

		if redemption.Amount.IsNil() {
			k.logger.Error(fmt.Sprintf("RRN (%v / %v)", redemption.Address, cAsset.BaseDenom))
			continue
		}

		sentAmount, err := k.handleSingleRedemption(ctx, eventManager, cAsset, redemption, available)
		if err != nil {
			return err
		}

		available = available.Sub(sentAmount)
	}

	return nil
}

func (k Keeper) handleSingleRedemption(ctx context.Context, eventManager sdk.EventManagerI, cAsset *denomtypes.CAsset, entry types.Redemption, available math.LegacyDec) (math.LegacyDec, error) {
	grossRedemptionAmountBase, redemptionAmountCAsset := k.CalculateAvailableRedemptionAmount(ctx, cAsset, entry.Amount.ToLegacyDec(), available)

	// Update the entry and process the payout
	entry.Amount = entry.Amount.Sub(redemptionAmountCAsset.RoundInt())
	if err := k.updateRedemption(ctx, cAsset.BaseDenom, entry); err != nil {
		return math.LegacyDec{}, errors.Wrap(err, "could not update redemption request")
	}

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
			sdk.Attribute{Key: "denom", Value: cAsset.BaseDenom},
			sdk.Attribute{Key: "redeemed", Value: redemptionAmountCAsset.String()},
			sdk.Attribute{Key: "received", Value: redemptionAmount.String()},
		),
	)

	return grossRedemptionAmountBase, nil
}

func (k Keeper) CalculateRedemptionAmount(ctx context.Context, cAsset *denomtypes.CAsset, requestedCAssetAmount math.LegacyDec) math.LegacyDec {
	if requestedCAssetAmount.Equal(math.LegacyZeroDec()) {
		return math.LegacyZeroDec()
	}

	// First it is calculated how much of the total share the withdrawal request's given tokens represent.
	cAssetSupply := math.LegacyNewDecFromInt(k.BankKeeper.GetSupply(ctx, cAsset.Name).Amount)
	cAssetValue := k.calculateCAssetValue(ctx, cAsset)

	// how much value of all cAssetValue does the redemption request represent
	redemptionShare := requestedCAssetAmount.Quo(cAssetSupply)
	redemptionValue := cAssetValue.Mul(redemptionShare)

	return redemptionValue
}

func (k Keeper) CalculateAvailableRedemptionAmount(ctx context.Context, cAsset *denomtypes.CAsset, requestedCAssetAmount, available math.LegacyDec) (math.LegacyDec, math.LegacyDec) {
	redemptionValue := k.CalculateRedemptionAmount(ctx, cAsset, requestedCAssetAmount)

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

func compareRedemptions(r1, r2 types.Redemption) bool {
	return r1.Amount.Equal(r2.Amount) &&
		r1.Fee.Equal(r2.Fee) &&
		r1.Address == r2.Address &&
		r1.AddedAt == r2.AddedAt
}
