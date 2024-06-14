package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) LiquidityForAddress(goCtx context.Context, req *types.QueryLiquidityForAddressRequest) (*types.QueryLiquidityForAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	_, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	liquidities := []*types.AddressLiquidity{}
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		userAmount := k.GetLiquidityByAddress(ctx, denom, req.Address)
		sum := k.GetLiquiditySum(ctx, denom)

		userAmountUSD, _ := k.GetValueInUSD(ctx, denom, userAmount.ToLegacyDec())
		sumUSD, _ := k.GetValueInUSD(ctx, denom, sum.ToLegacyDec())

		availableBalance, _ := k.getAvailableBalance(ctx, req.Address, denom)
		availableBalanceUSD, _ := k.GetValueInUSD(ctx, denom, availableBalance.ToLegacyDec())

		liquidity := types.AddressLiquidity{}
		liquidity.UserAmount = userAmount.String()
		liquidity.UserAmountUsd = userAmountUSD.String()
		liquidity.Denom = denom
		liquidity.Total = sum.String()
		liquidity.TotalUsd = sumUSD.String()
		liquidity.AvailableBalance = availableBalance.String()
		liquidity.AvailableBalanceUsd = availableBalanceUSD.String()

		liquidities = append(liquidities, &liquidity)
	}

	return &types.QueryLiquidityForAddressResponse{Liquidity: liquidities}, nil
}

func (k Keeper) getAvailableBalance(ctx sdk.Context, address, denom string) (math.Int, error) {
	acc, _ := sdk.AccAddressFromBech32(address)
	coins := k.BankKeeper.SpendableCoins(ctx, acc)

	found, coin := coins.Find(denom)
	if found {
		return coin.Amount, nil
	} else {
		return math.ZeroInt(), nil
	}
}
