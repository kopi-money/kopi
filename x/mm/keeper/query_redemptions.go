package keeper

import (
	"context"
	sdkmath "cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/mm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetRedemptionRequest(ctx context.Context, req *types.GetRedemptionRequestQuery) (*types.GetRedemptionRequestResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	request, found := k.GetRedemption(ctx, req.Denom, req.Address)
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

	requests := k.GetRedemptions(ctx, req.Denom)

	requestSum := sdkmath.ZeroInt()
	requestCount := 0
	maxFee := sdkmath.LegacyZeroDec()

	for _, request := range requests {
		requestCount++
		requestSum = requestSum.Add(request.Amount)
		maxFee = sdkmath.LegacyMaxDec(maxFee, request.Fee)
	}

	maxFeeStr := ""
	if maxFee.GT(sdkmath.LegacyZeroDec()) {
		maxFeeStr = maxFee.String()
	}

	return &types.GetRedemptionStatsRequestResponse{
		NumRequests: int64(requestCount),
		WithdrawSum: requestSum.String(),
		MaxFee:      maxFeeStr,
	}, nil
}

func (k Keeper) GetRedemptionsRequest(ctx context.Context, req *types.GetRedemptionsQuery) (*types.GetRedemptionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	requests := k.GetRedemptions(ctx, req.Denom)

	response := types.GetRedemptionsResponse{}
	for _, request := range requests {
		response.Requests = append(response.Requests, &types.RedemptionRequest{
			Address: request.Address,
			Amount:  request.Amount.String(),
			Fee:     request.Fee.String(),
		})
	}

	return &response, nil
}
