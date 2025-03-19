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

	if _, err := sdk.AccAddressFromBech32(req.Address); err != nil {
		return nil, types.ErrInvalidAddress
	}

	liquidities := []types.AddressLiquidity{}
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		userAmount := k.GetLiquidityByAddress(ctx, denom, req.Address)
		sum := k.GetPoolLiquidity(ctx, denom)

		userAmountUSD, _ := k.DenomKeeper.GetValueInUSD(ctx, denom, userAmount.ToLegacyDec())
		sumUSD, _ := k.DenomKeeper.GetValueInUSD(ctx, denom, sum.ToLegacyDec())

		availableBalance, _ := k.getAvailableBalance(ctx, req.Address, denom)
		availableBalanceUSD, _ := k.DenomKeeper.GetValueInUSD(ctx, denom, availableBalance.ToLegacyDec())

		liquidities = append(liquidities, types.AddressLiquidity{
			Denom:               denom,
			UserAmount:          userAmount.String(),
			UserAmountUsd:       userAmountUSD.String(),
			Total:               sum.String(),
			TotalUsd:            sumUSD.String(),
			AvailableBalance:    availableBalance.String(),
			AvailableBalanceUsd: availableBalanceUSD.String(),
		})
	}

	return &types.QueryLiquidityForAddressResponse{
		Liquidity: liquidities,
	}, nil
}

func (k Keeper) LiquidityPositionForPositionIndex(ctx context.Context, req *types.QueryLiquidityPositionForPositionIndexRequest) (*types.QueryLiquidityPositionForPositionIndexResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &types.QueryLiquidityPositionForPositionIndexResponse{
		LiquidityPositionEntries: k.getLiquidityPositionEntries(ctx, req.PositionIndex),
	}, nil
}

func (k Keeper) LiquidityPositions(ctx context.Context, req *types.QueryLiquidityPositionsRequest) (*types.QueryLiquidityPositionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	liquidityPositionsAddress := []types.QueryLiquidityPositionForAddressResponse{}

	addresses, err := k.liquidityPositions.OuterKeys(ctx)
	if err != nil {
		return nil, err
	}

	for _, address := range addresses {
		liquidityPositionsAddress = append(liquidityPositionsAddress, types.QueryLiquidityPositionForAddressResponse{
			Address:            address,
			LiquidityPositions: k.getLiquidityPositionsForAddress(ctx, address),
		})
	}

	return &types.QueryLiquidityPositionsResponse{
		Addresses: liquidityPositionsAddress,
	}, nil
}

func (k Keeper) LiquidityPositionsAddresses(ctx context.Context, req *types.QueryLiquidityPositionsAddressesRequest) (*types.QueryLiquidityPositionsAddressesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	addresses, err := k.liquidityPositions.OuterKeys(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryLiquidityPositionsAddressesResponse{
		Addresses: addresses,
	}, nil
}

func (k Keeper) LiquidityPositionForAddress(ctx context.Context, req *types.QueryLiquidityPositionForAddressRequest) (*types.QueryLiquidityPositionForAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &types.QueryLiquidityPositionForAddressResponse{
		Address:            req.Address,
		LiquidityPositions: k.getLiquidityPositionsForAddress(ctx, req.Address),
	}, nil
}

func (k Keeper) getLiquidityPositionsForAddress(ctx context.Context, address string) (list []types.LiquidityPositionResponse) {
	iterator := k.liquidityPositions.Iterator(ctx, nil, address)
	for iterator.Valid() {
		keyValue := iterator.GetNextKeyValue()
		liquidityPositionEntries := k.getLiquidityPositionEntries(ctx, keyValue.Key())

		if len(liquidityPositionEntries) > 0 {
			list = append(list, types.LiquidityPositionResponse{
				PositionIndex:            keyValue.Key(),
				AutoCompound:             keyValue.Value().Value().AutoCompound,
				LiquidityPositionEntries: liquidityPositionEntries,
			})
		}
	}

	return
}

func (k Keeper) getLiquidityPositionEntries(ctx context.Context, depositIndex uint64) (list []types.LiquidityPositionEntry) {
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		userAmount := k.GetLiquidityByPositionIndex(ctx, denom, depositIndex)
		if !userAmount.IsPositive() {
			continue
		}

		userAmountUSD, _ := k.DenomKeeper.GetValueInUSD(ctx, denom, userAmount.ToLegacyDec())

		list = append(list, types.LiquidityPositionEntry{
			Denom:         denom,
			UserAmount:    userAmount.String(),
			UserAmountUsd: userAmountUSD.String(),
		})
	}

	return
}

func (k Keeper) getAvailableBalance(ctx context.Context, address, denom string) (math.Int, error) {
	acc, _ := sdk.AccAddressFromBech32(address)
	coins := k.BankKeeper.SpendableCoins(ctx, acc)

	found, coin := coins.Find(denom)
	if found {
		return coin.Amount, nil
	} else {
		return math.ZeroInt(), nil
	}
}
