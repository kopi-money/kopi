package keeper

import (
	"context"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k msgServer) AddDeposit(goCtx context.Context, msg *types.MsgAddDeposit) (*types.Void, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

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

	if err = k.checkSpendableCoins(ctx, address, msg.Denom, amount); err != nil {
		return nil, err
	}

	coins := sdk.NewCoins(sdk.NewCoin(msg.Denom, amount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, address, types.PoolVault, coins); err != nil {
		return nil, errors.Wrap(err, "could not send coins to module")
	}

	newCAssetTokens := k.CalculateNewCAssetAmount(ctx, amount, cAsset)
	if newCAssetTokens.LTE(math.ZeroInt()) {
		k.logger.Error("zero c assets printed")
		return nil, types.ErrZeroCAssets
	}

	coins = sdk.NewCoins(sdk.NewCoin(cAsset.Name, newCAssetTokens))
	if err = k.BankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
		return nil, err
	}

	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, address, coins); err != nil {
		return nil, errors.Wrap(err, "could not send coins to module")
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("funds_deposited",
			sdk.Attribute{Key: "address", Value: msg.Creator},
			sdk.Attribute{Key: "denom", Value: msg.Denom},
			sdk.Attribute{Key: "amount", Value: msg.Amount},
		),
	)

	return &types.Void{}, nil
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

	if !canBeZero && amount.Equal(math.ZeroInt()) {
		return math.Int{}, types.ErrZeroAmount
	}

	return amount, nil
}

func (k Keeper) checkSpendableCoins(ctx context.Context, address sdk.AccAddress, denom string, amount math.Int) error {
	var spendableCoins math.Int
	for _, coin := range k.BankKeeper.SpendableCoins(ctx, address) {
		if coin.Denom == denom {
			spendableCoins = coin.Amount
			break
		}
	}

	if spendableCoins.IsNil() || amount.GT(spendableCoins) {
		return types.ErrNotEnoughFunds
	}

	return nil
}
