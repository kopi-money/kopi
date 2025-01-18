package keeper

import (
	"context"
	"github.com/kopi-money/kopi/cache"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k msgServer) UpdateTradeFee(ctx context.Context, req *types.MsgUpdateTradeFee) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		tradeFee, err := math.LegacyNewDecFromStr(req.TradeFee)
		if err != nil {
			return err
		}

		params := k.GetParams(innerCtx)
		params.TradeFee = tradeFee

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateOrderFee(ctx context.Context, req *types.MsgUpdateOrderFee) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		orderFee, err := math.LegacyNewDecFromStr(req.OrderFee)
		if err != nil {
			return err
		}

		params := k.GetParams(innerCtx)
		params.OrderFee = orderFee

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateReserveShare(ctx context.Context, req *types.MsgUpdateReserveShare) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		reserveShare, err := math.LegacyNewDecFromStr(req.ReserveShare)
		if err != nil {
			return err
		}

		params := k.GetParams(innerCtx)
		params.ReserveShare = reserveShare

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateVirtualLiquidityDecay(ctx context.Context, req *types.MsgUpdateVirtualLiquidityDecay) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		virtualLiquidityDecay, err := math.LegacyNewDecFromStr(req.VirtualLiquidityDecay)
		if err != nil {
			return err
		}

		params := k.GetParams(innerCtx)
		params.VirtualLiquidityDecay = virtualLiquidityDecay

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateMaxOrderLife(ctx context.Context, req *types.MsgUpdateMaxOrderLife) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)
		params.MaxOrderLife = req.MaxOrderLife

		if err := k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateTradeAmountDecay(ctx context.Context, req *types.MsgUpdateTradeAmountDecay) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		tradeAmountDecay, err := math.LegacyNewDecFromStr(req.TradeAmountDecay)
		if err != nil {
			return err
		}

		params := k.GetParams(innerCtx)
		params.TradeAmountDecay = tradeAmountDecay

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateDiscountLevels(ctx context.Context, req *types.MsgUpdateDiscountLevels) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)
		params.DiscountLevels = req.DiscountLevels

		if err := k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) RemoveDexDenom(ctx context.Context, req *types.MsgRemoveDexDenom) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		if !k.DenomKeeper.IsValidDenom(innerCtx, req.Name) {
			return types.ErrDenomNotFound
		}

		if k.DenomKeeper.IsCollateralDenom(innerCtx, req.Name) {
			return types.ErrCannotRemoveCollateralDenom
		}

		if err := k.Keeper.RemoveAllLiquidityForDenom(innerCtx, req.Name); err != nil {
			return err
		}

		if err := k.DenomKeeper.RemoveDenom(innerCtx, req.Name); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}
