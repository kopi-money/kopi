package keeper

import (
	"context"
	"cosmossdk.io/collections"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/x/mm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetCollateralStats(ctx context.Context, req *types.GetCollateralStatsQuery) (*types.GetCollateralStatsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	totalUSD := math.LegacyZeroDec()
	var stats []*types.CollateralDenomStats

	for _, denom := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		sum := k.getCollateralSum(ctx, denom.Denom)
		sumUSD, err := k.DexKeeper.GetValueInUSD(ctx, denom.Denom, sum.ToLegacyDec())
		if err != nil {
			return nil, errors.Wrap(err, "could not get collateral sum in usd")
		}

		depositCap, err := k.DenomKeeper.GetDepositCap(ctx, denom.Denom)
		if err != nil {
			return nil, errors.Wrap(err, "could not get deposit cap")
		}

		depositCapUsed := math.LegacyZeroDec()
		if depositCap.GT(math.ZeroInt()) {
			depositCapUsed = sum.ToLegacyDec().Quo(depositCap.ToLegacyDec())
		}

		totalUSD = totalUSD.Add(sumUSD)
		priceUSD, err := k.DexKeeper.GetPriceInUSD(ctx, denom.Denom)
		if err != nil {
			return nil, errors.Wrap(err, "could not get price in usd")
		}

		stats = append(stats, &types.CollateralDenomStats{
			Denom:              denom.Denom,
			DepositedMarket:    sum.String(),
			DepositedMarketUsd: sumUSD.String(),
			Ltv:                denom.Ltv.String(),
			PriceUsd:           priceUSD.String(),
			DepositedUser:      sum.String(),
			DepositCap:         depositCap.String(),
			DepositCapUsed:     depositCapUsed.String(),
		})
	}

	return &types.GetCollateralStatsResponse{Stats: stats, TotalUsd: totalUSD.String()}, nil
}

func (k Keeper) GetCollateralDenomStats(ctx context.Context, req *types.GetCollateralDenomStatsQuery) (*types.GetCollateralDenomStatsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	denom := k.DenomKeeper.GetCollateralDenom(ctx, req.Denom)
	if denom == nil {
		return nil, types.ErrInvalidCollateralDenom
	}

	sum := math.ZeroInt()
	collaterals := []*types.UserCollateral{}

	iterator := k.CollateralIterator(ctx, denom.Denom)
	for iterator.Valid() {
		collateral := iterator.GetNext()
		sum = sum.Add(collateral.Amount)

		collaterals = append(collaterals, &types.UserCollateral{
			Address: collateral.Address,
			Amount:  collateral.Amount.String(),
		})
	}

	sumUSD, err := k.DexKeeper.GetValueInUSD(ctx, denom.Denom, sum.ToLegacyDec())
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

	totalUSD := math.LegacyZeroDec()
	var stats []*types.CollateralDenomStats

	for _, denom := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		collateral, has := k.collateral.Get(ctx, collections.Join(denom.Denom, req.Address))
		if !has {
			collateral.Amount = math.ZeroInt()
		}

		depositCap, err := k.DenomKeeper.GetDepositCap(ctx, denom.Denom)
		if err != nil {
			return nil, errors.Wrap(err, "could not get deposit cap")
		}

		collateralSum := k.getCollateralSum(ctx, denom.Denom)
		collateralSumUSD, err := k.DexKeeper.GetValueInUSD(ctx, denom.Denom, collateral.Amount.ToLegacyDec())
		if err != nil {
			continue
		}

		depositCapUsed := math.LegacyZeroDec()
		if depositCap.GT(math.ZeroInt()) {
			depositCapUsed = collateralSum.ToLegacyDec().Quo(depositCap.ToLegacyDec())
		}

		priceUSD, err := k.DexKeeper.GetPriceInUSD(ctx, denom.Denom)
		if err != nil {
			return nil, errors.Wrap(err, "could not get price in usd")
		}

		totalUSD = totalUSD.Add(collateralSumUSD)

		stats = append(stats, &types.CollateralDenomStats{
			Denom:              denom.Denom,
			DepositedMarket:    collateralSum.String(),
			DepositedMarketUsd: collateralSumUSD.String(),
			Ltv:                denom.Ltv.String(),
			PriceUsd:           priceUSD.String(),
			DepositedUser:      collateral.Amount.String(),
			DepositCap:         depositCap.String(),
			DepositCapUsed:     depositCapUsed.String(),
		})
	}

	return &types.GetCollateralStatsResponse{Stats: stats, TotalUsd: totalUSD.String()}, nil
}

func (k Keeper) GetCollateralDenomUserStats(ctx context.Context, req *types.GetCollateralDenomUserStatsQuery) (*types.GetCollateralDenomUserStatsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	address, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	available := k.BankKeeper.SpendableCoin(ctx, address, req.Denom)
	availableUSD, err := k.DexKeeper.GetValueInUSD(ctx, req.Denom, available.Amount.ToLegacyDec())
	if err != nil {
		return nil, errors.Wrap(err, "could not get available value in used")
	}

	collateral, has := k.collateral.Get(ctx, collections.Join(req.Denom, req.Address))
	if !has {
		collateral.Amount = math.ZeroInt()
	}

	providedUSD, err := k.DexKeeper.GetValueInUSD(ctx, req.Denom, collateral.Amount.ToLegacyDec())
	if err != nil {
		return nil, errors.Wrap(err, "could not get provided value in used")
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
		return nil, errors.Wrap(err, "could not calculate withdrawable amount")
	}

	withdrawableUSD, err := k.DexKeeper.GetValueInUSD(ctx, req.Denom, withdrawable)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert withdrawable amount to usd")
	}

	return &types.GetWithdrawableCollateralResponse{
		Amount:    withdrawable.String(),
		AmountUsd: withdrawableUSD.String(),
	}, nil
}
