package keeper

import (
	"context"

	"github.com/kopi-money/kopi/x/tokenfactory/types"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) QueryTokenConversion(ctx context.Context, req *types.QueryTokenConversionRequest) (*types.QueryTokenConversionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if _, has := k.factoryDenoms.Get(ctx, req.FullName); !has {
		return nil, types.ErrDenomDoesNotExist
	}

	tokenConversion, has := k.tokenConversions.Get(ctx, req.FullName)
	if !has {
		return nil, types.ErrTokenConversionDoesNotExist
	}

	var conversions []types.TokenConversionCreation
	for _, conversion := range tokenConversion.Entries {
		conversions = append(conversions, types.TokenConversionCreation{
			Denom:            conversion.NewToken,
			ConversationRate: conversion.ConversionRate.String(),
		})
	}

	return &types.QueryTokenConversionResponse{Conversions: conversions}, nil
}
