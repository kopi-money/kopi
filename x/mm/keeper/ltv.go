package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/constants"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

func (k Keeper) CalculateBorrowableAmount(ctx context.Context, address, borrowDenom string) (math.LegacyDec, error) {
	collateralBaseValue, err := k.calculateCollateralBaseValue(ctx, address)
	if err != nil {
		return math.LegacyDec{}, err
	}

	loanBaseValue, _, err := k.calculateLoanBaseValue(ctx, address)
	if err != nil {
		return math.LegacyDec{}, err
	}

	borrowableBaseValue := collateralBaseValue.Sub(loanBaseValue)
	borrowableBaseValue = math.LegacyMaxDec(math.LegacyZeroDec(), borrowableBaseValue)

	borrowableValue, err := k.DenomKeeper.GetValueIn(ctx, constants.BaseCurrency, borrowDenom, borrowableBaseValue)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return borrowableValue, nil
}

func (k Keeper) calculateCollateralBaseValue(ctx context.Context, address string) (math.LegacyDec, error) {
	borrowableAmount := math.LegacyZeroDec()
	for _, collateral := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		amount, err := k.calculateCollateralValueForDenom(ctx, collateral, address)
		if err != nil {
			return math.LegacyDec{}, err
		}

		borrowableAmount = borrowableAmount.Add(amount)
	}

	return borrowableAmount, nil
}

func (k Keeper) calculateCollateralValueForDenom(ctx context.Context, collateralDenom denomtypes.CollateralDenom, address string) (math.LegacyDec, error) {
	collateral, found := k.collateral.Get(ctx, collateralDenom.DexDenom, address)
	if !found || collateral.Amount.IsZero() {
		return math.LegacyZeroDec(), nil
	}

	amountBase, err := k.DenomKeeper.GetValueInBase(ctx, collateralDenom.DexDenom, collateral.Amount.ToLegacyDec())
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("unable to calculate collateral value for %s: %w", collateralDenom.DexDenom, err)
	}

	return amountBase.Mul(collateralDenom.Ltv), nil
}

func (k Keeper) calculateLoanBaseValue(ctx context.Context, address string) (math.LegacyDec, map[string]math.LegacyDec, error) {
	loanSum := math.LegacyZeroDec()
	loanValues := make(map[string]math.LegacyDec)

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		loanValue := k.GetLoanValue(ctx, cAsset.BaseDexDenom, address)
		loanValues[cAsset.BaseDexDenom] = loanValue

		loanValueBase, err := k.DenomKeeper.GetValueInBase(ctx, cAsset.BaseDexDenom, loanValue)
		if err != nil {
			return math.LegacyDec{}, loanValues, err
		}

		loanSum = loanSum.Add(loanValueBase)
	}

	return loanSum, loanValues, nil
}
