package keeper

import (
	"context"
	"fmt"
	"sort"

	"cosmossdk.io/collections"
	"github.com/cosmos/cosmos-sdk/cache"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

const minimumPayoutAmount = 100_000

func (k Keeper) LoadRedemptionRequest(ctx context.Context, denom, address string) (types.Redemption, bool) {
	return k.redemptions.Get(ctx, denom, address)
}

func (k Keeper) RedemptionIterator(ctx context.Context, denom string) cache.Iterator[string, types.Redemption] {
	rng := collections.NewPrefixedPairRange[string, string](denom)
	return k.redemptions.Iterator(ctx, rng, denom)
}

func (k Keeper) GetRedemptionSum(ctx context.Context, denom string) math.Int {
	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolRedemption)
	return k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()).AmountOf(denom)
}

func (k Keeper) GetDenomRedemptions(ctx context.Context) (list []types.DenomRedemption) {
	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		var redemptions []types.Redemption

		iterator := k.RedemptionIterator(ctx, cAsset.BaseDexDenom)
		for iterator.Valid() {
			redemption := iterator.GetNext()
			redemptions = append(redemptions, redemption)
		}

		list = append(list, types.DenomRedemption{
			Denom:       cAsset.BaseDexDenom,
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
			return fmt.Errorf("could not set redemption: %w", err)
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

	k.redemptions.Set(ctx, denom, redemption.Address, redemption)
	return nil
}

func (k Keeper) removeRedemption(ctx context.Context, denom, address string) {
	k.redemptions.Remove(ctx, denom, address)
}

func (k Keeper) HandleRedemptions(ctx context.Context) error {
	for _, CAsset := range k.DenomKeeper.GetCAssets(ctx) {
		if err := k.handleRedemptionsForCAsset(ctx, CAsset); err != nil {
			return fmt.Errorf("could not handle withdrawals for CAsset %v: %w", CAsset.DexDenom, err)
		}
	}

	return nil
}

func (k Keeper) handleRedemptionsForCAsset(ctx context.Context, cAsset denomtypes.CAsset) error {
	redemptions := k.RedemptionIterator(ctx, cAsset.BaseDexDenom).GetAll()
	if len(redemptions) == 0 {
		return nil
	}

	moduleAccount := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
	found, coin := k.BankKeeper.SpendableCoins(ctx, moduleAccount.GetAddress()).Find(cAsset.BaseDexDenom)
	if !found || coin.Amount.IsZero() {
		return nil
	}

	sort.SliceStable(redemptions, func(i, j int) bool {
		if redemptions[i].Fee.Equal(redemptions[j].Fee) {
			return redemptions[i].AddedAt < redemptions[j].AddedAt
		}

		return redemptions[i].Fee.GT(redemptions[j].Fee)
	})

	available := math.LegacyNewDecFromInt(coin.Amount)
	for available.IsPositive() && len(redemptions) > 0 {
		redemption := redemptions[0]
		redemptions = redemptions[1:]

		if redemption.Amount.IsNil() {
			continue
		}

		sentAmount, err := k.handleSingleRedemption(ctx, cAsset, redemption, available)
		if err != nil {
			return err
		}

		available = available.Sub(sentAmount)
	}

	return nil
}

func (k Keeper) handleSingleRedemption(ctx context.Context, cAsset denomtypes.CAsset, entry types.Redemption, available math.LegacyDec) (math.LegacyDec, error) {
	grossRedemptionAmountBase, redemptionAmountCAsset, payoutshare := k.CalculateAvailableRedemptionAmount(ctx, cAsset, entry.Amount.ToLegacyDec(), available)
	if grossRedemptionAmountBase.IsZero() {
		return math.LegacyZeroDec(), nil
	}

	// Update the entry and process the payout
	entry.Amount = entry.Amount.Sub(redemptionAmountCAsset.RoundInt())
	if err := k.updateRedemption(ctx, cAsset.BaseDexDenom, entry); err != nil {
		return math.LegacyDec{}, fmt.Errorf("could not update redemption request: %w", err)
	}

	// subtract the priority cost set by the user to be handled with higher priority
	feeCost := grossRedemptionAmountBase.Mul(entry.Fee)
	redemptionAmount := grossRedemptionAmountBase.Sub(feeCost)
	if err := k.handleRedemptionFee(ctx, cAsset, feeCost); err != nil {
		return math.LegacyDec{}, err
	}

	payoutAmount := redemptionAmount.TruncateInt()
	// If the payout amount is not for the full requested amount, only proceed if the payout amount is above the minimum
	// payout. This is to prevent sending dust.
	if payoutshare.LT(math.LegacyOneDec()) && payoutAmount.LT(math.NewInt(minimumPayoutAmount)) {
		return math.LegacyZeroDec(), nil
	}

	// send redeemed coins (sub fee) to user
	acc, _ := sdk.AccAddressFromBech32(entry.Address)
	coins := sdk.NewCoins(sdk.NewCoin(cAsset.BaseDexDenom, redemptionAmount.TruncateInt()))
	if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolVault, acc, coins); err != nil {
		return math.LegacyDec{}, err
	}

	// Burn the CAsset tokens that have been redeemed
	coins = sdk.NewCoins(sdk.NewCoin(cAsset.DexDenom, redemptionAmountCAsset.TruncateInt()))
	if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolRedemption, types.ModuleName, coins); err != nil {
		return math.LegacyDec{}, err
	}

	if err := k.BankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
		return math.LegacyDec{}, err
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("redemption_request_executed",
			sdk.Attribute{Key: "address", Value: entry.Address},
			sdk.Attribute{Key: "denom", Value: cAsset.BaseDexDenom},
			sdk.Attribute{Key: "redeemed", Value: redemptionAmountCAsset.String()},
			sdk.Attribute{Key: "received", Value: redemptionAmount.String()},
		),
	)

	return grossRedemptionAmountBase, nil
}

func (k Keeper) CalculateRedemptionAmount(ctx context.Context, cAsset denomtypes.CAsset, requestedCAssetAmount math.LegacyDec) math.LegacyDec {
	if requestedCAssetAmount.IsZero() {
		return math.LegacyZeroDec()
	}

	// First it is calculated how much of the total share the withdrawal request's given tokens represent.
	cAssetSupply := k.BankKeeper.GetSupply(ctx, cAsset.DexDenom).Amount.ToLegacyDec()
	if cAssetSupply.IsZero() {
		return math.LegacyZeroDec()
	}

	cAssetValue := k.CalculateCAssetValue(ctx, cAsset)

	// how much value of all cAssetValue does the redemption request represent
	redemptionShare := requestedCAssetAmount.Quo(cAssetSupply) // C
	redemptionValue := cAssetValue.Mul(redemptionShare)

	return redemptionValue
}

func (k Keeper) CalculateAvailableRedemptionAmount(ctx context.Context, cAsset denomtypes.CAsset, requestedCAssetAmount, available math.LegacyDec) (math.LegacyDec, math.LegacyDec, math.LegacyDec) {
	redemptionValue := k.CalculateRedemptionAmount(ctx, cAsset, requestedCAssetAmount)
	if redemptionValue.IsZero() {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()
	}

	// how much of what is requested can be paid out
	redeemAmount := math.LegacyMinDec(redemptionValue, available)
	if redeemAmount.IsZero() {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()
	}

	// the share of what is paid out in relation to what has been requested
	requestedShare := redeemAmount.Quo(redemptionValue) // C

	// how much of the given cAssets have been used
	usedCAssets := requestedCAssetAmount.Mul(requestedShare)
	return redeemAmount, usedCAssets, requestedShare
}

func (k Keeper) handleRedemptionFee(ctx context.Context, cAsset denomtypes.CAsset, amount math.LegacyDec) error {
	protocolShare := k.GetParams(ctx).ProtocolShare
	protocolAmount := protocolShare.Mul(amount)
	protocolAmountInt := protocolAmount.TruncateInt()

	if !protocolAmountInt.IsPositive() {
		return nil
	}

	coins := sdk.NewCoins(sdk.NewCoin(cAsset.BaseDexDenom, protocolAmountInt))
	if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolVault, dextypes.PoolReserve, coins); err != nil {
		return err
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("redemption_fee_protocol",
			sdk.Attribute{Key: "denom", Value: cAsset.BaseDexDenom},
			sdk.Attribute{Key: "fee", Value: protocolAmount.TruncateInt().String()},
		),
	)

	return nil
}
