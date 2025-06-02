package keeper

import (
	"context"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) FeeDenomsAdd(ctx context.Context, msg *types.MsgFeeDenomsAdd) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	minimumTradeAmount, ok := math.NewIntFromString(msg.MinimumTradeAmount)
	if !ok {
		return nil, types.ErrInvalidAmount
	}

	params := k.GetParams(ctx)
	params.FeeDenoms = append(params.FeeDenoms, types.FeeDenom{
		Denom:              msg.Denom,
		MinimumTradeAmount: minimumTradeAmount,
	})

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
