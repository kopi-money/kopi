package keeper

import (
	"context"
	"fmt"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"strconv"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k msgServer) UpdateTradeFee(ctx context.Context, req *types.MsgUpdateTradeFee) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

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

func (k msgServer) UpdateOrderFee(ctx context.Context, req *types.MsgUpdateOrderFee) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	orderFee, err := math.LegacyNewDecFromStr(req.OrderFee)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.OrderFee = orderFee

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateReserveShare(ctx context.Context, req *types.MsgUpdateReserveShare) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

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

func (k msgServer) UpdateVirtualLiquidityDecay(ctx context.Context, req *types.MsgUpdateVirtualLiquidityDecay) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

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

func (k msgServer) UpdateMaxOrderLife(ctx context.Context, req *types.MsgUpdateMaxOrderLife) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
	params.MaxOrderLife = req.MaxOrderLife

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateTradeAmountDecay(ctx context.Context, req *types.MsgUpdateTradeAmountDecay) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

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

func (k msgServer) UpdateDiscountLevels(ctx context.Context, req *types.MsgUpdateDiscountLevels) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
	params.DiscountLevels = req.DiscountLevels

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateEpochLength(ctx context.Context, req *types.MsgUpdateEpochLength) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	epochLength, err := strconv.ParseUint(req.EpochLength, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid epoch length: %v", req.EpochLength)
	}

	params := k.GetParams(ctx)
	params.EpochLength = epochLength

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, err
}

func (k msgServer) RemoveDexDenom(ctx context.Context, req *types.MsgRemoveDexDenom) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	if !k.DenomKeeper.IsValidDenom(ctx, req.Name) {
		return nil, denomtypes.ErrInvalidDexAsset
	}

	if k.DenomKeeper.IsCollateralDenom(ctx, req.Name) {
		return nil, types.ErrCannotRemoveCollateralDenom
	}

	if err := k.Keeper.RemoveAllLiquidityForDenom(ctx, req.Name); err != nil {
		return nil, fmt.Errorf("remove liquidity from denom: %w", err)
	}

	if err := k.RemoveDenomOrders(ctx, req.Name); err != nil {
		return nil, fmt.Errorf("remove orders: %w", err)
	}

	if err := k.DenomKeeper.RemoveDenom(ctx, req.Name); err != nil {
		return nil, fmt.Errorf("remove denom: %w", err)
	}

	return &types.Void{}, nil
}

func (k Keeper) RemoveDenomOrders(ctx context.Context, denom string) error {
	iterator := k.orders.Iterator(ctx, nil)
	for iterator.Valid() {
		order := iterator.GetNext()

		if order.DenomGiving == denom || order.DenomReceiving == denom {
			if err := k.RemoveOrder(ctx, order); err != nil {
				return fmt.Errorf("RemoveOrder: %w", err)
			}
		}
	}

	return nil
}
