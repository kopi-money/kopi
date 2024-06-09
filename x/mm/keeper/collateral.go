package keeper

import (
	"context"
	"cosmossdk.io/collections"
	"github.com/kopi-money/kopi/cache"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k Keeper) GetAllDenomCollaterals(ctx context.Context) (list []types.Collaterals) {
	for _, collateralDemom := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		var collaterals []*types.Collateral
		iterator := k.CollateralIterator(ctx, collateralDemom.Denom)
		for iterator.Valid() {
			collateral := iterator.GetNext()
			collaterals = append(collaterals, &collateral)
		}

		list = append(list, types.Collaterals{
			Denom:       collateralDemom.Denom,
			Collaterals: collaterals,
		})
	}

	return
}

func (k Keeper) CollateralIterator(ctx context.Context, denom string) *cache.Iterator[collections.Pair[string, string], types.Collateral] {
	extraFilters := []cache.Filter[collections.Pair[string, string]]{
		func(key collections.Pair[string, string]) bool {
			return key.K1() == denom
		},
	}
	return k.collateral.Iterator(ctx, extraFilters...)
}

func (k Keeper) LoadCollateral(ctx context.Context, denom, address string) (types.Collateral, bool) {
	return k.collateral.Get(ctx, collections.Join(denom, address))
}

func (k Keeper) SetCollateral(ctx context.Context, denom, address string, amount, change math.Int) {
	key := collections.Join(denom, address)
	k.collateral.Set(ctx, key, types.Collateral{Address: address, Amount: amount})
	k.updateCollateralSum(ctx, denom, change)
}

func (k Keeper) removeCollateral(ctx context.Context, denom, address string, change math.Int) {
	key := collections.Join(denom, address)
	k.collateral.Remove(ctx, key)
	k.updateCollateralSum(ctx, denom, change)
}

func (k Keeper) updateCollateralSum(ctx context.Context, denom string, change math.Int) {
	collateralSum := k.getCollateralSum(ctx, denom).Add(change)
	k.setCollateralSum(ctx, denom, collateralSum)
}

func (k Keeper) getCollateralForDenomForAddress(ctx context.Context, denom, address string) (types.Collateral, bool) {
	key := collections.Join(denom, address)
	return k.collateral.Get(ctx, key)
}

func (k Keeper) getCollateralForDenomForAddressWithDefault(ctx context.Context, denom, address string) math.Int {
	collateral, has := k.getCollateralForDenomForAddress(ctx, denom, address)
	if has {
		return collateral.Amount
	}

	return math.ZeroInt()
}

func (k Keeper) checkSupplyCap(ctx context.Context, denom string, amountToAdd math.Int) error {
	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolCollateral)
	found, supply := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()).Find(denom)
	if !found {
		return nil
	}

	depositCap, err := k.DenomKeeper.GetDepositCap(ctx, denom)
	if err != nil {
		return err
	}

	if supply.Amount.Add(amountToAdd).GT(depositCap) {
		return types.ErrDepositLimitExceeded
	}

	return nil
}

// Calculates a user's collateral value in the base currency
func (k Keeper) calcCollateralValueBase(ctx context.Context, address string) (math.LegacyDec, error) {
	sum := math.LegacyZeroDec()

	for _, collateralDenom := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		amount := k.getCollateralForDenomForAddressWithDefault(ctx, collateralDenom.Denom, address)
		if amount.LTE(math.ZeroInt()) {
			continue
		}

		value := amount.ToLegacyDec().Mul(collateralDenom.Ltv).RoundInt()
		valueBase, err := k.DexKeeper.GetValueInBase(ctx, collateralDenom.Denom, value)
		if err != nil {
			return math.LegacyDec{}, errors.Wrap(err, "could not convert collateral amount to base")
		}

		sum = sum.Add(valueBase)
	}

	return sum, nil
}

func (k Keeper) CalcWithdrawableCollateralAmount(ctx context.Context, address, denom string) (math.Int, error) {
	loanSumBase, err := k.getUserLoansSumBase(ctx, address)
	if err != nil {
		return math.Int{}, errors.Wrap(err, "could not get loan sum")
	}

	// When there are no outstanding loans, the whole collateral amount can be withdrawn
	if loanSumBase.Equal(math.LegacyZeroDec()) {
		amount := k.getCollateralForDenomForAddressWithDefault(ctx, denom, address)
		return amount, nil
	}

	collateralDenomLTV, err := k.DenomKeeper.GetLTV(ctx, denom)
	if err != nil {
		return math.Int{}, err
	}

	collateralSumBase, err := k.calcCollateralValueBase(ctx, address)
	if err != nil {
		return math.Int{}, errors.Wrap(err, "could not calculate collateral sum without")
	}

	if loanSumBase.GT(math.LegacyZeroDec()) && loanSumBase.GTE(collateralSumBase) {
		return math.ZeroInt(), nil
	}

	excessAmountBase := collateralSumBase.Sub(loanSumBase)
	excessAmount, err := k.DexKeeper.GetValueIn(ctx, utils.BaseCurrency, denom, excessAmountBase.RoundInt())
	if err != nil {
		return math.Int{}, errors.Wrap(err, "could not convert back to denom currency")
	}

	excessAmount = excessAmount.Quo(collateralDenomLTV)
	collateral := k.getCollateralForDenomForAddressWithDefault(ctx, denom, address)
	excessAmountInt := math.MinInt(collateral, excessAmount.TruncateInt())

	return excessAmountInt, nil
}

func compareCollaterals(c1, c2 types.Collateral) bool {
	return c1.Address == c2.Address && c1.Amount.Equal(c2.Amount)
}
