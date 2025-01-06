package keeper

import (
	"context"
	"fmt"

	"github.com/kopi-money/kopi/cache"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) UpdateFeeAmount(ctx context.Context, req *types.MsgUpdateFeeAmount) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		feeAmount, ok := math.NewIntFromString(req.FeeAmount)
		if !ok {
			return fmt.Errorf("invalid amount: %v", req.FeeAmount)
		}

		params := k.GetParams(innerCtx)
		params.CreationFee = feeAmount

		if err := k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateMinimumUnlock(ctx context.Context, req *types.MsgUpdateMinimumUnlock) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)
		params.MinimumUnlockInSeconds = req.MinimumUnlock

		if err := k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateReserveFee(ctx context.Context, req *types.MsgUpdateReserveFee) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		reserveFee, err := math.LegacyNewDecFromStr(req.ReserveFee)
		if err != nil {
			return fmt.Errorf("invalid reserve fee: %w", err)
		}

		params := k.GetParams(innerCtx)
		params.ReserveFee = reserveFee

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateMinimumPoolSize(ctx context.Context, req *types.MsgUpdateMinimumPoolSize) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		minimumPoolSize, ok := math.NewIntFromString(req.MinimumPoolSize)
		if !ok {
			return fmt.Errorf("invalid amount: %v", req.MinimumPoolSize)
		}

		params := k.GetParams(innerCtx)
		params.MinimumPoolSize = minimumPoolSize

		if err := k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}
