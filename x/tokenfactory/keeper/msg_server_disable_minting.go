package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) DisableMinting(ctx context.Context, msg *types.MsgDisableMinting) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExists
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	factoryDenom.Mintable = false

	k.SetDenom(ctx, factoryDenom)

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"factory_denom_disabled_minting",
			sdk.NewAttribute("full_name", factoryDenom.FullName),
		),
	})

	return &types.Void{}, nil
}
