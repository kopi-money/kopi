package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) ChangeAdmin(ctx context.Context, msg *types.MsgChangeAdmin) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExists
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	if _, err := sdk.AccAddressFromBech32(msg.NewAdmin); err != nil {
		return nil, types.ErrInvalidAddress
	}

	oldAdmin := factoryDenom.Admin
	factoryDenom.Admin = msg.NewAdmin

	k.SetDenom(ctx, factoryDenom)

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"factory_denom_admin_change",
			sdk.NewAttribute("full_name", factoryDenom.FullName),
			sdk.NewAttribute("old_admin", oldAdmin),
			sdk.NewAttribute("new_admin", msg.NewAdmin),
		),
	})

	return &types.Void{}, nil
}
