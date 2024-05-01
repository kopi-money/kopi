package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) CreateDenom(goCtx context.Context, msg *types.MsgCreateDenom) (*types.Void, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	denom := toFullName(msg.Denom)
	if err := k.Keeper.CreateDenom(ctx, denom, msg.Creator); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"denom_created",
			sdk.NewAttribute("denom", denom),
			sdk.NewAttribute("creator", msg.Creator),
		),
	})

	return &types.Void{}, nil
}
