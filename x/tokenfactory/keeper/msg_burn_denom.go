package keeper

import (
	"context"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) BurnDenom(goCtx context.Context, msg *types.MsgBurnDenom) (*types.Void, error) {
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

	targetAddr, err := sdk.AccAddressFromBech32(msg.TargetAddress)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	coins := sdk.NewCoins(sdk.NewCoin(denom.Denom, amount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, targetAddr, types.ModuleName, coins); err != nil {
		return nil, err
	}

	if err = k.BankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"coins_burned",
			sdk.NewAttribute("denom", denom.Denom),
			sdk.NewAttribute("amount", msg.Amount),
			sdk.NewAttribute("target_address", msg.TargetAddress),
		),
	})

	return &types.Void{}, nil
}
