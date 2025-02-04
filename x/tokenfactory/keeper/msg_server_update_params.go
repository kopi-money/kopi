package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) UpdateFeeAmount(ctx context.Context, req *types.MsgUpdateFeeAmount) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	feeAmount, ok := math.NewIntFromString(req.FeeAmount)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %v", req.FeeAmount)
	}

	params := k.GetParams(ctx)
	params.CreationFee = feeAmount

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateMinimumUnlock(ctx context.Context, req *types.MsgUpdateMinimumUnlock) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
	params.MinimumUnlockInSeconds = req.MinimumUnlock

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateReserveFee(ctx context.Context, req *types.MsgUpdateReserveFee) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	reserveFee, err := math.LegacyNewDecFromStr(req.ReserveFee)
	if err != nil {
		return nil, fmt.Errorf("invalid reserve fee: %w", err)
	}

	params := k.GetParams(ctx)
	params.ReserveFee = reserveFee

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, err
}

func (k msgServer) UpdateMinimumPoolSize(ctx context.Context, req *types.MsgUpdateMinimumPoolSize) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	minimumPoolSize, ok := math.NewIntFromString(req.MinimumPoolSize)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %v", req.MinimumPoolSize)
	}

	params := k.GetParams(ctx)
	params.MinimumPoolSize = minimumPoolSize

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}
