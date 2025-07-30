package keeper

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/kopi-money/constants"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) LiquidityAll(ctx context.Context, _ *types.QueryGetLiquidityAllRequest) (*types.QueryGetLiquidityAllResponse, error) {
	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("get highest usd reference: %w", err)
	}

	var (
		entries   []types.QueryGetLiquidityAllResponseEntry
		amountUSD math.LegacyDec
		feeAcc    = k.AccountKeeper.GetModuleAccount(ctx, types.PoolFeeIncome).GetAddress()
	)

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		val := k.GetPoolLiquidity(ctx, denom)
		sum := k.SumLiquidity(ctx, denom)

		amountUSD, err = k.DenomKeeper.GetValueIn(ctx, denom, referenceDenom, val.ToLegacyDec())
		if err != nil {
			amountUSD = math.LegacyZeroDec()
			k.Logger().Error(fmt.Sprintf("convert value %s > %s: %v", denom, referenceDenom, err))
		}

		feeAmount := k.GetLiquidityByAddress(ctx, denom, feeAcc.String())
		feeAmountUSD, _ := k.DenomKeeper.GetValueInUSD(ctx, denom, feeAmount.ToLegacyDec())
		movingLiquidity := k.getMovingLiquidity(ctx, denom).Amount

		entries = append(entries, types.QueryGetLiquidityAllResponseEntry{
			Denom:                 denom,
			Amount:                val.String(),
			AmountUsd:             amountUSD.String(),
			AvailableFeeAmount:    feeAmount.String(),
			AvailableFeeAmountUsd: feeAmountUSD.String(),
			LiquidityPositions:    sum.String(),
			MovingLiquidity:       movingLiquidity.String(),
		})
	}

	return &types.QueryGetLiquidityAllResponse{
		Denoms: entries,
	}, nil
}

func (k Keeper) LiquiditySum(ctx context.Context, _ *types.QueryGetLiquiditySumRequest) (*types.QueryGetLiquiditySumResponse, error) {
	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("get highest usd reference: %w", err)
	}

	valueUSD := math.LegacyZeroDec()
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		val := k.GetPoolLiquidity(ctx, denom)

		var price math.LegacyDec
		price, err = k.DenomKeeper.CalculatePrice(ctx, denom, referenceDenom)
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
	res.Amount = k.GetPoolLiquidity(ctx, req.Denom).String()

	if req.Denom != constants.BaseCurrency {
		pair, err := k.GetLiquidityPair(ctx, req.Denom)
		if err == nil {
			res.VirtualBase = pair.Base.Virtual.String()
			res.VirtualOther = pair.Other.Virtual.String()
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

	iterator := k.liquidityEntries.Iterator(ctx, nil, denom)
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

	iterator := k.liquidityEntries.Iterator(ctx, nil, req.Denom)

	var entries []types.LiquidityEntry
	for iterator.Valid() {
		liq := iterator.GetNext()

		entries = append(entries, types.LiquidityEntry{
			Address:       liq.Address,
			Amount:        liq.Amount.String(),
			Index:         strconv.Itoa(int(liq.Index)),
			PositionIndex: strconv.Itoa(int(liq.PositionIndex)),
		})
	}

	return &types.QueryGetLiquidityQueueResponse{
		Entries: entries,
	}, nil
}

func (k Keeper) LiquidityGrouped(ctx context.Context, req *types.QueryGetLiquidityQueueRequest) (*types.QueryLiquidityGroupedResponse, error) {
	type Value struct {
		sum math.Int
		num int
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	iterator := k.liquidityEntries.Iterator(ctx, nil, req.Denom)

	entryMap := make(map[string]Value)
	for iterator.Valid() {
		liq := iterator.GetNext()

		value, has := entryMap[liq.Address]
		if !has {
			value = Value{
				sum: math.ZeroInt(),
				num: 0,
			}
		}

		value.sum = value.sum.Add(liq.Amount)
		value.num = value.num + 1
		entryMap[liq.Address] = value
	}

	var entries []types.LiquidityGroupedEntry
	for key, value := range entryMap {
		entries = append(entries, types.LiquidityGroupedEntry{
			Address:    key,
			Sum:        value.sum.String(),
			NumEntries: int64(value.num),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Address < entries[j].Address
	})

	return &types.QueryLiquidityGroupedResponse{
		Entries: entries,
	}, nil
}

func (k Keeper) LiquidityPool(ctx context.Context, _ *types.QueryLiquidityPoolRequest) (*types.QueryLiquidityPoolResponse, error) {
	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)
	coins := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

	var entries []types.LiquidityPoolEntry

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		sum := k.GetPoolLiquidity(ctx, denom)
		entrySum := k.getSummedLiquidity(ctx, denom)

		entries = append(entries, types.LiquidityPoolEntry{
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
