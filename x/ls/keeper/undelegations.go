package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	"github.com/kopi-money/kopi/x/ls/types"
)

func (k Keeper) storeNewUndelegation(ctx context.Context, address string, amount math.Int) {
	nextIndex, _ := k.undelegationsNextIndex.Get(ctx)
	nextIndex++
	k.SetDelegationNextIndex(ctx, nextIndex)

	k.undelegations.Set(ctx, nextIndex, types.Undelegation{
		Address: address,
		Amount:  amount,
	})
}

func (k Keeper) updateUndelegation(ctx context.Context, index uint64, undelegation types.Undelegation) {
	if undelegation.Amount.IsPositive() {
		k.undelegations.Set(ctx, index, undelegation)
	} else {
		k.undelegations.Remove(ctx, index)
	}
}

func (k Keeper) SetDelegationNextIndex(ctx context.Context, nextIndex uint64) {
	k.undelegationsNextIndex.Set(ctx, nextIndex)
}

// HandleUndelegations checks whether there are spendable coins. If yes, they are used to process waiting undelegations.
func (k Keeper) HandleUndelegations(ctx context.Context) error {
	moduleAcc := k.accountKeeper.GetModuleAccount(ctx, types.ModuleName)
	spendable := k.bankKeeper.SpendableCoins(ctx, moduleAcc.GetAddress()).AmountOf(constants.BaseCurrency)
	if spendable.IsZero() {
		return nil
	}

	iterator := k.undelegations.Iterator(ctx, nil)
	for spendable.IsPositive() && iterator.Valid() {
		keyValue := iterator.GetNextKeyValue()
		undelegation := keyValue.Value().Value()

		payout := math.MinInt(spendable, undelegation.Amount)

		acc, _ := sdk.AccAddressFromBech32(undelegation.Address)
		coins := sdk.NewCoins(sdk.NewCoin(constants.BaseCurrency, payout))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, acc, coins); err != nil {
			return fmt.Errorf("error in sending coins from module account: %v", err)
		}

		spendable = spendable.Sub(payout)
		undelegation.Amount = undelegation.Amount.Sub(payout)
		k.updateUndelegation(ctx, keyValue.Key(), *undelegation)
	}

	return nil
}
