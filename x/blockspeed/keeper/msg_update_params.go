package keeper

import (
	"context"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/blockspeed/types"
)

func (k msgServer) UpdateMovingAverageFactor(ctx context.Context, req *types.MsgUpdateMovingAverageFactor) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	movingAverageFactor, err := math.LegacyNewDecFromStr(req.MovingAverageFactor)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.MovingAverageFactor = movingAverageFactor

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, err
}
