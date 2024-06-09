package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ReserveFunds(goCtx context.Context, req *types.QueryReserveFundsRequest) (*types.QueryReserveFundsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	address := k.AccountKeeper.GetModuleAccount(ctx, types.PoolReserve).GetAddress()

	total := math.LegacyZeroDec()
	funds := []*types.Denom{}
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		amount := k.GetLiquidityByAddress(ctx, denom, address.String())
		has, coin := k.BankKeeper.SpendableCoins(ctx, address).Find(denom)
		if has {
			amount = amount.Add(coin.Amount)
		}

		priceUSD, err := k.GetPriceInUSD(ctx, denom)
		if err != nil {
			return nil, err
		}

		funds = append(funds, &types.Denom{
			Denom:     denom,
			Amount:    amount.String(),
			AmountUsd: amount.ToLegacyDec().Quo(priceUSD).String(),
		})

		total = total.Add(amount.ToLegacyDec().Quo(priceUSD))
	}

	funds = append(funds, &types.Denom{
		Denom:     "total",
		Amount:    total.String(),
		AmountUsd: total.String(),
	})

	return &types.QueryReserveFundsResponse{Funds: funds}, nil
}

func (k Keeper) ReserveFundsPerDenom(goCtx context.Context, req *types.QueryReserveFundsPerDenomRequest) (*types.Denom, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	address := k.AccountKeeper.GetModuleAccount(ctx, types.PoolReserve).GetAddress()
	amount := k.GetLiquidityByAddress(ctx, req.Denom, address.String())
	has, coin := k.BankKeeper.SpendableCoins(ctx, address).Find(req.Denom)
	if has {
		amount = amount.Add(coin.Amount)
	}

	priceUSD, err := k.GetPriceInUSD(ctx, req.Denom)
	if err != nil {
		return nil, err
	}

	return &types.Denom{
		Denom:     req.Denom,
		Amount:    amount.String(),
		AmountUsd: amount.ToLegacyDec().Mul(priceUSD).String(),
	}, nil
}
