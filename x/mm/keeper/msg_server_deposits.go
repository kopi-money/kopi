package keeper

import (
	"context"
	"fmt"
	"strings"

	denomtypes "github.com/kopi-money/kopi/x/denominations/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k msgServer) AddDeposit(ctx context.Context, msg *types.MsgAddDeposit) (*types.Void, error) {
	cAsset, err := k.DenomKeeper.GetCAssetByBaseName(ctx, msg.Denom)
	if err != nil {
		return nil, err
	}

	amount, err := parseAmount(msg.Amount, false)
	if err != nil {
		return nil, err
	}

	address, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	if _, err = k.Deposit(ctx, address, cAsset, amount); err != nil {
		return nil, err
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("funds_deposited",
			sdk.Attribute{Key: "address", Value: msg.Creator},
			sdk.Attribute{Key: "denom", Value: msg.Denom},
			sdk.Attribute{Key: "amount", Value: msg.Amount},
		),
	)

	return &types.Void{}, nil
}

func (k Keeper) Deposit(ctx context.Context, address sdk.AccAddress, cAsset *denomtypes.CAsset, amount math.Int) (math.Int, error) {
	if k.BankKeeper.SpendableCoin(ctx, address, cAsset.BaseDexDenom).Amount.LT(amount) {
		return math.Int{}, types.ErrNotEnoughFunds
	}

	newCAssetTokens, err := k.CalculateNewCAssetAmount(ctx, cAsset, amount)
	if err != nil {
		return math.Int{}, err
	}

	if newCAssetTokens.LTE(math.ZeroInt()) {
		return math.Int{}, types.ErrZeroCAssets
	}

	coins := sdk.NewCoins(sdk.NewCoin(cAsset.BaseDexDenom, amount))
	if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, address, types.PoolVault, coins); err != nil {
		return math.Int{}, fmt.Errorf("could not send coins to module: %w", err)
	}

	coins = sdk.NewCoins(sdk.NewCoin(cAsset.DexDenom, newCAssetTokens))
	if err := k.BankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
		return math.Int{}, err
	}

	if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, address, coins); err != nil {
		return math.Int{}, fmt.Errorf("could not send coins to module: %w", err)
	}

	return newCAssetTokens, nil
}

func parseAmount(amountStr string, canBeZero bool) (math.Int, error) {
	amountStr = strings.ReplaceAll(amountStr, ",", "")
	amount, ok := math.NewIntFromString(amountStr)
	if !ok {
		return math.Int{}, types.ErrInvalidAmountFormat
	}

	if amount.LT(math.ZeroInt()) {
		return math.Int{}, types.ErrNegativeAmount
	}

	if !canBeZero && amount.IsZero() {
		return math.Int{}, types.ErrZeroAmount
	}

	return amount, nil
}
