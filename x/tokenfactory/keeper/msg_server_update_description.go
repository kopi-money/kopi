package keeper

import (
	"context"
	"fmt"
	"github.com/kopi-money/kopi/constants"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) UpdateDescription(ctx context.Context, msg *types.MsgUpdateDescription) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	if len(msg.Description) > constants.MaxDescriptionLength {
		return nil, fmt.Errorf("description too long")
	}

	factoryDenom.Description = msg.Description

	k.SetDenom(ctx, factoryDenom)

	return &types.Void{}, nil
}
