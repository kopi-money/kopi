package keeper

import (
	"context"
	"cosmossdk.io/math"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

// GetVaultAmount return the amount of funds held in the base denom of an CAsset. For example, when akUSD is the CAsset,
// this functions return the amount of available kUSD
func (k Keeper) GetVaultAmount(ctx context.Context, cAsset *denomtypes.CAsset) math.Int {
	address := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault).GetAddress()
	amount := k.BankKeeper.SpendableCoins(ctx, address).AmountOf(cAsset.BaseDenom)
	return amount
}

func (k Keeper) getCAssetSupply(ctx context.Context, cAsset *denomtypes.CAsset) math.Int {
	return k.BankKeeper.GetSupply(ctx, cAsset.Name).Amount
}

// CalculateNewCAssetAmount calculates how much new a-tokens have to be minted given how much value is being added to
// the vault
func (k Keeper) CalculateNewCAssetAmount(ctx context.Context, addedAmount math.Int, cAsset *denomtypes.CAsset) math.Int {
	cAssetSupply := k.getCAssetSupply(ctx, cAsset)
	if cAssetSupply.Equal(math.ZeroInt()) {
		return addedAmount
	}

	cAssetValue := k.calculateCAssetValue(ctx, cAsset)
	valueShare := addedAmount.ToLegacyDec().Quo(cAssetValue)

	var newTokens math.Int
	if valueShare.Equal(math.LegacyOneDec()) {
		newTokens = addedAmount
	} else {
		newTokens = cAssetSupply.ToLegacyDec().Quo(math.LegacyOneDec().Sub(valueShare)).RoundInt().Sub(cAssetSupply)
		//msg := fmt.Sprintf("Given: %v, supply: %v, vaultAmount: %v, valueShare: %v, newTokens: %v", addedAmount.String(), cAssetSupply.String(), vaultAmount.String(), valueShare.String(), newTokens.String())
		//k.logger.Info(msg)
	}

	return newTokens
}

// calculateCAssetValue calculates the total underlying of an CAsset. This includes funds lying in the vault as well as
// funds in outstanding loans.
func (k Keeper) calculateCAssetValue(ctx context.Context, cAsset *denomtypes.CAsset) math.LegacyDec {
	cAssetValue := k.GetVaultAmount(ctx, cAsset).ToLegacyDec()
	for _, loan := range k.GetAllLoansByDenom(ctx, cAsset.BaseDenom) {
		cAssetValue = cAssetValue.Add(loan.Amount)
	}

	return cAssetValue
}

// calculateCAssetPrice calculates the price of a CAsset in relation to its base denomination.
func (k Keeper) calculateCAssetPrice(ctx context.Context, cAsset *denomtypes.CAsset) math.LegacyDec {
	CAssetValue := k.calculateCAssetValue(ctx, cAsset)
	CAssetSupply := math.LegacyNewDecFromInt(k.BankKeeper.GetSupply(ctx, cAsset.Name).Amount)

	CAssetPrice := math.LegacyOneDec()
	if CAssetSupply.GT(math.LegacyZeroDec()) {
		CAssetPrice = CAssetValue.Quo(CAssetSupply)
	}

	return CAssetPrice
}

func (k Keeper) ConvertToBaseAmount(ctx context.Context, cAsset *denomtypes.CAsset, amountCAsset math.Int) math.LegacyDec {
	if amountCAsset.Equal(math.ZeroInt()) {
		return math.LegacyZeroDec()
	}

	cAssetValue := k.calculateCAssetValue(ctx, cAsset)
	cAssetSupply := k.getCAssetSupply(ctx, cAsset)

	return convertToBaseAmount(cAssetSupply.ToLegacyDec(), cAssetValue, amountCAsset.ToLegacyDec())
}

func convertToBaseAmount(supply, value, amountCAsset math.LegacyDec) math.LegacyDec {
	if amountCAsset.Equal(math.LegacyZeroDec()) {
		return math.LegacyZeroDec()
	}

	return amountCAsset.Quo(supply).Mul(value)
}
