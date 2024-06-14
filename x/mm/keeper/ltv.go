package keeper

import (
	"context"
	"cosmossdk.io/collections"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/utils"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

func (k Keeper) calculateBorrowableAmount(ctx context.Context, address, borrowDenom string) (math.LegacyDec, error) {
	collateralBaseValue, err := k.calculateCollateralBaseValue(ctx, address)
	if err != nil {
		return math.LegacyDec{}, err
	}

	loanBaseValue, err := k.calculateLoanBaseValue(ctx, address)
	if err != nil {
		return math.LegacyDec{}, err
	}

	borrowableBaseValue := collateralBaseValue.Sub(loanBaseValue)
	borrowableBaseValue = math.LegacyMaxDec(math.LegacyZeroDec(), borrowableBaseValue)

	borrowableValue, err := k.DexKeeper.GetValueIn(ctx, utils.BaseCurrency, borrowDenom, borrowableBaseValue)
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

func (k Keeper) calculateCollateralValueForDenom(ctx context.Context, collateralDenom *denomtypes.CollateralDenom, address string) (math.LegacyDec, error) {
	key := collections.Join(collateralDenom.Denom, address)
	collateral, found := k.collateral.Get(ctx, key)
	if !found {
		return math.LegacyZeroDec(), nil
	}

	price, err := k.DexKeeper.CalculatePrice(ctx, collateralDenom.Denom, utils.BaseCurrency)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return collateral.Amount.ToLegacyDec().Quo(price).Mul(collateralDenom.Ltv), nil
}

func (k Keeper) calculateLoanBaseValue(ctx context.Context, address string) (math.LegacyDec, error) {
	loanSum := math.LegacyZeroDec()

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		loanValue := k.GetLoanValue(ctx, cAsset.BaseDenom, address)

		loanValueBase, err := k.DexKeeper.GetValueInBase(ctx, cAsset.BaseDenom, loanValue)
		if err != nil {
			return math.LegacyDec{}, err
		}

		loanSum = loanSum.Add(loanValueBase)
	}

	return loanSum, nil
}
