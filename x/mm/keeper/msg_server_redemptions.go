package keeper

import (
	"context"
	"fmt"

	denomtypes "github.com/kopi-money/kopi/x/denominations/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k msgServer) CreateRedemptionRequest(ctx context.Context, msg *types.MsgCreateRedemptionRequest) (*types.Void, error) {
	cAsset, err := k.DenomKeeper.GetCAssetByName(ctx, msg.Denom)
	if err != nil {
		return nil, err
	}

	cAssetAmount, err := parseAmount(msg.CAssetAmount, false)
	if err != nil {
		return nil, err
	}

	address, _ := sdk.AccAddressFromBech32(msg.Creator)

	fee, err := k.checkFee(ctx, msg.Fee)
	if err != nil {
		return nil, err
	}

	if err = k.Keeper.CreateRedemptionRequest(ctx, address, cAsset, cAssetAmount, fee); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k Keeper) CreateRedemptionRequest(ctx context.Context, address sdk.AccAddress, cAsset denomtypes.CAsset, amount math.Int, fee math.LegacyDec) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	_, has := k.redemptions.Get(ctx, cAsset.BaseDexDenom, address.String())
	if has {
		return types.ErrRedemptionRequestAlreadyExists

	}

	spendable := k.BankKeeper.SpendableCoin(ctx, address, cAsset.DexDenom).Amount
	if spendable.LT(amount) {
		k.Logger().Info(fmt.Sprintf("%v < %v", spendable.Int64(), amount.Int64()))
		return types.ErrNotEnoughFunds
	}

	coins := sdk.NewCoins(sdk.NewCoin(cAsset.DexDenom, amount))
	if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, address, types.PoolRedemption, coins); err != nil {
		return fmt.Errorf("could not send coins from account to module: %w", err)
	}

	redemption := types.Redemption{
		AddedAt: sdkCtx.BlockHeight(),
		Address: address.String(),
		Amount:  amount,
		Fee:     fee,
	}

	if err := k.SetRedemption(ctx, cAsset.BaseDexDenom, redemption); err != nil {
		return fmt.Errorf("could not set redemption request: %w", err)
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent("redemption_request_created",
			sdk.Attribute{Key: "address", Value: address.String()},
			sdk.Attribute{Key: "denom", Value: cAsset.BaseDexDenom},
			sdk.Attribute{Key: "amount", Value: amount.String()},
			sdk.Attribute{Key: "fee", Value: fee.String()},
		),
	)

	return nil
}

func (k msgServer) CancelRedemptionRequest(goCtx context.Context, msg *types.MsgCancelRedemptionRequest) (*types.Void, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	cAsset, err := k.DenomKeeper.GetCAssetByName(ctx, msg.Denom)
	if err != nil {
		return nil, err
	}

	redemption, has := k.redemptions.Get(ctx, cAsset.BaseDexDenom, msg.Creator)
	if !has {
		return nil, types.ErrRedemptionRequestNotFound
	}

	address, _ := sdk.AccAddressFromBech32(msg.Creator)
	coins := sdk.NewCoins(sdk.NewCoin(cAsset.DexDenom, redemption.Amount))
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolRedemption, address, coins); err != nil {
		return nil, err
	}

	k.redemptions.Remove(ctx, cAsset.BaseDexDenom, msg.Creator)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("redemption_request_canceled",
			sdk.Attribute{Key: "address", Value: msg.Creator},
			sdk.Attribute{Key: "denom", Value: msg.Denom},
		),
	)

	return &types.Void{}, nil
}

func (k msgServer) UpdateRedemptionRequest(goCtx context.Context, msg *types.MsgUpdateRedemptionRequest) (*types.Void, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	fee, err := k.checkFee(ctx, msg.Fee)
	if err != nil {
		return nil, fmt.Errorf("invalid fee: %w", err)
	}

	cAsset, err := k.DenomKeeper.GetCAssetByName(ctx, msg.Denom)
	if err != nil {
		return nil, fmt.Errorf("invalid cAsset denom: %w", err)
	}

	cAssetAmount, err := parseAmount(msg.CAssetAmount, false)
	if err != nil {
		return nil, fmt.Errorf("invalid cAsset amount: %w", err)
	}

	redemption, has := k.redemptions.Get(ctx, cAsset.BaseDexDenom, msg.Creator)
	if !has {
		return nil, types.ErrRedemptionRequestNotFound
	}

	address, _ := sdk.AccAddressFromBech32(msg.Creator)
	coins := sdk.NewCoins(sdk.NewCoin(cAsset.DexDenom, redemption.Amount))
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolRedemption, address, coins); err != nil {
		return nil, fmt.Errorf("could not send coins from redemption pool to user: %w", err)
	}

	if k.BankKeeper.SpendableCoin(ctx, address, cAsset.DexDenom).Amount.LT(cAssetAmount) {
		return nil, types.ErrNotEnoughFunds
	}

	coins = sdk.NewCoins(sdk.NewCoin(cAsset.DexDenom, cAssetAmount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, address, types.PoolRedemption, coins); err != nil {
		return nil, fmt.Errorf("could not send coins from user to redemption pool: %w", err)
	}

	redemption.Fee = fee
	redemption.Amount = cAssetAmount

	if err = k.SetRedemption(ctx, cAsset.BaseDexDenom, redemption); err != nil {
		return nil, fmt.Errorf("could not set redemption: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("redemption_request_updated",
			sdk.Attribute{Key: "address", Value: msg.Creator},
			sdk.Attribute{Key: "denom", Value: msg.Denom},
			sdk.Attribute{Key: "amount", Value: msg.CAssetAmount},
			sdk.Attribute{Key: "fee", Value: msg.Fee},
		),
	)

	return &types.Void{}, nil
}

func (k Keeper) GetMinimumRedemptionFee(ctx context.Context) math.LegacyDec {
	return k.GetParams(ctx).MinRedemptionFee
}

func (k Keeper) GetMaximumRedemptionFee(ctx context.Context) math.LegacyDec {
	return k.GetParams(ctx).MaxRedemptionFee
}

func (k Keeper) checkFee(ctx context.Context, priorityStr string) (math.LegacyDec, error) {
	priority, err := math.LegacyNewDecFromStr(priorityStr)
	if err != nil {
		return priority, err
	}

	minimumFee := k.GetMinimumRedemptionFee(ctx)
	if priority.LT(minimumFee) {
		return priority, types.ErrRedemptionFeeTooLow
	}

	maximumFee := k.GetMaximumRedemptionFee(ctx)
	if priority.GT(maximumFee) {
		return priority, types.ErrRedemptionFeeTooHigh
	}

	return priority, nil
}
