package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

var minimumAmount = math.NewInt(1_000_000_000) // i.e. 1000

// GetVaultAmount return the amount of funds held in the base denom of an CAsset. For example, when akUSD is the CAsset,
// this functions return the amount of available kUSD
func (k Keeper) GetVaultAmount(ctx context.Context, cAsset denomtypes.CAsset) math.Int {
	address := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault).GetAddress()
	amount := k.BankKeeper.SpendableCoins(ctx, address).AmountOf(cAsset.BaseDexDenom)
	return amount
}

func (k Keeper) getCAssetSupply(ctx context.Context, cAsset denomtypes.CAsset) math.Int {
	return k.BankKeeper.GetSupply(ctx, cAsset.DexDenom).Amount
}

// CalculateNewCAssetAmount calculates how much new c-tokens have to be minted given how much value is being added to
// the vault.
func (k Keeper) CalculateNewCAssetAmount(ctx context.Context, cAsset denomtypes.CAsset, addedAmount math.Int) (math.Int, error) {
	cAssetSupply := k.getCAssetSupply(ctx, cAsset)
	newTokens := math.ZeroInt()

	amountBelowThreshold := minimumAmount.Sub(cAssetSupply)
	if amountBelowThreshold.IsPositive() {
		newTokens = math.MinInt(amountBelowThreshold, addedAmount)
		addedAmount = addedAmount.Sub(newTokens)
	}

	if addedAmount.IsPositive() {
		newTokensFromShare, err := k.CalculateNewCAssetAmountWithShare(ctx, cAsset, addedAmount)
		if err != nil {
			return math.Int{}, err
		}

		newTokens = newTokens.Add(newTokensFromShare)
	}

	return newTokens, nil
}

func (k Keeper) CalculateNewCAssetAmountWithShare(ctx context.Context, cAsset denomtypes.CAsset, addedAmount math.Int) (math.Int, error) {
	cAssetSupply := k.getCAssetSupply(ctx, cAsset)
	if cAssetSupply.LT(minimumAmount) {
		return addedAmount, nil
	}

	loanSum := k.GetLoanSumWithDefault(ctx, cAsset.BaseDexDenom).LoanSum
	vaultSize := k.GetVaultAmount(ctx, cAsset).ToLegacyDec()

	cAssetValue := loanSum.Add(vaultSize)

	newTotalValue := addedAmount.ToLegacyDec().Add(cAssetValue)
	if !newTotalValue.IsPositive() {
		return math.Int{}, fmt.Errorf("new total value not positive")
	}

	valueShare := addedAmount.ToLegacyDec().Quo(newTotalValue) // C

	var newTokens math.Int
	if valueShare.Equal(math.LegacyOneDec()) {
		newTokens = addedAmount
	} else {
		newTokens = cAssetSupply.ToLegacyDec().Quo(math.LegacyOneDec().Sub(valueShare)).TruncateInt().Sub(cAssetSupply) // C
	}

	return newTokens, nil
}

// CalculateCAssetValue calculates the total underlying of an cAsset. This includes funds lying in the vault as well as
// funds in outstanding loans. The amount is expressed in the cAsset's base denom.
func (k Keeper) CalculateCAssetValue(ctx context.Context, cAsset denomtypes.CAsset) math.LegacyDec {
	loanSum := k.GetLoanSumWithDefault(ctx, cAsset.BaseDexDenom).LoanSum
	vaultSize := k.GetVaultAmount(ctx, cAsset).ToLegacyDec()

	return vaultSize.Add(loanSum)
}

func (k Keeper) CalculateCAssetRedemptionValue(ctx context.Context, cAsset denomtypes.CAsset) math.LegacyDec {
	supply := k.BankKeeper.GetSupply(ctx, cAsset.DexDenom)
	if supply.Amount.IsZero() {
		return math.LegacyZeroDec()
	}

	value := k.CalculateCAssetValue(ctx, cAsset)
	redemptionValue := value.Quo(supply.Amount.ToLegacyDec()) // C
	return redemptionValue
}

// calculateCAssetPrice calculates the price of a CAsset in relation to its base denomination.
func (k Keeper) calculateCAssetPrice(ctx context.Context, cAsset denomtypes.CAsset) math.LegacyDec {
	cAssetValue := k.CalculateCAssetValue(ctx, cAsset)
	cAssetSupply := math.LegacyNewDecFromInt(k.BankKeeper.GetSupply(ctx, cAsset.DexDenom).Amount)

	cAssetPrice := math.LegacyOneDec()
	if cAssetSupply.IsPositive() {
		cAssetPrice = cAssetValue.Quo(cAssetSupply) // C
	}

	return cAssetPrice
}

func (k Keeper) ConvertToBaseAmount(ctx context.Context, cAsset denomtypes.CAsset, amountCAsset math.LegacyDec) math.LegacyDec {
	if amountCAsset.IsZero() {
		return math.LegacyZeroDec()
	}

	cAssetValue := k.CalculateCAssetValue(ctx, cAsset)
	cAssetSupply := k.getCAssetSupply(ctx, cAsset)

	return convertToBaseAmount(cAssetSupply.ToLegacyDec(), cAssetValue, amountCAsset)
}

func convertToBaseAmount(supply, value, amountCAsset math.LegacyDec) math.LegacyDec {
	if amountCAsset.IsZero() {
		return math.LegacyZeroDec()
	}

	return amountCAsset.Quo(supply).Mul(value) // C
}
