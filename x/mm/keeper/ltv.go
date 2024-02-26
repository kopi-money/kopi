package keeper

import (
	"context"
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
	borrowableValue, err := k.DexKeeper.GetValueIn(ctx, utils.BaseCurrency, borrowDenom, borrowableBaseValue.TruncateInt())
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
	collateral, found := k.GetCollateral(ctx, collateralDenom.Denom, address)
	if !found {
		return math.LegacyZeroDec(), nil
	}

	price, err := k.DexKeeper.CalculatePrice(ctx, collateralDenom.Denom, utils.BaseCurrency)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return collateral.Amount.ToLegacyDec().Mul(price).Mul(collateralDenom.Ltv), nil
}

func (k Keeper) calculateLoanBaseValue(ctx context.Context, address string) (math.LegacyDec, error) {
	loanSum := math.LegacyZeroDec()

	for _, CAsset := range k.DenomKeeper.GetCAssets(ctx) {
		loan, found := k.GetLoan(ctx, CAsset.BaseDenom, address)
		if !found {
			continue
		}

		price, err := k.DexKeeper.CalculatePrice(ctx, CAsset.BaseDenom, utils.BaseCurrency)
		if err != nil {
			return math.LegacyDec{}, err
		}

		loanSum = loanSum.Add(loan.Amount.Mul(price))
	}

	return loanSum, nil
}
