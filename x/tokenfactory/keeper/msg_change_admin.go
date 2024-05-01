package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) ChangeAdmin(goCtx context.Context, msg *types.MsgChangeAdmin) (*types.Void, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	denom, has := k.GetDenom(ctx, toFullName(msg.Denom))
	if !has {
		return nil, types.ErrDenomDoesntExists
	}

	if denom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	_, err := sdk.AccAddressFromBech32(msg.NewAdmin)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	oldAdmin := denom.Admin
	denom.Admin = msg.NewAdmin

	k.SetDenom(ctx, denom)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"changed_admin",
			sdk.NewAttribute("denom", denom.Denom),
			sdk.NewAttribute("old_admin", oldAdmin),
			sdk.NewAttribute("new_admin", msg.NewAdmin),
		),
	})

	return &types.Void{}, nil
}
