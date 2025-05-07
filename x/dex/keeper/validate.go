package keeper

import (
	"context"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) precheckTrade(ctx context.Context, creator, denom string, amount *math.Int, allowIncomplete bool) error {
	return k.precheckTradeWithBalance(ctx, creator, denom, amount, allowIncomplete, true)
}

func (k Keeper) precheckTradeWithBalance(ctx context.Context, creator, denom string, amount *math.Int, allowIncomplete, checkBalance bool) error {
	if !amount.GT(math.ZeroInt()) {
		return types.ErrNegativeAmount
	}

	if !k.DenomKeeper.IsValidDenom(ctx, denom) {
		return denomtypes.ErrInvalidDexAsset
	}

	address, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return types.ErrInvalidAddress
	}

	if checkBalance {
		spendable := k.BankKeeper.SpendableCoin(ctx, address, denom).Amount
		if spendable.LT(*amount) {
			if allowIncomplete {
				amount = &spendable
			} else {
				return types.ErrNotEnoughFunds
			}
		}
	}

	return nil
}

func ParseAmount(amountStr string) (math.Int, error) {
	amountStr = strings.ReplaceAll(amountStr, ",", "")
	amountStr = strings.ReplaceAll(amountStr, "_", "")

	amountInt, ok := math.NewIntFromString(amountStr)
	if !ok {
		return math.Int{}, fmt.Errorf("invalid amount string: '%v'", amountStr)
	}

	if amountInt.LT(math.ZeroInt()) {
		return math.Int{}, types.ErrNegativeAmount
	}

	return amountInt, nil
}
