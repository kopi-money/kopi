package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"github.com/cosmos/cosmos-sdk/cache"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/constants"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k Keeper) GetAllDenomCollaterals(ctx context.Context) (list []types.Collaterals) {
	for _, collateralDemom := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		var collaterals []types.Collateral
		iterator := k.CollateralIterator(ctx, collateralDemom.DexDenom)
		for iterator.Valid() {
			collateral := iterator.GetNext()
			collaterals = append(collaterals, collateral)
		}

		list = append(list, types.Collaterals{
			Denom:       collateralDemom.DexDenom,
			Collaterals: collaterals,
		})
	}

	return
}

func (k Keeper) CollateralIterator(ctx context.Context, denom string) cache.Iterator[string, types.Collateral] {
	rng := collections.NewPrefixedPairRange[string, string](denom)
	return k.collateral.Iterator(ctx, rng, denom)
}

func (k Keeper) LoadCollateral(ctx context.Context, denom, address string) (types.Collateral, bool) {
	return k.collateral.Get(ctx, denom, address)
}

func (k Keeper) SetCollateral(ctx context.Context, denom, address string, amount math.Int) {
	if amount.IsZero() {
		k.collateral.Remove(ctx, denom, address)
	} else {
		k.collateral.Set(ctx, denom, address, types.Collateral{Address: address, Amount: amount})
	}
}

func (k Keeper) removeCollateral(ctx context.Context, denom, address string) {
	k.collateral.Remove(ctx, denom, address)
}

func (k Keeper) getCollateralForDenomForAddress(ctx context.Context, denom, address string) (types.Collateral, bool) {
	return k.collateral.Get(ctx, denom, address)
}

func (k Keeper) GetCollateralForDenomForAddressWithDefault(ctx context.Context, denom, address string) math.Int {
	collateral, has := k.getCollateralForDenomForAddress(ctx, denom, address)
	if has {
		return collateral.Amount
	}

	return math.ZeroInt()
}

func (k Keeper) checkSupplyCap(ctx context.Context, denom string, amountToAdd math.Int) error {
	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolCollateral)
	supply := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()).AmountOf(denom)

	depositCap, err := k.DenomKeeper.GetDepositCap(ctx, denom)
	if err != nil {
		return err
	}

	if supply.Add(amountToAdd).GT(depositCap) {
		return types.ErrCollateralDepositLimitExceeded
	}

	return nil
}

// Calculates a user's collateral value in the base currency
func (k Keeper) calcCollateralValueBase(ctx context.Context, address string) (math.LegacyDec, error) {
	sum := math.LegacyZeroDec()

	for _, collateralDenom := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		amount := k.GetCollateralForDenomForAddressWithDefault(ctx, collateralDenom.DexDenom, address)
		if amount.LTE(math.ZeroInt()) {
			continue
		}

		value := amount.ToLegacyDec().Mul(collateralDenom.Ltv)
		valueBase, err := k.DenomKeeper.GetValueInBase(ctx, collateralDenom.DexDenom, value)
		if err != nil {
			return math.LegacyDec{}, fmt.Errorf("could not convert collateral amount to base: %w", err)
		}

		sum = sum.Add(valueBase)
	}

	return sum, nil
}

func (k Keeper) CalcWithdrawableCollateralAmount(ctx context.Context, address, denom string) (math.LegacyDec, error) {
	loanSumBase, err := k.getUserLoansSumBase(ctx, address)
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("could not get loan sum: %w", err)
	}

	// When there are no outstanding loans, the whole collateral amount can be withdrawn
	if loanSumBase.IsZero() {
		amount := k.GetCollateralForDenomForAddressWithDefault(ctx, denom, address)
		return amount.ToLegacyDec(), nil
	}

	collateralDenomLTV, err := k.DenomKeeper.GetLTV(ctx, denom)
	if err != nil {
		return math.LegacyDec{}, err
	}

	collateralSumBase, err := k.calcCollateralValueBase(ctx, address)
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("could not calculate collateral sum without: %w", err)
	}

	if loanSumBase.IsPositive() && loanSumBase.GTE(collateralSumBase) {
		return math.LegacyZeroDec(), nil
	}

	excessAmountBase := collateralSumBase.Sub(loanSumBase)
	excessAmount, err := k.DenomKeeper.GetValueIn(ctx, constants.BaseCurrency, denom, excessAmountBase)
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("could not convert back to denom currency: %w", err)
	}

	if !collateralDenomLTV.IsPositive() {
		return math.LegacyDec{}, fmt.Errorf("collateral denom ltv is not positive")
	}

	excessAmount = excessAmount.Quo(collateralDenomLTV) // C
	collateral := k.GetCollateralForDenomForAddressWithDefault(ctx, denom, address)
	excessAmount = math.LegacyMinDec(collateral.ToLegacyDec(), excessAmount)

	return excessAmount, nil
}
