package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) UpdateFeeAmount(ctx context.Context, req *types.MsgUpdateFeeAmount) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	feeAmount, ok := math.NewIntFromString(req.FeeAmount)
	if !ok {
		return nil, fmt.Errorf("invalid amount")
	}

	params := k.GetParams(ctx)
	params.CreationFee = feeAmount

	if err := params.Validate(); err != nil {
		return nil, err
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}
