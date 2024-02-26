package keeper

import (
	"context"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/mm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetVaultValues(ctx context.Context, req *types.GetVaultValuesQuery) (*types.GetVaultValuesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var vaults []*types.Vault

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		vault := types.Vault{
			Denom:   cAsset.BaseDenom,
			Balance: k.getBalance(ctx, cAsset.BaseDenom).String(),
			LoanSum: k.GetLoansSum(ctx, cAsset.BaseDenom).String(),
			Supply:  k.BankKeeper.GetSupply(ctx, cAsset.Name).Amount.String(),
		}

		vaults = append(vaults, &vault)
	}

	return &types.GetVaultValuesResponse{Vaults: vaults}, nil
}

func (k Keeper) getBalance(ctx context.Context, denom string) math.Int {
	address := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault).GetAddress()
	coins := k.BankKeeper.SpendableCoins(ctx, address)

	for _, coin := range coins {
		if coin.Denom == denom {
			return coin.Amount
		}
	}

	return math.ZeroInt()
}
