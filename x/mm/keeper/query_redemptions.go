package keeper

import (
	"context"
	"cosmossdk.io/collections"

	sdkmath "cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/mm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetRedemptionRequest(ctx context.Context, req *types.GetRedemptionRequestQuery) (*types.GetRedemptionRequestResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	key := collections.Join(req.Denom, req.Address)
	request, found := k.redemptions.Get(ctx, key)
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

func (k Keeper) GetRedemptionStatsRequest(ctx context.Context, req *types.GetRedemptionStatsRequestQuery) (*types.GetRedemptionStatsRequestResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	requestSum := sdkmath.LegacyZeroDec()
	numRequests := 0

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		denomRequestSum, _, denomNumRequests := k.getRedemptionDenomStats(ctx, cAsset.Name)
		requestSumUsd, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.Name, denomRequestSum)
		if err != nil {
			return nil, err
		}

		requestSum = requestSum.Add(requestSumUsd)
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
