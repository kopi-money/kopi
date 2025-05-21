package keeper

import (
	"context"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) QueryDistribution(ctx context.Context, req *types.QueryDistributionRequest) (*types.QueryDistributionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	factoryDenom, has := k.factoryDenoms.Get(ctx, req.FullName)
	if !has {
		return nil, types.ErrDenomDoesNotExists
	}

	accAdmin, _ := sdk.AccAddressFromBech32(factoryDenom.Admin)
	accPool := k.AccountKeeper.GetModuleAccount(ctx, types.PoolFactoryLiquidity)
	accOffers := k.AccountKeeper.GetModuleAccount(ctx, types.PoolOffers)
	accVesting := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVestings)
	accUnlocking := k.AccountKeeper.GetModuleAccount(ctx, types.PoolUnlocking)

	supply := k.BankKeeper.GetSupply(ctx, req.FullName)
	amountAdmin := k.BankKeeper.SpendableCoin(ctx, accAdmin, req.FullName).Amount
	amountPool := k.BankKeeper.SpendableCoin(ctx, accPool.GetAddress(), req.FullName).Amount
	amountOffers := k.BankKeeper.SpendableCoin(ctx, accOffers.GetAddress(), req.FullName).Amount
	amountVesting := k.BankKeeper.SpendableCoin(ctx, accVesting.GetAddress(), req.FullName).Amount
	amountUnlocking := k.BankKeeper.SpendableCoin(ctx, accUnlocking.GetAddress(), req.FullName).Amount
	amountWallets := supply.Amount.Sub(amountPool).Sub(amountOffers).Sub(amountVesting).Sub(amountAdmin).Sub(amountUnlocking)

	supplyDec := supply.Amount.ToLegacyDec()
	shareAdmin := amountAdmin.ToLegacyDec().Quo(supplyDec)
	sharePool := amountPool.ToLegacyDec().Quo(supplyDec)
	shareOffers := amountOffers.ToLegacyDec().Quo(supplyDec)
	shareVesting := amountVesting.ToLegacyDec().Quo(supplyDec)
	shareUnlocking := amountUnlocking.ToLegacyDec().Quo(supplyDec)
	shareWallets := amountWallets.ToLegacyDec().Quo(supplyDec)

	return &types.QueryDistributionResponse{
		TotalSupply:    supply.Amount.String(),
		Pool:           amountPool.String(),
		Vested:         amountVesting.String(),
		Offered:        amountOffers.String(),
		Unlocking:      amountUnlocking.String(),
		Wallets:        amountWallets.String(),
		Admin:          amountAdmin.String(),
		ShareAdmin:     shareAdmin.String(),
		SharePool:      sharePool.String(),
		ShareVested:    shareVesting.String(),
		ShareOffered:   shareOffers.String(),
		ShareUnlocking: shareUnlocking.String(),
		ShareWallets:   shareWallets.String(),
	}, nil
}

func (k Keeper) QueryPoolLiquidityDistribution(ctx context.Context, req *types.QueryPoolLiquidityDistributionRequest) (*types.QueryPoolLiquidityDistributionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	pool, has := k.liquidityPools.Get(ctx, req.FullName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	var (
		liquidityProviders []types.LiquidityProvider
		sumAmountKCoin     = math.ZeroInt()
		sumAmountFactory   = math.ZeroInt()
	)

	iterator := k.liquidityProviderShares.Iterator(ctx, nil, req.FullName)
	for iterator.Valid() {
		keyValue := iterator.GetNextKeyValue()

		amountFactory, amountKCoin := pool.GetAmounts(keyValue.Value().Value().Share)

		liquidityProviders = append(liquidityProviders, types.LiquidityProvider{
			Address:       keyValue.Key(),
			AmountKcoin:   amountKCoin.String(),
			AmountFactory: amountFactory.String(),
		})

		sumAmountKCoin = sumAmountKCoin.Add(amountKCoin)
		sumAmountFactory = sumAmountFactory.Add(amountFactory)
	}

	return &types.QueryPoolLiquidityDistributionResponse{
		Providers:     liquidityProviders,
		AmountKcoin:   sumAmountKCoin.String(),
		AmountFactory: sumAmountFactory.String(),
	}, nil
}
