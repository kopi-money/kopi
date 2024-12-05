package keeper

import (
	"context"
	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/cache"

	"github.com/kopi-money/kopi/x/ls/types"
)

func (k msgServer) UpdateParams(ctx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.SetParams(innerCtx, req.Params)
	})

	return &types.MsgUpdateParamsResponse{}, err
}
