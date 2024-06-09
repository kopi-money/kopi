package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k msgServer) UpdateTradeFee(goCtx context.Context, req *types.MsgUpdateTradeFee) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := startTX(sdk.UnwrapSDKContext(goCtx))
	defer k.CommitToCache(ctx)
	defer k.CommitToDB(ctx)

	tradeFee, err := math.LegacyNewDecFromStr(req.TradeFee)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.TradeFee = tradeFee

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateReserveShare(goCtx context.Context, req *types.MsgUpdateReserveShare) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := startTX(sdk.UnwrapSDKContext(goCtx))
	defer k.CommitToCache(ctx)
	defer k.CommitToDB(ctx)

	reserveShare, err := math.LegacyNewDecFromStr(req.ReserveShare)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.ReserveShare = reserveShare

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateVirtualLiquidityDecay(goCtx context.Context, req *types.MsgUpdateVirtualLiquidityDecay) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := startTX(sdk.UnwrapSDKContext(goCtx))
	defer k.CommitToCache(ctx)
	defer k.CommitToDB(ctx)

	virtualLiquidityDecay, err := math.LegacyNewDecFromStr(req.VirtualLiquidityDecay)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.VirtualLiquidityDecay = virtualLiquidityDecay

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateFeeReimbursement(goCtx context.Context, req *types.MsgUpdateFeeReimbursement) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := startTX(sdk.UnwrapSDKContext(goCtx))
	defer k.CommitToCache(ctx)
	defer k.CommitToDB(ctx)

	feeReimbursement, err := math.LegacyNewDecFromStr(req.FeeReimbursement)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.FeeReimbursement = feeReimbursement

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateMaxOrderLife(goCtx context.Context, req *types.MsgUpdateMaxOrderLife) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := startTX(sdk.UnwrapSDKContext(goCtx))
	defer k.CommitToCache(ctx)
	defer k.CommitToDB(ctx)

	params := k.GetParams(ctx)
	params.MaxOrderLife = req.MaxOrderLife

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateTradeAmountDecay(goCtx context.Context, req *types.MsgUpdateTradeAmountDecay) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := startTX(sdk.UnwrapSDKContext(goCtx))
	defer k.CommitToCache(ctx)
	defer k.CommitToDB(ctx)

	tradeAmountDecay, err := math.LegacyNewDecFromStr(req.TradeAmountDecay)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.TradeAmountDecay = tradeAmountDecay

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateDiscountLevels(goCtx context.Context, req *types.MsgUpdateDiscountLevels) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := startTX(sdk.UnwrapSDKContext(goCtx))
	defer k.CommitToCache(ctx)
	defer k.CommitToDB(ctx)

	params := k.GetParams(ctx)
	params.DiscountLevels = req.DiscountLevels

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}
