package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ReserveFunds(ctx context.Context, _ *types.QueryReserveFundsRequest) (*types.QueryReserveFundsResponse, error) {
	address := k.AccountKeeper.GetModuleAccount(ctx, types.PoolReserve).GetAddress()

	total := math.LegacyZeroDec()
	funds := []*types.Denom{}
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		amount := k.GetLiquidityByAddress(ctx, denom, address.String())
		has, coin := k.BankKeeper.SpendableCoins(ctx, address).Find(denom)
		if has {
			amount = amount.Add(coin.Amount)
		}

		priceUSD, err := k.DenomKeeper.GetPriceInUSD(ctx, denom)
		if err != nil {
			if errors.Is(err, types.ErrZeroPrice) {
				priceUSD = math.LegacyZeroDec()
			} else {
				return nil, err
			}
		}

		if !priceUSD.IsPositive() {
			return nil, fmt.Errorf("priceUSD must be positive")
		}

		funds = append(funds, &types.Denom{
			Denom:     denom,
			Amount:    amount.String(),
			AmountUsd: amount.ToLegacyDec().Quo(priceUSD).String(), // C
		})

		total = total.Add(amount.ToLegacyDec().Quo(priceUSD)) // C
	}

	funds = append(funds, &types.Denom{
		Denom:     "total",
		Amount:    total.String(),
		AmountUsd: total.String(),
	})

	return &types.QueryReserveFundsResponse{
		Funds: funds,
	}, nil
}

func (k Keeper) ReserveFundsPerDenom(ctx context.Context, req *types.QueryReserveFundsPerDenomRequest) (*types.Denom, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	address := k.AccountKeeper.GetModuleAccount(ctx, types.PoolReserve).GetAddress()
	amount := k.GetLiquidityByAddress(ctx, req.Denom, address.String())
	has, coin := k.BankKeeper.SpendableCoins(ctx, address).Find(req.Denom)
	if has {
		amount = amount.Add(coin.Amount)
	}

	priceUSD, err := k.DenomKeeper.GetPriceInUSD(ctx, req.Denom)
	if err != nil {
		return nil, err
	}

	return &types.Denom{
		Denom:     req.Denom,
		Amount:    amount.String(),
		AmountUsd: amount.ToLegacyDec().Mul(priceUSD).String(),
	}, nil
}
