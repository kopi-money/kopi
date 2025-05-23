package keeper

import (
	"context"

	"github.com/kopi-money/kopi/x/mm/types"
)

func (k Keeper) GetVaultValues(ctx context.Context, _ *types.GetVaultValuesQuery) (*types.GetVaultValuesResponse, error) {
	address := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault).GetAddress()
	balance := k.BankKeeper.SpendableCoins(ctx, address)

	var vaults []types.Vault
	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		vaults = append(vaults, types.Vault{
			Denom:   cAsset.BaseDexDenom,
			Balance: balance.AmountOf(cAsset.BaseDexDenom).String(),
			LoanSum: k.GetLoanSumWithDefault(ctx, cAsset.BaseDexDenom).LoanSum.String(),
			Supply:  k.BankKeeper.GetSupply(ctx, cAsset.DexDenom).Amount.String(),
		})
	}

	return &types.GetVaultValuesResponse{
		Vaults: vaults,
	}, nil
}
