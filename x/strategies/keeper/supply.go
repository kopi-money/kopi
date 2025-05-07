package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/strategies/types"
)

type CalculateValue []func() (math.LegacyDec, error)

func (cv CalculateValue) get() (math.LegacyDec, error) {
	value := math.LegacyZeroDec()
	for _, calcValue := range cv {
		v, err := calcValue()
		if err != nil {
			return math.LegacyDec{}, err
		}

		value = value.Add(v)
	}

	return value, nil
}

func (k Keeper) calculateNewAAssetAmount(ctx context.Context, denom string, addedAmount math.Int, calculateValue CalculateValue) (math.Int, error) {
	aAssetSupply := k.BankKeeper.GetSupply(ctx, denom).Amount
	if aAssetSupply.IsZero() {
		return addedAmount, nil
	}

	aAssetValue, err := calculateValue.get()
	if err != nil {
		return math.Int{}, fmt.Errorf("calculate aasset value: %w", err)
	}

	if !aAssetValue.IsPositive() {
		return math.Int{}, fmt.Errorf("%v aasset amount is not positive", denom)
	}

	valueShare := addedAmount.ToLegacyDec().Quo(aAssetValue) // C

	var newTokens math.Int
	if valueShare.Equal(math.LegacyOneDec()) {
		newTokens = addedAmount
	} else {
		newTokens = aAssetSupply.ToLegacyDec().Quo(math.LegacyOneDec().Sub(valueShare)).RoundInt().Sub(aAssetSupply) // C
	}

	return newTokens, nil
}

func (k Keeper) calculateRedemptionAmount(ctx context.Context, arbitrageDenom denomtypes.ArbitrageDenom, requestedAAssetAmount, available math.Int, calculateValue CalculateValue, allowIncomplete bool) (math.Int, math.Int, error) {
	redemptionValue, err := k.calculateRedemptionValue(ctx, arbitrageDenom, requestedAAssetAmount, calculateValue)
	if err != nil {
		return math.Int{}, math.Int{}, fmt.Errorf("calculate redemption value: %w", err)
	}

	if redemptionValue.GT(available) && !allowIncomplete {
		return math.Int{}, math.Int{}, types.ErrNotEnoughVault
	}

	if !redemptionValue.IsPositive() {
		return math.Int{}, math.Int{}, fmt.Errorf("redemption amount is not positive")
	}

	redeemAmount := math.MinInt(redemptionValue, available)
	requestedShare := redeemAmount.ToLegacyDec().Quo(redemptionValue.ToLegacyDec()) // C

	// how much of the given cAssets have been used
	usedTokens := requestedAAssetAmount.ToLegacyDec().Mul(requestedShare).TruncateInt()
	return redeemAmount, usedTokens, nil
}

func (k Keeper) calculateRedemptionValue(ctx context.Context, arbitrageDenom denomtypes.ArbitrageDenom, requestedAAssetAmount math.Int, calculateValue CalculateValue) (math.Int, error) {
	if requestedAAssetAmount.IsZero() {
		return math.ZeroInt(), nil
	}

	// First it is calculated how much of the total share the withdrawal request's given tokens represent.
	assetSupply := math.LegacyNewDecFromInt(k.BankKeeper.GetSupply(ctx, arbitrageDenom.DexDenom).Amount)
	assetValue, err := calculateValue.get()
	if err != nil {
		return math.Int{}, fmt.Errorf("calculate aAsset value: %w", err)
	}

	if !assetSupply.IsPositive() {
		return math.Int{}, fmt.Errorf("assetSupply amount is not positive")
	}

	// how much value of all cAssetValue does the redemption request represent
	redemptionShare := requestedAAssetAmount.ToLegacyDec().Quo(assetSupply) // C
	redemptionValue := assetValue.Mul(redemptionShare).TruncateInt()
	return redemptionValue, nil
}
