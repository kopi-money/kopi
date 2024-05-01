package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k msgServer) CreateRedemptionRequest(goCtx context.Context, msg *types.MsgCreateRedemptionRequest) (*types.Void, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	cAsset, err := k.DenomKeeper.GetCAssetByName(ctx, msg.Denom)
	if err != nil {
		return nil, err
	}

	cAssetAmount, err := parseAmount(msg.CAssetAmount, false)
	if err != nil {
		return nil, err
	}

	address, _ := sdk.AccAddressFromBech32(msg.Creator)

	if err = k.checkSpendableCoins(ctx, address, cAsset.Name, cAssetAmount); err != nil {
		return nil, err
	}

	coins := sdk.NewCoins(sdk.NewCoin(cAsset.Name, cAssetAmount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, address, types.PoolRedemption, coins); err != nil {
		return nil, err
	}

	fee, err := k.checkFee(ctx, msg.Fee)
	if err != nil {
		return nil, err
	}

	_, has := k.GetRedemption(ctx, cAsset.BaseDenom, msg.Creator)
	if has {
		return nil, types.ErrRedemptionRequestAlreadyExists

	}

	redemption := types.Redemption{
		AddedAt: ctx.BlockHeight(),
		Address: msg.Creator,
		Amount:  cAssetAmount,
		Fee:     fee,
	}

	k.SetRedemption(ctx, cAsset.Name, redemption)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("redemption_request_created",
			sdk.Attribute{Key: "address", Value: msg.Creator},
			sdk.Attribute{Key: "denom", Value: msg.Denom},
			sdk.Attribute{Key: "amount", Value: msg.CAssetAmount},
			sdk.Attribute{Key: "fee", Value: msg.Fee},
		),
	)

	return &types.Void{}, nil
}

func (k msgServer) CancelRedemptionRequest(goCtx context.Context, msg *types.MsgCancelRedemptionRequest) (*types.Void, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	cAsset, err := k.DenomKeeper.GetCAssetByName(ctx, msg.Denom)
	if err != nil {
		return nil, err
	}

	redemption, has := k.GetRedemption(ctx, msg.Denom, msg.Creator)
	if !has {
		return nil, types.ErrRedemptionRequestNotFound
	}

	address, _ := sdk.AccAddressFromBech32(msg.Creator)
	coins := sdk.NewCoins(sdk.NewCoin(cAsset.Name, redemption.Amount))
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolRedemption, address, coins); err != nil {
		return nil, err
	}

	k.RemoveRedemption(ctx, msg.Denom, msg.Creator)

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
		return nil, errors.Wrap(err, "invalid fee")
	}

	cAsset, err := k.DenomKeeper.GetCAssetByName(ctx, msg.Denom)
	if err != nil {
		return nil, errors.Wrap(err, "invalid cAsset denom")
	}

	cAssetAmount, err := parseAmount(msg.CAssetAmount, false)
	if err != nil {
		return nil, errors.Wrap(err, "invalid cAsset amount")
	}

	redemption, has := k.GetRedemption(ctx, msg.Denom, msg.Creator)
	if !has {
		return nil, types.ErrRedemptionRequestNotFound
	}

	address, _ := sdk.AccAddressFromBech32(msg.Creator)
	coins := sdk.NewCoins(sdk.NewCoin(cAsset.Name, redemption.Amount))
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolRedemption, address, coins); err != nil {
		return nil, errors.Wrap(err, "could not send coins from redemption pool to user")
	}

	if err = k.checkSpendableCoins(ctx, address, cAsset.Name, cAssetAmount); err != nil {
		return nil, err
	}

	coins = sdk.NewCoins(sdk.NewCoin(cAsset.Name, cAssetAmount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, address, types.PoolRedemption, coins); err != nil {
		return nil, errors.Wrap(err, "could not send coins from user to redemption pool")
	}

	redemption.Fee = fee
	redemption.Amount = cAssetAmount

	k.SetRedemption(ctx, msg.Denom, redemption)

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

func (k Keeper) checkFee(ctx context.Context, priorityStr string) (math.LegacyDec, error) {
	minimumFee := k.GetParams(ctx).MinRedemptionFee

	priority, err := math.LegacyNewDecFromStr(priorityStr)
	if err != nil {
		return priority, err
	}

	if priority.LT(minimumFee) {
		return priority, types.ErrRedemptionFeeTooLow
	}

	if priority.GT(math.LegacyOneDec()) {
		return priority, types.ErrRedemptionFeeTooHigh
	}

	return priority, nil
}
