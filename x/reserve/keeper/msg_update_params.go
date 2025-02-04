package keeper

import (
	"context"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/reserve/types"
)

func (k msgServer) UpdateKCoinBurnShare(ctx context.Context, req *types.MsgUpdateKCoinBurnShare) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	kCoinBurnShare, err := math.LegacyNewDecFromStr(req.KcoinBurnShare)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.KcoinBurnShare = kCoinBurnShare

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, err
}

func (k msgServer) UpdateBuyThreshold(ctx context.Context, req *types.MsgUpdateBuyThreshold) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	buyThreshold, err := math.LegacyNewDecFromStr(req.BuyThreshold)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.BuyThreshold = buyThreshold

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, err
}

func (k msgServer) UpdateSellThreshold(ctx context.Context, req *types.MsgUpdateSellThreshold) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	sellThreshold, err := math.LegacyNewDecFromStr(req.SellThreshold)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.SellThreshold = sellThreshold

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, err
}
