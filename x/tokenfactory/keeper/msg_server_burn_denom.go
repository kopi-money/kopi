package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) BurnDenom(ctx context.Context, msg *types.MsgBurnDenom) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExists
	}

	amount, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return nil, types.ErrInvalidAmountFormat
	}

	if !amount.GT(math.ZeroInt()) {
		return nil, types.ErrNonPositiveAmount
	}

	addr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	coins := sdk.NewCoins(sdk.NewCoin(factoryDenom.FullName, amount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, coins); err != nil {
		return nil, err
	}

	if err = k.BankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
		return nil, err
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"factory_denom_coins_burned",
			sdk.NewAttribute("factor_denom_full_name", factoryDenom.FullName),
			sdk.NewAttribute("amount", msg.Amount),
		),
	})

	return &types.Void{}, nil
}
