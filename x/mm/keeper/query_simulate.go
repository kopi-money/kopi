package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k Keeper) SimulateDeposit(ctx context.Context, req *types.SimulateDepositQuery) (*types.SimulateDepositResponse, error) {
	cAsset, err := k.DenomKeeper.GetCAssetByBaseName(ctx, req.DepositDenom)
	if err != nil {
		return nil, types.ErrInvalidDepositDenom
	}

	amount, ok := math.NewIntFromString(req.DepositAmount)
	if !ok {
		return nil, fmt.Errorf("invalid deposit amount: %s", req.DepositAmount)
	}

	newCAssetTokens, err := k.CalculateNewCAssetAmount(ctx, cAsset, amount)
	if err != nil {
		return nil, err
	}

	if !newCAssetTokens.IsPositive() {
		return nil, types.ErrZeroCAssets
	}

	return &types.SimulateDepositResponse{
		DepositAmount: req.DepositAmount,
		ReceiveAmount: newCAssetTokens.String(),
	}, nil
}

func (k Keeper) SimulateRedemption(ctx context.Context, req *types.SimulateRedemptionQuery) (*types.SimulateRedemptionResponse, error) {
	cAsset, err := k.DenomKeeper.GetCAssetByBaseName(ctx, req.RedemptionDenom)
	if err != nil {
		return nil, denomtypes.ErrInvalidCAsset
	}

	amount, ok := math.NewIntFromString(req.RedemptionAmount)
	if !ok {
		return nil, fmt.Errorf("invalid deposit amount: %s", req.RedemptionAmount)
	}

	moduleAccount := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
	available := k.BankKeeper.SpendableCoins(ctx, moduleAccount.GetAddress()).AmountOf(cAsset.BaseDexDenom)

	grossRedemptionAmountBase, redemptionAmountCAsset := k.CalculateAvailableRedemptionAmount(ctx, cAsset, amount.ToLegacyDec(), available.ToLegacyDec())
	maximumRedemptionAmount := k.CalculateRedemptionAmount(ctx, cAsset, amount.ToLegacyDec())

	return &types.SimulateRedemptionResponse{
		VaultSize:               available.String(),
		RedemptionAmount:        redemptionAmountCAsset.String(),
		AmountReceived:          grossRedemptionAmountBase.String(),
		MaximumRedemptionAmount: maximumRedemptionAmount.String(),
	}, nil
}
