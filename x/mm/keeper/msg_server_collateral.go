package keeper

import (
	"context"
	"cosmossdk.io/collections"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k msgServer) AddCollateral(goCtx context.Context, msg *types.MsgAddCollateral) (*types.Void, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.DenomKeeper.IsValidCollateralDenom(ctx, msg.Denom) {
		return nil, types.ErrInvalidCollateralDenom
	}

	amount, err := parseAmount(msg.Amount, false)
	if err != nil {
		return nil, err
	}

	if err = k.checkSupplyCap(ctx, msg.Denom, amount); err != nil {
		return nil, err
	}

	address, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	if err = k.checkSpendableCoins(ctx, address, msg.Denom, amount); err != nil {
		return nil, err
	}

	key := collections.Join(msg.Denom, msg.Creator)
	collateral, found := k.collateral.Get(ctx, key)
	if !found {
		collateral = types.Collateral{Address: msg.Creator, Amount: math.ZeroInt()}
	}

	newAmount := collateral.Amount.Add(amount)
	k.SetCollateral(ctx, msg.Denom, msg.Creator, newAmount)

	coins := sdk.NewCoins(sdk.NewCoin(msg.Denom, amount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, address, types.PoolCollateral, coins); err != nil {
		return nil, errors.Wrap(err, "could not send coins to module")
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("collateral_added",
			sdk.Attribute{Key: "address", Value: msg.Creator},
			sdk.Attribute{Key: "denom", Value: msg.Denom},
			sdk.Attribute{Key: "amount", Value: msg.Amount},
		),
	)

	return &types.Void{}, nil
}

func (k msgServer) RemoveCollateral(goCtx context.Context, msg *types.MsgRemoveCollateral) (*types.Void, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.DenomKeeper.IsValidCollateralDenom(ctx, msg.Denom) {
		return nil, types.ErrInvalidDepositDenom
	}

	amount, err := parseAmount(msg.Amount, false)
	if err != nil {
		return nil, err
	}

	address, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	key := collections.Join(msg.Denom, msg.Creator)
	collateral, found := k.collateral.Get(ctx, key)
	if !found {
		return nil, types.ErrNoCollateralFound
	}

	newAmount := collateral.Amount.Sub(amount)
	if newAmount.LT(math.ZeroInt()) {
		return nil, types.ErrNegativeCollateral
	}

	k.SetCollateral(ctx, msg.Denom, msg.Creator, newAmount)

	coins := sdk.NewCoins(sdk.NewCoin(msg.Denom, amount))
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolCollateral, address, coins); err != nil {
		return nil, errors.Wrap(err, "could not send coins to user wallet")
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("collateral_removed",
			sdk.Attribute{Key: "address", Value: msg.Creator},
			sdk.Attribute{Key: "denom", Value: msg.Denom},
			sdk.Attribute{Key: "amount", Value: msg.Amount},
		),
	)

	return &types.Void{}, nil
}
