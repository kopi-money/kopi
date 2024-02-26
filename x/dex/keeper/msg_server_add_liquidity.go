package keeper

import (
	"context"
	sdkerrors "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k msgServer) AddLiquidity(goCtx context.Context, msg *types.MsgAddLiquidity) (*types.MsgAddLiquidityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	amount, err := parseAmount(msg.Amount)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "could not parse amount")
	}

	address, err := k.validateMsg(ctx, msg.Creator, msg.Denom, amount)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "could not validate message")
	}

	if err = k.Keeper.AddLiquidity(ctx, ctx.EventManager(), address, msg.Denom, amount); err != nil {
		return nil, sdkerrors.Wrap(err, "could not add liquidity")
	}

	return &types.MsgAddLiquidityResponse{}, nil
}
