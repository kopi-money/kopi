package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
	"strings"
)

func (k Keeper) validateMsg(ctx context.Context, creator, denom string, amount math.Int) (sdk.AccAddress, error) {
	if !amount.GT(math.ZeroInt()) {
		return nil, types.ErrNegativeAmount
	}

	if !k.DenomKeeper.IsValidDenom(ctx, denom) {
		return nil, types.ErrDenomNotFound
	}

	address, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	if err = k.checkSpendableCoins(ctx, address, denom, amount); err != nil {
		return nil, err
	}

	return address, nil
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
		msg := fmt.Sprintf("NEF: %v, %v, wants: %v, has: %v", address.String(), denom, amount.String(), spendableCoins.String())
		k.Logger().Info(msg)
		return types.ErrNotEnoughFunds
	}

	return nil
}

func parseAmount(amountStr string) (math.Int, error) {
	amountStr = strings.ReplaceAll(amountStr, ",", "")
	amountInt, ok := math.NewIntFromString(amountStr)
	if !ok {
		return math.Int{}, fmt.Errorf("invalid amount string: '%v'", amountStr)
	}

	if amountInt.LT(math.ZeroInt()) {
		return math.Int{}, types.ErrNegativeAmount
	}

	return amountInt, nil
}
