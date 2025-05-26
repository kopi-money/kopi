package keeper

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k Keeper) createVesting(ctx context.Context, senderAddress, receiverAddress, factoryDenom string, amount math.Int, startTime, vestedUntil time.Time, numUnlockSteps int64) error {
	if vestedUntil.Before(startTime) {
		return types.ErrVestingInvalidEnd
	}

	if numUnlockSteps < 1 {
		return types.ErrVestingNegativeSteps
	}

	millis := vestedUntil.Sub(startTime).Milliseconds()
	stepSize := int64(float64(millis) / float64(numUnlockSteps))

	k.Logger().Error(fmt.Sprintf("millis: %v", millis))
	k.Logger().Error(fmt.Sprintf("stepsize: %v", stepSize))

	unlockAmountPerStep := amount.ToLegacyDec().Quo(math.LegacyNewDec(numUnlockSteps))

	var (
		previousUnlocking = startTime
		unlockings        []time.Time
	)

	for range numUnlockSteps {
		previousUnlocking = previousUnlocking.Add(time.Duration(stepSize) * time.Millisecond)
		k.Logger().Error(fmt.Sprintf("next: %v", previousUnlocking))
		unlockings = append(unlockings, previousUnlocking)
	}

	nextIndex, _ := k.vestingsNextIndex.Get(ctx)
	nextIndex += 1
	k.vestingsNextIndex.Set(ctx, nextIndex)

	vesting := types.Vesting{
		Index:                 nextIndex,
		Address:               receiverAddress,
		FactoryDenom:          factoryDenom,
		AmountVested:          amount,
		AmountLeft:            amount,
		AmountUnlockedPerStep: unlockAmountPerStep,
		NextUnlocks:           unlockings,
	}

	k.vestings.Set(ctx, nextIndex, vesting)

	acc, _ := sdk.AccAddressFromBech32(senderAddress)
	coins := sdk.NewCoins(sdk.NewCoin(factoryDenom, amount))
	if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolVestings, coins); err != nil {
		return err
	}

	return nil
}

func (k Keeper) cancelVesting(ctx context.Context, adminAddress, factoryDenom string, index uint64) error {
	vesting, has := k.vestings.Get(ctx, index)
	if !has {
		return types.ErrVestingNotFound
	}

	if vesting.FactoryDenom != factoryDenom {
		return types.ErrVestingDifferentDenom
	}

	k.vestings.Remove(ctx, index)

	acc, _ := sdk.AccAddressFromBech32(adminAddress)
	coins := sdk.NewCoins(sdk.NewCoin(factoryDenom, vesting.AmountLeft))
	if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolVestings, acc, coins); err != nil {
		return err
	}

	return nil
}

func (k Keeper) HandleVestings(ctx context.Context, blockTime time.Time) error {
	iterator := k.vestings.Iterator(ctx, nil)
	for iterator.Valid() {
		keyValue := iterator.GetNextKeyValue()

		vesting := keyValue.Value().Value()
		if blockTime.After(vesting.NextUnlocks[0]) {
			if err := k.handleVesting(ctx, *vesting); err != nil {
				return err
			}
		}
	}

	return nil
}

func (k Keeper) handleVesting(ctx context.Context, vesting types.Vesting) error {
	var payoutAmount math.Int
	if len(vesting.NextUnlocks) == 1 {
		payoutAmount = vesting.AmountLeft
	} else {
		payoutAmount = vesting.AmountUnlockedPerStep.TruncateInt()
	}

	acc, _ := sdk.AccAddressFromBech32(vesting.Address)
	coins := sdk.NewCoins(sdk.NewCoin(vesting.FactoryDenom, payoutAmount))
	if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolVestings, acc, coins); err != nil {
		return err
	}

	vesting.NextUnlocks = vesting.NextUnlocks[1:]
	if len(vesting.NextUnlocks) == 0 {
		k.vestings.Remove(ctx, vesting.Index)
	} else {
		vesting.AmountLeft = vesting.AmountLeft.Sub(payoutAmount)
		k.vestings.Set(ctx, vesting.Index, vesting)
	}

	return nil
}

func (k Keeper) getVestedAmount(ctx context.Context, denom, address string) math.Int {
	iterator := k.vestings.Iterator(ctx, nil)
	for iterator.Valid() {
		vesting := iterator.GetNext()
		if vesting.FactoryDenom == denom && vesting.Address == address {
			return vesting.AmountLeft
		}
	}

	return math.ZeroInt()
}
