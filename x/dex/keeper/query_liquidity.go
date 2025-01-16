package keeper

import (
	"context"
	"fmt"

	"github.com/kopi-money/kopi/constants"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) LiquidityAll(ctx context.Context, _ *types.QueryGetLiquidityAllRequest) (*types.QueryGetLiquidityAllResponse, error) {
	referenceDenom, err := k.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get highest usd reference: %w", err)
	}

	var (
		entries   []*types.QueryGetLiquidityAllResponseEntry
		amountUSD math.LegacyDec
	)

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		val := k.GetLiquiditySum(ctx, denom)

		amountUSD, err = k.GetValueIn(ctx, denom, referenceDenom, val.ToLegacyDec())
		if err != nil {
			return nil, fmt.Errorf("could not convert value %s > %s: %w", denom, referenceDenom, err)
		}

		entries = append(entries, &types.QueryGetLiquidityAllResponseEntry{
			Denom:     denom,
			Amount:    val.String(),
			AmountUsd: amountUSD.String(),
		})
	}

	return &types.QueryGetLiquidityAllResponse{
		Denoms: entries,
	}, nil
}

func (k Keeper) LiquiditySum(ctx context.Context, _ *types.QueryGetLiquiditySumRequest) (*types.QueryGetLiquiditySumResponse, error) {
	referenceDenom, err := k.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get highest usd reference: %w", err)
	}

	valueUSD := math.LegacyZeroDec()
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		val := k.GetLiquiditySum(ctx, denom)

		var price math.LegacyDec
		price, err = k.CalculatePrice(ctx, denom, referenceDenom)
		if err != nil {
			return nil, fmt.Errorf("calculate price: %w", err)
		}

		if !price.IsPositive() {
			return nil, fmt.Errorf("price is not positive")
		}

		valueUSD = valueUSD.Add(val.ToLegacyDec().Quo(price)) // C
	}

	return &types.QueryGetLiquiditySumResponse{
		ValueUsd: valueUSD.String(),
	}, nil
}

func (k Keeper) Liquidity(ctx context.Context, req *types.QueryGetLiquidityRequest) (*types.QueryGetLiquidityResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	res := types.QueryGetLiquidityResponse{}
	res.Amount = k.GetLiquiditySum(ctx, req.Denom).String()

	if req.Denom != constants.BaseCurrency {
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

	iterator := k.LiquidityIterator(ctx, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()
		sum = sum.Add(liq.Amount)
	}

	return sum
}

func (k Keeper) LiquidityQueue(ctx context.Context, req *types.QueryGetLiquidityQueueRequest) (*types.QueryGetLiquidityQueueResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	iterator := k.LiquidityIterator(ctx, req.Denom)

	var entries []*types.LiquidityEntry
	for iterator.Valid() {
		liq := iterator.GetNext()

		entries = append(entries, &types.LiquidityEntry{
			Address: liq.Address,
			Amount:  liq.Amount.String(),
		})
	}

	return &types.QueryGetLiquidityQueueResponse{
		Entries: entries,
	}, nil
}

func (k Keeper) LiquidityPool(ctx context.Context, _ *types.QueryLiquidityPoolRequest) (*types.QueryLiquidityPoolResponse, error) {
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

	return &types.QueryLiquidityPoolResponse{
		Entries: entries,
	}, nil
}
