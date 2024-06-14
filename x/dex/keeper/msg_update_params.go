package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/cache"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k msgServer) UpdateTradeFee(goCtx context.Context, req *types.MsgUpdateTradeFee) (*types.Void, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		tradeFee, err := math.LegacyNewDecFromStr(req.TradeFee)
		if err != nil {
			return err
		}

		params := k.GetParams(ctx)
		params.TradeFee = tradeFee

		if err = k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateOrderFee(goCtx context.Context, req *types.MsgUpdateOrderFee) (*types.Void, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		orderFee, err := math.LegacyNewDecFromStr(req.OrderFee)
		if err != nil {
			return err
		}

		params := k.GetParams(ctx)
		params.OrderFee = orderFee

		if err = k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateReserveShare(goCtx context.Context, req *types.MsgUpdateReserveShare) (*types.Void, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		reserveShare, err := math.LegacyNewDecFromStr(req.ReserveShare)
		if err != nil {
			return err
		}

		params := k.GetParams(ctx)
		params.ReserveShare = reserveShare

		if err = k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateVirtualLiquidityDecay(goCtx context.Context, req *types.MsgUpdateVirtualLiquidityDecay) (*types.Void, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		virtualLiquidityDecay, err := math.LegacyNewDecFromStr(req.VirtualLiquidityDecay)
		if err != nil {
			return err
		}

		params := k.GetParams(ctx)
		params.VirtualLiquidityDecay = virtualLiquidityDecay

		if err = k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateFeeReimbursement(goCtx context.Context, req *types.MsgUpdateFeeReimbursement) (*types.Void, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		feeReimbursement, err := math.LegacyNewDecFromStr(req.FeeReimbursement)
		if err != nil {
			return err
		}

		params := k.GetParams(ctx)
		params.FeeReimbursement = feeReimbursement

		if err = k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateMaxOrderLife(goCtx context.Context, req *types.MsgUpdateMaxOrderLife) (*types.Void, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)
		params.MaxOrderLife = req.MaxOrderLife

		if err := k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateTradeAmountDecay(goCtx context.Context, req *types.MsgUpdateTradeAmountDecay) (*types.Void, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		tradeAmountDecay, err := math.LegacyNewDecFromStr(req.TradeAmountDecay)
		if err != nil {
			return err
		}

		params := k.GetParams(ctx)
		params.TradeAmountDecay = tradeAmountDecay

		if err = k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateDiscountLevels(goCtx context.Context, req *types.MsgUpdateDiscountLevels) (*types.Void, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)
		params.DiscountLevels = req.DiscountLevels

		if err := k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}
