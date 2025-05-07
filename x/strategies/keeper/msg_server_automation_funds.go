package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	"github.com/kopi-money/kopi/x/strategies/types"
)

func (k msgServer) AutomationsAddFunds(ctx context.Context, msg *types.MsgAutomationsAddFunds) (*types.Void, error) {
	amount, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return nil, types.ErrInvalidAmountFormat
	}

	if err := k.depositAutomationFunds(ctx, amount, msg.Creator); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k Keeper) depositAutomationFunds(ctx context.Context, amount math.Int, address string) error {
	if !amount.GT(math.ZeroInt()) {
		return types.ErrZeroAmount
	}

	acc, _ := sdk.AccAddressFromBech32(address)
	coins := sdk.NewCoins(sdk.NewCoin(constants.KUSD, amount))
	if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolAutomationFunds, coins); err != nil {
		return err
	}

	funds := k.GetAutomationFunds(ctx, address)
	funds = funds.Add(amount)
	k.SetAutomationFunds(ctx, address, funds)

	return nil
}

func (k msgServer) AutomationsWithdrawFunds(ctx context.Context, msg *types.MsgAutomationsWidthrawFunds) (*types.Void, error) {
	amount, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return nil, types.ErrInvalidAmountFormat
	}

	if err := k.withdrawAutomationFunds(ctx, amount, msg.Creator); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k Keeper) withdrawAutomationFunds(ctx context.Context, amount math.Int, address string) error {
	if !amount.GT(math.ZeroInt()) {
		return types.ErrZeroAmount
	}

	funds := k.GetAutomationFunds(ctx, address)
	funds = funds.Sub(amount)

	if funds.LT(math.ZeroInt()) {
		return types.ErrAutomationFundsWithdrawlTooLarge
	}

	k.SetAutomationFunds(ctx, address, funds)

	acc, _ := sdk.AccAddressFromBech32(address)

	coins := sdk.NewCoins(sdk.NewCoin(constants.KUSD, amount))
	if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolAutomationFunds, acc, coins); err != nil {
		return fmt.Errorf("send coins from module to account: %w", err)
	}

	return nil
}
