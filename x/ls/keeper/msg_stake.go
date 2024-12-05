package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/kopi-money/kopi/constants"
	"github.com/kopi-money/kopi/x/ls/types"
	"sort"
)

// Delegate executes the message to delegate new tokens. The function checks whether the amount is valid, mints new
// sXKP, sends them to the user and then executes the delegation.
func (k msgServer) Delegate(ctx context.Context, msg *types.MsgDelegate) (*types.Void, error) {
	amount, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s", msg.Amount)
	}

	acc, _ := sdk.AccAddressFromBech32(msg.Creator)
	if k.bankKeeper.SpendableCoins(ctx, acc).AmountOf(constants.BaseCurrency).LT(amount) {
		return nil, fmt.Errorf("not enough funds to stake given amount")
	}

	moduleAcc := k.accountKeeper.GetModuleAccount(ctx, types.ModuleName)

	coins := sdk.NewCoins(sdk.NewCoin(constants.BaseCurrency, amount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("failed to send coins from account to module")
	}

	newTokens, err := k.calculateSAssetAmount(ctx, moduleAcc.GetAddress(), amount)
	if err != nil {
		return nil, fmt.Errorf("unable to calculate new tokens: %v", err)
	}

	coins = sdk.NewCoins(sdk.NewCoin(constants.LST, newTokens))
	if err = k.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("unable to mint coins: %v", err)
	}

	if err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, acc, coins); err != nil {
		return nil, fmt.Errorf("unable to send coins from module to user: %v", err)
	}

	if err = k.delegate(ctx, moduleAcc.GetAddress()); err != nil {
		return nil, fmt.Errorf("unable to stake: %v", err)
	}

	return &types.Void{}, nil
}

// calculateSAssetAmount calculates the amount of sXKP to be minted given the added amount of XKP. To do that, the added
// amount of XKP is set in relation to the total amount of staked XKP. This function assumes that the new funds have
// already been sent to the module.
func (k Keeper) calculateSAssetAmount(ctx context.Context, moduleAcc sdk.AccAddress, addedAmount math.Int) (math.Int, error) {
	sAssetSupply := k.bankKeeper.GetSupply(ctx, constants.LST).Amount.ToLegacyDec()
	if sAssetSupply.IsZero() {
		return addedAmount, nil
	}

	spendableCoins := k.bankKeeper.SpendableCoins(ctx, moduleAcc).AmountOf(constants.BaseCurrency)
	delegated, err := k.getDelegationSum(ctx, moduleAcc)
	if err != nil {
		return math.Int{}, fmt.Errorf("could not get module's delegation sum: %v", err)
	}

	moduleValue := spendableCoins.ToLegacyDec().Add(delegated)
	valueShare := addedAmount.ToLegacyDec().Quo(moduleValue)

	var newTokens math.Int
	if valueShare.Equal(math.LegacyOneDec()) {
		newTokens = addedAmount
	} else {
		newTokens = sAssetSupply.Quo(math.LegacyOneDec().Sub(valueShare)).Sub(sAssetSupply).TruncateInt()
	}

	return newTokens, nil
}

// stake adds all spendable coins to the validator from the top N that has the least amount staked to it. If two
// validator have the same amount staked to, the one higher in the list is preferred.
func (k Keeper) delegate(ctx context.Context, moduleAcc sdk.AccAddress) error {
	delegationAmounts, err := k.getDelegationAmounts(ctx, moduleAcc)
	if err != nil {
		return fmt.Errorf("error getting delegation amounts: %v", err)
	}

	sort.SliceStable(delegationAmounts, func(i, j int) bool {
		if delegationAmounts[i].inTopN != delegationAmounts[j].inTopN {
			return delegationAmounts[i].inTopN
		}

		if delegationAmounts[i].amount.Equal(delegationAmounts[j].amount) {
			return i < j
		}

		return delegationAmounts[i].amount.LT(delegationAmounts[j].amount)
	})

	spendableCoins := k.bankKeeper.SpendableCoins(ctx, moduleAcc).AmountOf(constants.BaseCurrency)
	validatorAddress, _ := sdk.ValAddressFromBech32(delegationAmounts[0].validator)
	validator, err := k.stakingKeeper.GetValidator(ctx, validatorAddress)
	if err != nil {
		return fmt.Errorf("error getting validator: %v", err)
	}

	if _, err = k.stakingKeeper.Delegate(ctx, moduleAcc, spendableCoins, stakingtypes.Unbonded, validator, true); err != nil {
		return fmt.Errorf("could not delegate: %w", err)
	}

	return nil
}
