package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/reserve/types"
)

func (k msgServer) UpdateKCoinBurnShare(ctx context.Context, msg *types.MsgUpdateKCoinBurnShare) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	kCoinBurnShare, err := math.LegacyNewDecFromStr(msg.KcoinBurnShare)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.KcoinBurnShare = kCoinBurnShare

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateBuyThreshold(ctx context.Context, msg *types.MsgUpdateBuyThreshold) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	buyThreshold, err := math.LegacyNewDecFromStr(msg.BuyThreshold)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.BuyThreshold = buyThreshold

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateSellThreshold(ctx context.Context, msg *types.MsgUpdateSellThreshold) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	sellThreshold, err := math.LegacyNewDecFromStr(msg.SellThreshold)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.SellThreshold = sellThreshold

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateTradeFeeStakers(ctx context.Context, msg *types.MsgUpdateTradeFeeStakers) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	fromBase, err := math.LegacyNewDecFromStr(msg.TradeFeeBaseIncomeShareToStakers)
	if err != nil {
		return nil, err
	}

	fromOthers, err := math.LegacyNewDecFromStr(msg.TradeFeeOtherIncomeShareToStakers)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.TradeFeeBaseIncomeShareToStakers = fromBase
	params.TradeFeeOtherIncomeShareToStakers = fromOthers

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateReserveSellAmounts(ctx context.Context, msg *types.MsgUpdateReserveSellAmounts) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	var reserveSellAmounts []types.ReserveSellAmount
	for _, rsa := range msg.ReserveSellAmount {
		amount, ok := math.NewIntFromString(rsa.Amount)
		if !ok {
			return nil, fmt.Errorf("invalid amount (%v): %s", rsa.Denom, rsa.Amount)
		}

		reserveSellAmounts = append(reserveSellAmounts, types.ReserveSellAmount{
			Denom:      rsa.Denom,
			SellAmount: amount,
		})
	}

	params := k.GetParams(ctx)
	params.ReserveSellAmounts = reserveSellAmounts

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}
