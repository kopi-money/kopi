package keeper

import (
	"context"
	"github.com/kopi-money/kopi/utils"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) LiquidityAll(c context.Context, req *types.QueryGetLiquidityAllRequest) (*types.QueryGetLiquidityAllResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	var entries []*types.QueryGetLiquidityAllResponseEntry
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		val := k.GetLiquiditySum(ctx, denom)

		amountUSD, err := k.GetValueInUSD(ctx, denom, val.ToLegacyDec())
		if err != nil {
			return nil, err
		}

		entries = append(entries, &types.QueryGetLiquidityAllResponseEntry{
			Denom:     denom,
			Amount:    val.String(),
			AmountUsd: amountUSD.String(),
		})
	}

	return &types.QueryGetLiquidityAllResponse{Denoms: entries}, nil
}

func (k Keeper) LiquiditySum(ctx context.Context, req *types.QueryGetLiquiditySumRequest) (*types.QueryGetLiquiditySumResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	valueUSD := math.LegacyZeroDec()
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		val := k.GetLiquiditySum(ctx, denom)
		price, _ := k.GetPriceInUSD(ctx, denom)
		valueUSD = valueUSD.Add(val.ToLegacyDec().Quo(price))
	}

	return &types.QueryGetLiquiditySumResponse{ValueUsd: valueUSD.String()}, nil
}

func (k Keeper) Liquidity(ctx context.Context, req *types.QueryGetLiquidityRequest) (*types.QueryGetLiquidityResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	res := types.QueryGetLiquidityResponse{}
	res.Amount = k.GetLiquiditySum(ctx, req.Denom).String()

	if req.Denom != utils.BaseCurrency {
		pair, err := k.GetLiquidityPair(ctx, req.Denom)
		if err == nil {
			res.VirtualOther = pair.VirtualOther.String()
			res.VirtualBase = pair.VirtualBase.String()
		}
	}

	res.Sum = k.getSummedLiquidity(ctx, req.Denom).String()

	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	coins := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())
	found, coin := coins.Find(req.Denom)
	if found {
		res.Pool = coin.Amount.String()
	} else {
		res.Pool = math.ZeroInt().String()
	}

	return &res, nil
}

func (k Keeper) getSummedLiquidity(ctx context.Context, denom string) math.Int {
	sum := math.ZeroInt()
	for _, liq := range k.GetAllLiquidity(ctx) {
		if liq.Denom == denom {
			sum = sum.Add(liq.Amount)
		}
	}

	return sum
}

func (k Keeper) LiquidityQueue(c context.Context, req *types.QueryGetLiquidityQueueRequest) (*types.QueryGetLiquidityQueueResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	iterator := k.LiquidityIterator(ctx, req.Denom)

	var entries []*types.LiquidityEntry
	for iterator.Valid() {
		liq := iterator.GetNext()

		entries = append(entries, &types.LiquidityEntry{
			Address: liq.Address,
			Amount:  liq.Amount.String(),
		})
	}

	return &types.QueryGetLiquidityQueueResponse{Entries: entries}, nil
}

func (k Keeper) LiquidityPool(ctx context.Context, req *types.QueryLiquidityPoolRequest) (*types.QueryLiquidityPoolResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	coins := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

	var entries []*types.LiquidityPoolEntry

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		sum := k.GetLiquiditySum(ctx, denom)
		entrySum := k.getSummedLiquidity(ctx, denom)

		entries = append(entries, &types.LiquidityPoolEntry{
			Denom:        denom,
			PoolAmount:   coins.AmountOf(denom).String(),
			LiquiditySum: sum.String(),
			EntriesSum:   entrySum.String(),
		})
	}

	return &types.QueryLiquidityPoolResponse{Entries: entries}, nil
}
