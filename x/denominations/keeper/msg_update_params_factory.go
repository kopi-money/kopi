package keeper

import (
	"context"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) FactoryAddPoolDenom(ctx context.Context, msg *types.MsgFactoryAddPoolDenom) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	minimumPoolSize, ok := math.NewIntFromString(msg.MinimumPoolSize)
	if !ok {
		return nil, types.ErrInvalidAmount
	}

	moveThreshold, ok := math.NewIntFromString(msg.MoveThreshold)
	if !ok {
		return nil, types.ErrInvalidAmount
	}

	params := k.GetParams(ctx)
	params.FactoryPoolDenoms = append(params.FactoryPoolDenoms, types.FactoryPoolDenom{
		Denom:           msg.Denom,
		MinimumPoolSize: minimumPoolSize,
		MoveThreshold:   moveThreshold,
	})

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) FactoryUpdateMinimumPoolSize(ctx context.Context, msg *types.MsgFactoryUpdateMinimumPoolSize) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	minimumPoolSize, ok := math.NewIntFromString(msg.MinimumPoolSize)
	if !ok {
		return nil, types.ErrInvalidAmount
	}

	var (
		poolDenoms []types.FactoryPoolDenom
		found      bool
	)

	params := k.GetParams(ctx)
	for _, poolDenom := range params.FactoryPoolDenoms {
		if poolDenom.Denom == msg.Denom {
			poolDenom.MinimumPoolSize = minimumPoolSize
			found = true
		}

		poolDenoms = append(poolDenoms, poolDenom)
	}

	if !found {
		return nil, types.ErrInvalidFactoryPoolDenom
	}

	params.FactoryPoolDenoms = poolDenoms

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) FactoryUpdateMoveThreshold(ctx context.Context, msg *types.MsgFactoryUpdateMoveThreshold) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	moveThreshold, ok := math.NewIntFromString(msg.MoveThreshold)
	if !ok {
		return nil, types.ErrInvalidAmount
	}

	var (
		poolDenoms []types.FactoryPoolDenom
		found      bool
	)

	params := k.GetParams(ctx)
	for _, poolDenom := range params.FactoryPoolDenoms {
		if poolDenom.Denom == msg.Denom {
			poolDenom.MoveThreshold = moveThreshold
			found = true
		}

		poolDenoms = append(poolDenoms, poolDenom)
	}

	if !found {
		return nil, types.ErrInvalidFactoryPoolDenom
	}

	params.FactoryPoolDenoms = poolDenoms

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
