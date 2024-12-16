package keeper

import (
	"context"
	"strings"

	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) UpdateIconHash(ctx context.Context, msg *types.MsgUpdateIconHash) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	factoryDenom.IconHash = strings.ToUpper(msg.IconHash)

	k.SetDenom(ctx, factoryDenom)

	return &types.Void{}, nil
}
