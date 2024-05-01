package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k Keeper) GetAllDenomCollaterals(ctx context.Context) (list []types.Collaterals) {
	for _, collateralDemom := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		var collaterals []*types.Collateral
		for _, collateral := range k.GetAllCollaterals(ctx, collateralDemom.Denom) {
			collaterals = append(collaterals, &collateral)
		}

		list = append(list, types.Collaterals{
			Denom:       collateralDemom.Denom,
			Collaterals: collaterals,
		})
	}

	return
}

// GetAllCollaterals returns all deposits
func (k Keeper) GetAllCollaterals(ctx context.Context, denom string) (list []types.Collateral) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixCollaterals))

	iterator := storetypes.KVStorePrefixIterator(store, types.KeyDenom(denom))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Collateral
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// SetCollateral set a specific deposits in the store from its index
func (k Keeper) SetCollateral(ctx context.Context, denom string, collateral types.Collateral) {
	if collateral.Amount.LTE(math.ZeroInt()) {
		k.RemoveCollateral(ctx, denom, collateral.Address)
		return
	}

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixCollaterals))
	b := k.cdc.MustMarshal(&collateral)
	store.Set(types.KeyDenomAddress(denom, collateral.Address), b)
}

func (k Keeper) RemoveCollateral(ctx context.Context, denom, address string) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixCollaterals))
	store.Delete(types.KeyDenomAddress(denom, address))
}

// GetCollateral returns a deposits from its index
func (k Keeper) GetCollateral(ctx context.Context, denom, address string) (types.Collateral, bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixCollaterals))
	b := store.Get(types.KeyDenomAddress(denom, address))
	if b == nil {
		return types.Collateral{}, false
	}

	var deposit types.Collateral
	k.cdc.MustUnmarshal(b, &deposit)
	return deposit, true
}

func (k Keeper) getCollateralDenomForAddress(ctx context.Context, denom, address string) math.Int {
	amount, found := k.GetCollateral(ctx, denom, address)
	if !found {
		return math.ZeroInt()
	}

	return amount.Amount
}

func (k Keeper) getCollateralSum(ctx context.Context, denom string) math.Int {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixCollaterals))

	iterator := storetypes.KVStorePrefixIterator(store, types.KeyDenom(denom))
	defer iterator.Close()

	sum := math.ZeroInt()
	for ; iterator.Valid(); iterator.Next() {
		var val types.Collateral
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		sum = sum.Add(val.Amount)
	}

	return sum
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
		amount := k.getCollateralDenomForAddress(ctx, collateralDenom.Denom, address)
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
		amount := k.getCollateralDenomForAddress(ctx, denom, address)
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
	collateral := k.getCollateralDenomForAddress(ctx, denom, address)
	excessAmountInt := math.MinInt(collateral, excessAmount.TruncateInt())

	return excessAmountInt, nil
}
