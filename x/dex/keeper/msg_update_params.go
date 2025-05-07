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

func (k msgServer) UpdateTradeFee(ctx context.Context, msg *types.MsgUpdateTradeFee) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	tradeFee, err := math.LegacyNewDecFromStr(msg.TradeFee)
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

func (k msgServer) UpdateOrderFee(ctx context.Context, msg *types.MsgUpdateOrderFee) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	orderFee, err := math.LegacyNewDecFromStr(msg.OrderFee)
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

func (k msgServer) UpdateReserveShare(ctx context.Context, msg *types.MsgUpdateReserveShare) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	reserveShare, err := math.LegacyNewDecFromStr(msg.ReserveShare)
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

func (k msgServer) UpdatePriceIncreasingFactor(ctx context.Context, msg *types.MsgUpdatePriceIncreasingFactor) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	priceIncreasingFactor, err := math.LegacyNewDecFromStr(msg.PriceIncreasingFactor)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.PriceIncreasingFactor = priceIncreasingFactor

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateMaxOrderLife(ctx context.Context, msg *types.MsgUpdateMaxOrderLife) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)
	params.MaxOrderLife = msg.MaxOrderLife

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateTradeAmountDecay(ctx context.Context, msg *types.MsgUpdateTradeAmountDecay) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	tradeAmountDecay, err := math.LegacyNewDecFromStr(msg.TradeAmountDecay)
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

func (k msgServer) UpdateDiscountLevels(ctx context.Context, msg *types.MsgUpdateDiscountLevels) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)
	params.DiscountLevels = msg.DiscountLevels

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateEpochLength(ctx context.Context, msg *types.MsgUpdateEpochLength) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	epochLength, err := strconv.ParseUint(msg.EpochLength, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid epoch length: %v", msg.EpochLength)
	}

	params := k.GetParams(ctx)
	params.EpochLength = epochLength

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateLiquidityChangeDecay(ctx context.Context, msg *types.MsgUpdateLiquidityChangeDecay) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	liquidityChangeDecay, err := math.LegacyNewDecFromStr(msg.LiquidityChangeDecay)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.LiquidityChangeDecayFromDeposits = liquidityChangeDecay

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateMinimumLiquidityLock(ctx context.Context, msg *types.MsgUpdateMinimumLiquidityLock) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)
	params.MinimumLiquidityLockInBlocks = msg.MinimumLiquidityLockInBlocks

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) RemoveDexDenom(ctx context.Context, msg *types.MsgRemoveDexDenom) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	if !k.DenomKeeper.IsValidDenom(ctx, msg.Name) {
		return nil, denomtypes.ErrInvalidDexAsset
	}

	if k.DenomKeeper.IsCollateralDenom(ctx, msg.Name) {
		return nil, types.ErrCannotRemoveCollateralDenom
	}

	if err := k.Keeper.RemoveAllLiquidityForDenom(ctx, msg.Name); err != nil {
		return nil, fmt.Errorf("remove liquidity from denom: %w", err)
	}

	if err := k.RemoveDenomOrders(ctx, msg.Name); err != nil {
		return nil, fmt.Errorf("remove orders: %w", err)
	}

	if err := k.DenomKeeper.RemoveDenom(ctx, msg.Name); err != nil {
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
