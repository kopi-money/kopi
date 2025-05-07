package keeper

import (
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/mm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetRedemptionRequest(ctx context.Context, req *types.GetRedemptionRequestQuery) (*types.GetRedemptionRequestResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	request, found := k.redemptions.Get(ctx, req.Denom, req.Address)
	if !found {
		return &types.GetRedemptionRequestResponse{
			Fee:          "0",
			CAssetAmount: "0",
		}, nil
	}

	return &types.GetRedemptionRequestResponse{
		Fee:          request.Fee.String(),
		CAssetAmount: request.Amount.String(),
	}, nil
}

func (k Keeper) GetRedemptionStatsRequest(ctx context.Context, _ *types.GetRedemptionStatsRequestQuery) (*types.GetRedemptionStatsRequestResponse, error) {
	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("get reference denom: %w", err)
	}

	var (
		requestSum    = sdkmath.LegacyZeroDec()
		requestSumUSD sdkmath.LegacyDec
		numRequests   int
	)

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		denomRequestSum, _, denomNumRequests := k.getRedemptionDenomStats(ctx, cAsset.DexDenom)
		requestSumUSD, err = k.DenomKeeper.GetValueIn(ctx, cAsset.DexDenom, referenceDenom, denomRequestSum.ToLegacyDec())
		if err != nil {
			return nil, err
		}

		requestSum = requestSum.Add(requestSumUSD)
		numRequests += denomNumRequests
	}

	return &types.GetRedemptionStatsRequestResponse{
		NumRequests:    int64(numRequests),
		WithdrawSumUsd: requestSum.String(),
	}, nil
}

func (k Keeper) GetRedemptionDenomStatsRequest(ctx context.Context, req *types.GetRedemptionDenomStatsRequestQuery) (*types.GetRedemptionDenomStatsRequestResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	requestSum, maxFee, requestCount := k.getRedemptionDenomStats(ctx, req.Denom)

	maxFeeStr := ""
	if maxFee.GT(sdkmath.LegacyZeroDec()) {
		maxFeeStr = maxFee.String()
	}

	return &types.GetRedemptionDenomStatsRequestResponse{
		NumRequests: int64(requestCount),
		WithdrawSum: requestSum.String(),
		MaxFee:      maxFeeStr,
	}, nil
}

func (k Keeper) getRedemptionDenomStats(ctx context.Context, denom string) (sdkmath.Int, sdkmath.LegacyDec, int) {
	requestSum := sdkmath.ZeroInt()
	maxFee := sdkmath.LegacyZeroDec()

	iterator := k.RedemptionIterator(ctx, denom)
	for iterator.Valid() {
		request := iterator.GetNext()
		requestSum = requestSum.Add(request.Amount)
		maxFee = sdkmath.LegacyMaxDec(maxFee, request.Fee)
	}

	return requestSum, maxFee, k.redemptions.Size()
}

func (k Keeper) GetRedemptionsRequest(ctx context.Context, req *types.GetRedemptionsQuery) (*types.GetRedemptionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	response := types.GetRedemptionsResponse{}

	iterator := k.RedemptionIterator(ctx, req.Denom)
	for iterator.Valid() {
		request := iterator.GetNext()
		response.Requests = append(response.Requests, &types.RedemptionRequest{
			Address: request.Address,
			Amount:  request.Amount.String(),
			Fee:     request.Fee.String(),
		})
	}

	return &response, nil
}
