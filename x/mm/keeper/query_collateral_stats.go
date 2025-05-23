package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/x/mm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetCollateralStats(ctx context.Context, _ *types.GetCollateralStatsQuery) (*types.GetCollateralStatsResponse, error) {
	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("get reference denom: %w", err)
	}

	var (
		stats      []types.CollateralDenomStats
		totalUSD   = math.LegacyZeroDec()
		sumUSD     math.LegacyDec
		priceUSD   math.LegacyDec
		depositCap math.Int
	)

	for _, denom := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		sum := k.getCollateralSum(ctx, denom.DexDenom)
		sumUSD, err = k.DenomKeeper.GetValueIn(ctx, denom.DexDenom, referenceDenom, sum.ToLegacyDec())
		if err != nil {
			return nil, fmt.Errorf("get collateral sum in usd: %w", err)
		}

		depositCap, err = k.DenomKeeper.GetDepositCap(ctx, denom.DexDenom)
		if err != nil {
			return nil, fmt.Errorf("get deposit cap: %w", err)
		}

		depositCapUsed := math.LegacyZeroDec()
		if depositCap.IsPositive() {
			depositCapUsed = sum.ToLegacyDec().Quo(depositCap.ToLegacyDec()) // C
		}

		totalUSD = totalUSD.Add(sumUSD)
		priceUSD, err = k.DenomKeeper.CalculatePrice(ctx, denom.DexDenom, referenceDenom)
		if err != nil {
			return nil, fmt.Errorf("get price in usd (%v): %w", denom.DexDenom, err)
		}

		stats = append(stats, types.CollateralDenomStats{
			Denom:              denom.DexDenom,
			DepositedMarket:    sum.String(),
			DepositedMarketUsd: sumUSD.String(),
			Ltv:                denom.Ltv.String(),
			PriceUsd:           priceUSD.String(),
			DepositedUser:      sum.String(),
			DepositCap:         depositCap.String(),
			DepositCapUsed:     depositCapUsed.String(),
		})
	}

	return &types.GetCollateralStatsResponse{
		Stats:    stats,
		TotalUsd: totalUSD.String(),
	}, nil
}

func (k Keeper) GetCollateralDenomStats(ctx context.Context, req *types.GetCollateralDenomStatsQuery) (*types.GetCollateralDenomStatsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	denom, err := k.DenomKeeper.GetCollateralDenom(ctx, req.Denom)
	if err != nil {
		return nil, err
	}

	sum := math.ZeroInt()
	collaterals := []types.UserCollateral{}

	iterator := k.CollateralIterator(ctx, denom.DexDenom)
	for iterator.Valid() {
		collateral := iterator.GetNext()
		sum = sum.Add(collateral.Amount)

		collaterals = append(collaterals, types.UserCollateral{
			Address: collateral.Address,
			Amount:  collateral.Amount.String(),
		})
	}

	sumUSD, err := k.DenomKeeper.GetValueInUSD(ctx, denom.DexDenom, sum.ToLegacyDec())
	if err != nil {
		return nil, err
	}

	return &types.GetCollateralDenomStatsResponse{
		UserCollateral: collaterals,
		Sum:            sum.String(),
		SumUsd:         sumUSD.String(),
	}, nil
}

func (k Keeper) GetCollateralUserStats(ctx context.Context, req *types.GetCollateralUserStatsQuery) (*types.GetCollateralStatsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("get reference denom: %w", err)
	}

	var (
		totalUSD         = math.LegacyZeroDec()
		stats            = []types.CollateralDenomStats{}
		depositCap       math.Int
		collateralSumUSD math.LegacyDec
		priceUSD         math.LegacyDec
	)

	for _, denom := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		collateral, has := k.collateral.Get(ctx, denom.DexDenom, req.Address)
		if !has {
			collateral.Amount = math.ZeroInt()
		}

		depositCap, err = k.DenomKeeper.GetDepositCap(ctx, denom.DexDenom)
		if err != nil {
			return nil, fmt.Errorf("get deposit cap: %w", err)
		}

		collateralSum := k.getCollateralSum(ctx, denom.DexDenom)
		collateralSumUSD, err = k.DenomKeeper.GetValueIn(ctx, denom.DexDenom, referenceDenom, collateral.Amount.ToLegacyDec())
		if err != nil {
			continue
		}

		depositCapUsed := math.LegacyZeroDec()
		if depositCap.IsPositive() {
			depositCapUsed = collateralSum.ToLegacyDec().Quo(depositCap.ToLegacyDec()) // C
		}

		priceUSD, err = k.DenomKeeper.CalculatePrice(ctx, denom.DexDenom, referenceDenom)
		if err != nil {
			return nil, fmt.Errorf("get price in usd: %w", err)
		}

		totalUSD = totalUSD.Add(collateralSumUSD)

		stats = append(stats, types.CollateralDenomStats{
			Denom:              denom.DexDenom,
			DepositedMarket:    collateralSum.String(),
			DepositedMarketUsd: collateralSumUSD.String(),
			Ltv:                denom.Ltv.String(),
			PriceUsd:           priceUSD.String(),
			DepositedUser:      collateral.Amount.String(),
			DepositCap:         depositCap.String(),
			DepositCapUsed:     depositCapUsed.String(),
		})
	}

	return &types.GetCollateralStatsResponse{
		Stats:    stats,
		TotalUsd: totalUSD.String(),
	}, nil
}

func (k Keeper) GetCollateralDenomUserStats(ctx context.Context, req *types.GetCollateralDenomUserStatsQuery) (*types.GetCollateralDenomUserStatsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	address, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("get reference denom: %w", err)
	}

	available := k.BankKeeper.SpendableCoin(ctx, address, req.Denom)
	availableUSD, err := k.DenomKeeper.GetValueIn(ctx, req.Denom, referenceDenom, available.Amount.ToLegacyDec())
	if err != nil {
		return nil, fmt.Errorf("get available value in usd: %w", err)
	}

	collateral, has := k.collateral.Get(ctx, req.Denom, req.Address)
	if !has {
		collateral.Amount = math.ZeroInt()
	}

	providedUSD, err := k.DenomKeeper.GetValueIn(ctx, req.Denom, referenceDenom, collateral.Amount.ToLegacyDec())
	if err != nil {
		return nil, fmt.Errorf("get provided value in usd: %w", err)
	}

	return &types.GetCollateralDenomUserStatsResponse{
		Available:    available.Amount.String(),
		AvailableUsd: availableUSD.RoundInt().String(),
		Provided:     collateral.Amount.String(),
		ProvidedUsd:  providedUSD.RoundInt().String(),
	}, nil
}

func (k Keeper) GetWithdrawableCollateral(ctx context.Context, req *types.GetWithdrawableCollateralQuery) (*types.GetWithdrawableCollateralResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	withdrawable, err := k.CalcWithdrawableCollateralAmount(ctx, req.Address, req.Denom)
	if err != nil {
		return nil, fmt.Errorf("calculate withdrawable amount: %w", err)
	}

	withdrawableUSD, err := k.DenomKeeper.GetValueInUSD(ctx, req.Denom, withdrawable)
	if err != nil {
		return nil, fmt.Errorf("convert withdrawable amount to usd: %w", err)
	}

	return &types.GetWithdrawableCollateralResponse{
		Amount:    withdrawable.String(),
		AmountUsd: withdrawableUSD.String(),
	}, nil
}
