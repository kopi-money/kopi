package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/strategies/types"
)

func (k Keeper) ArbitrageSimulateDepositBase(ctx context.Context, req *types.ArbitrageSimulateDepositRequest) (*types.ArbitrageSimulateDepositResponse, error) {
	aAsset, err := k.DenomKeeper.GetArbitrageDenomByName(ctx, req.AAssetDenom)
	if err != nil {
		return nil, err
	}

	amount, ok := math.NewIntFromString(req.DepositAmount)
	if !ok {
		return nil, fmt.Errorf("invalid deposit amount: %s", req.DepositAmount)
	}

	cAsset, err := k.DenomKeeper.GetCAsset(ctx, aAsset.CAsset)
	if err != nil {
		return nil, err
	}

	cAssetAmount, err := k.MMKeeper.CalculateNewCAssetAmount(ctx, cAsset, amount)
	if err != nil {
		return nil, err
	}

	calculateValue := k.calculateArbitrageTokenValue(ctx, aAsset)
	calculateValue = append(calculateValue, func() (math.LegacyDec, error) {
		return cAssetAmount.ToLegacyDec(), nil
	})

	newTokens, err := k.calculateNewStrategyAssetAmount(ctx, aAsset.DexDenom, cAssetAmount, calculateValue)
	if err != nil {
		return nil, fmt.Errorf("could not calculate new strategy asset amount: %w", err)
	}

	return &types.ArbitrageSimulateDepositResponse{
		DepositAmount: req.DepositAmount,
		ReceiveAmount: newTokens.String(),
	}, nil
}

func (k Keeper) ArbitrageSimulateDepositCAsset(ctx context.Context, req *types.ArbitrageSimulateDepositRequest) (*types.ArbitrageSimulateDepositResponse, error) {
	aAsset, err := k.DenomKeeper.GetArbitrageDenomByName(ctx, req.AAssetDenom)
	if err != nil {
		return nil, err
	}

	amount, ok := math.NewIntFromString(req.DepositAmount)
	if !ok {
		return nil, fmt.Errorf("invalid deposit amount: %s", req.DepositAmount)
	}

	calculateValue := k.calculateArbitrageTokenValue(ctx, aAsset)
	calculateValue = append(calculateValue, func() (math.LegacyDec, error) {
		return amount.ToLegacyDec(), nil
	})

	newTokens, err := k.calculateNewStrategyAssetAmount(ctx, aAsset.DexDenom, amount, calculateValue)
	if err != nil {
		return nil, fmt.Errorf("could not calculate new strategy asset amount: %w", err)
	}

	return &types.ArbitrageSimulateDepositResponse{
		DepositAmount: req.DepositAmount,
		ReceiveAmount: newTokens.String(),
	}, nil
}

func (k Keeper) ArbitrageSimulateRedemption(ctx context.Context, req *types.ArbitrageSimulateRedemptionRequest) (*types.ArbitrageSimulateRedemptionResponse, error) {
	aAsset, err := k.DenomKeeper.GetArbitrageDenomByName(ctx, req.AAssetDenom)
	if err != nil {
		return nil, err
	}

	amount, ok := math.NewIntFromString(req.RedemptionAmount)
	if !ok {
		return nil, fmt.Errorf("invalid deposit amount: %s", req.RedemptionAmount)
	}

	calculateValue := k.calculateArbitrageTokenValue(ctx, aAsset)
	redemptionValue, err := k.calculateRedemptionValue(ctx, aAsset, amount, calculateValue)
	if err != nil {
		return nil, fmt.Errorf("could not calculate redemption amount: %w", err)
	}

	return &types.ArbitrageSimulateRedemptionResponse{
		AmountReceived: redemptionValue.String(),
	}, nil
}
