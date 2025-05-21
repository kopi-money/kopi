package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) BurnDenom(ctx context.Context, msg *types.MsgBurnDenom) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, fmt.Errorf("token factory denom not found: %v", msg.FullFactoryDenomName)
	}

	if factoryDenom.LocalName != "" {
		return nil, types.ErrLocalTokenBurn
	}

	amount, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return nil, types.ErrInvalidAmountFormat
	}

	if !amount.IsPositive() {
		return nil, types.ErrNonPositiveAmount
	}

	if err := k.burnDenom(ctx, factoryDenom, amount, msg.Creator); err != nil {
		return nil, fmt.Errorf("burn denom: %w", err)
	}

	return &types.Void{}, nil
}

func (k Keeper) burnDenom(ctx context.Context, factoryDenom types.FactoryDenom, amount math.Int, address string) error {
	addr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return types.ErrInvalidAddress
	}

	coins := sdk.NewCoins(sdk.NewCoin(factoryDenom.FullName, amount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, coins); err != nil {
		return fmt.Errorf("send coins from acc to module: %w", err)
	}

	if err = k.BankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
		return fmt.Errorf("burn coins: %w", err)
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"factory_denom_coins_burned",
			sdk.NewAttribute("factor_denom_full_name", factoryDenom.FullName),
			sdk.NewAttribute("amount", amount.String()),
		),
	})

	return nil
}
