package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	"github.com/kopi-money/kopi/x/ls/types"
	"sort"
)

// Undelegate executes the undelegation. It sends the sXKP to the module, burns them, calculates how much XKP the user
// is entitled to and starts the undelegation process.
func (k msgServer) Undelegate(ctx context.Context, msg *types.MsgUndelegate) (*types.Void, error) {
	amount, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s", msg.Amount)
	}

	acc, _ := sdk.AccAddressFromBech32(msg.Creator)
	if k.bankKeeper.SpendableCoins(ctx, acc).AmountOf(constants.BaseCurrency).LT(amount) {
		return nil, fmt.Errorf("not enough funds to stake given amount")
	}

	moduleAcc := k.accountKeeper.GetModuleAccount(ctx, types.ModuleName)
	undelegationAmount, err := k.calculateUndelegationAmount(ctx, moduleAcc.GetAddress(), amount)
	if err != nil {
		return nil, fmt.Errorf("error calculating undelegation amount: %v", err)
	}

	if undelegationAmount.IsZero() {
		return nil, fmt.Errorf("undelegation amount is zero")
	}

	coins := sdk.NewCoins(sdk.NewCoin(constants.LST, amount))
	if err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("error sending coins to module account: %v", err)
	}

	if err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("error burning coins: %v", err)
	}

	if err = k.undelegate(ctx, acc, moduleAcc.GetAddress(), undelegationAmount); err != nil {
		return nil, fmt.Errorf("error undelegating: %v", err)
	}

	return &types.Void{}, nil
}

// calculateUndelegationAmount calculates how much XKP the user is entitled to given how much sXKP the user wants to
// redeem. This is done by checking how much the given sXKP are in relation to all sXKP in circulation. That share is
// then multiplied with the module's value, ie its delegated XKP and potential spendable balance.
func (k Keeper) calculateUndelegationAmount(ctx context.Context, moduleAcc sdk.AccAddress, amountSAsset math.Int) (math.Int, error) {
	sAssetSupply := k.bankKeeper.GetSupply(ctx, constants.LST).Amount.ToLegacyDec()
	if sAssetSupply.IsZero() {
		return math.ZeroInt(), nil
	}

	sAssetShare := amountSAsset.ToLegacyDec().Quo(sAssetSupply)

	spendableCoins := k.bankKeeper.SpendableCoins(ctx, moduleAcc).AmountOf(constants.BaseCurrency)
	delegated, err := k.getDelegationSum(ctx, moduleAcc)
	if err != nil {
		return math.Int{}, fmt.Errorf("could not get module's delegation sum: %v", err)
	}

	moduleValue := spendableCoins.ToLegacyDec().Add(delegated)
	unstakeAmount := moduleValue.Mul(sAssetShare).TruncateInt()
	return unstakeAmount, nil
}

// undelegate starts the undelegation process for the given user and amount. For the different delegated validators,
// priority is first whether that validator is still in the top N or not, then biggest delegation to smallest. If
// undelegating from one validator is not enough, undelegations from multiple validators are started.
func (k Keeper) undelegate(ctx context.Context, userAcc, moduleAcc sdk.AccAddress, undelegationAmounLeftInt math.Int) error {
	delegationAmounts, err := k.getDelegationAmounts(ctx, moduleAcc, false)
	if err != nil {
		return fmt.Errorf("error getting delegation amounts: %v", err)
	}

	sort.SliceStable(delegationAmounts, func(i, j int) bool {
		if delegationAmounts[i].inTopN != delegationAmounts[j].inTopN {
			return !delegationAmounts[i].inTopN
		}

		return delegationAmounts[i].amount.GT(delegationAmounts[j].amount)
	})

	unstakeAmounLeft := undelegationAmounLeftInt.ToLegacyDec()
	for unstakeAmounLeft.IsPositive() && len(delegationAmounts) > 0 {
		delegationAmount := delegationAmounts[0]
		if len(delegationAmounts) > 1 {
			delegationAmounts = delegationAmounts[1:]
		}

		unstakeAmount := math.LegacyMinDec(unstakeAmounLeft, delegationAmount.amount).TruncateDec()
		unstakeAmounLeft = unstakeAmounLeft.Sub(unstakeAmount)

		if err = k.undelegateFromValidator(ctx, userAcc, moduleAcc, delegationAmount.validator, unstakeAmount.TruncateInt()); err != nil {
			return fmt.Errorf("error undelegation: %v", err)
		}
	}

	return nil
}
func (k Keeper) undelegateFromValidator(ctx context.Context, userAcc, moduleAcc sdk.AccAddress, validator string, unstakeAmount math.Int) error {
	validatorAddress, _ := sdk.ValAddressFromBech32(validator)

	shares, err := k.stakingKeeper.ValidateUnbondAmount(ctx, moduleAcc, validatorAddress, unstakeAmount)
	if err != nil {
		return err
	}

	if _, _, err = k.stakingKeeper.Undelegate(ctx, moduleAcc, validatorAddress, shares); err != nil {
		return err
	}

	k.storeNewUndelegation(ctx, userAcc.String(), unstakeAmount)
	return nil
}
