package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k msgServer) AddCollateral(ctx context.Context, msg *types.MsgAddCollateral) (*types.Void, error) {
	amount, err := parseAmount(msg.Amount, false)
	if err != nil {
		return nil, err
	}

	address, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	if _, err = k.Keeper.AddCollateral(ctx, address, address, msg.Denom, amount); err != nil {
		return nil, fmt.Errorf("add collateral: %w", err)
	}

	return &types.Void{}, nil
}

func (k msgServer) AddCollateralForBeneficiary(ctx context.Context, msg *types.MsgAddCollateralForBeneficiary) (*types.Void, error) {
	amount, err := parseAmount(msg.Amount, false)
	if err != nil {
		return nil, err
	}

	addressPayer, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	addressBeneficiary, err := sdk.AccAddressFromBech32(msg.Beneficiary)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	if _, err = k.Keeper.AddCollateral(ctx, addressPayer, addressBeneficiary, msg.Denom, amount); err != nil {
		return nil, fmt.Errorf("add collateral: %w", err)
	}

	return &types.Void{}, nil
}

func (k Keeper) AddCollateral(ctx context.Context, addressPayee, addressBeneficiary sdk.AccAddress, denom string, amount math.Int) (math.Int, error) {
	if !k.DenomKeeper.IsValidCollateralDenom(ctx, denom) {
		return math.Int{}, types.ErrInvalidCollateralDenom
	}

	if amount.IsZero() {
		return math.Int{}, types.ErrZeroAmount
	}

	if amount.LT(math.ZeroInt()) {
		return math.Int{}, types.ErrNegativeAmount
	}

	if err := k.checkSupplyCap(ctx, denom, amount); err != nil {
		return math.Int{}, err
	}

	if k.BankKeeper.SpendableCoin(ctx, addressPayee, denom).Amount.LT(amount) {
		return math.Int{}, types.ErrNotEnoughFunds
	}

	collateral, found := k.collateral.Get(ctx, denom, addressBeneficiary.String())
	if !found {
		collateral = types.Collateral{Address: addressBeneficiary.String(), Amount: math.ZeroInt()}
	}

	newAmount := collateral.Amount.Add(amount)
	k.SetCollateral(ctx, denom, addressBeneficiary.String(), newAmount)

	coins := sdk.NewCoins(sdk.NewCoin(denom, amount))
	if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, addressPayee, types.PoolCollateral, coins); err != nil {
		return math.Int{}, fmt.Errorf("send coins to module: %w", err)
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("collateral_added",
			sdk.Attribute{Key: "address", Value: addressBeneficiary.String()},
			sdk.Attribute{Key: "denom", Value: denom},
			sdk.Attribute{Key: "amount", Value: amount.String()},
		),
	)

	return newAmount, nil
}

func (k msgServer) RemoveCollateral(ctx context.Context, msg *types.MsgRemoveCollateral) (*types.Void, error) {
	amount, err := parseAmount(msg.Amount, false)
	if err != nil {
		return nil, err
	}

	address, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	if _, err = k.WithdrawCollateral(ctx, address, msg.Denom, amount); err != nil {
		return nil, fmt.Errorf("withdraw collateral: %w", err)
	}

	return &types.Void{}, nil
}

func (k Keeper) WithdrawCollateral(ctx context.Context, address sdk.AccAddress, denom string, amount math.Int) (math.Int, error) {
	if !k.DenomKeeper.IsValidCollateralDenom(ctx, denom) {
		return math.Int{}, types.ErrInvalidCollateralDenom
	}

	withdrawableAmount, err := k.CalcWithdrawableCollateralAmount(ctx, address.String(), denom)
	if err != nil {
		return math.Int{}, fmt.Errorf("calculate withdrawable amount: %w", err)
	}

	if amount.ToLegacyDec().GT(withdrawableAmount) {
		return math.Int{}, types.ErrCannotWithdrawCollateral
	}

	collateral, found := k.collateral.Get(ctx, denom, address.String())
	if !found {
		return math.Int{}, types.ErrNoCollateralFound
	}

	amount = math.MinInt(collateral.Amount, amount)
	newAmount := collateral.Amount.Sub(amount)

	k.SetCollateral(ctx, denom, address.String(), newAmount)

	coins := sdk.NewCoins(sdk.NewCoin(denom, amount))
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolCollateral, address, coins); err != nil {
		return math.Int{}, fmt.Errorf("send coins to user wallet: %w", err)
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("collateral_removed",
			sdk.Attribute{Key: "address", Value: address.String()},
			sdk.Attribute{Key: "denom", Value: denom},
			sdk.Attribute{Key: "amount", Value: amount.String()},
		),
	)

	return newAmount, nil
}
