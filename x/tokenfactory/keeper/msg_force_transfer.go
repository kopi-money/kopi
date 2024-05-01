package keeper

import (
	"context"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) ForceTransfer(goCtx context.Context, msg *types.MsgForceTransfer) (*types.Void, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	denom, has := k.GetDenom(ctx, toFullName(msg.Denom))
	if !has {
		return nil, types.ErrDenomDoesntExists
	}

	if denom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	amount, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return nil, types.ErrInvalidAmount
	}

	if !amount.GT(math.ZeroInt()) {
		return nil, types.ErrNonPositiveAmount
	}

	fromAddr, err := sdk.AccAddressFromBech32(msg.TargetAddress)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	targetAddr, err := sdk.AccAddressFromBech32(msg.TargetAddress)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	coins := sdk.NewCoins(sdk.NewCoin(denom.Denom, amount))
	if err = k.BankKeeper.SendCoins(ctx, fromAddr, targetAddr, coins); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"force_transfer",
			sdk.NewAttribute("denom", denom.Denom),
			sdk.NewAttribute("amount", msg.Amount),
			sdk.NewAttribute("from_address", msg.FromAddress),
			sdk.NewAttribute("target_address", msg.TargetAddress),
		),
	})

	return &types.Void{}, nil
}
