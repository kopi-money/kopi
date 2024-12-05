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

type ValidatorAmount struct {
	validator string
	amount    math.LegacyDec
	inTopN    bool
}

type ValidatorAmounts []ValidatorAmount

func (va ValidatorAmounts) contains(validator string) bool {
	for _, val := range va {
		if val.validator == validator {
			return true
		}
	}

	return false
}

// RestakeRewards withdraws rewards from all validators and then stakes it with the one validator in the top N that
// currently has the lowest delegations.
func (k Keeper) RestakeRewards(ctx context.Context) error {
	moduleAcc := k.accountKeeper.GetModuleAccount(ctx, types.ModuleName)
	rewards, err := k.withdrawRewards(ctx, moduleAcc.GetAddress())
	if err != nil {
		return fmt.Errorf("could not withdraw rewards: %w", err)
	}

	if rewards.IsZero() {
		return nil
	}

	delegationAmounts, err := k.getDelegationAmounts(ctx, moduleAcc.GetAddress(), false)
	if err != nil {
		return fmt.Errorf("error getting delegation amounts: %v", err)
	}

	// We sort to find the top-n-validator with the lowest staked amount
	sort.SliceStable(delegationAmounts, func(i, j int) bool {
		if delegationAmounts[i].inTopN != delegationAmounts[j].inTopN {
			return delegationAmounts[i].inTopN
		}

		return delegationAmounts[i].amount.LT(delegationAmounts[j].amount)
	})

	validatorAddress, _ := sdk.ValAddressFromBech32(delegationAmounts[0].validator)
	validator, err := k.stakingKeeper.GetValidator(ctx, validatorAddress)
	if err != nil {
		return fmt.Errorf("error getting validator: %w", err)
	}

	if _, err = k.stakingKeeper.Delegate(ctx, moduleAcc.GetAddress(), rewards.AmountOf(constants.BaseCurrency), stakingtypes.Unbonded, validator, true); err != nil {
		return fmt.Errorf("could not delegate: %w", err)
	}

	return nil
}

// withdrawRewards gets the account's delegations and collects all the rewards.
func (k Keeper) withdrawRewards(ctx context.Context, accAddr sdk.AccAddress) (sdk.Coins, error) {
	delegations, err := k.getDelegations(ctx, accAddr)
	if err != nil {
		return nil, fmt.Errorf("could not get delegations: %w", err)
	}

	rewards := sdk.NewCoins()
	for _, delegation := range delegations {
		var reward sdk.Coins
		reward, err = k.withdrawReward(ctx, accAddr, delegation)
		if err != nil {
			return nil, fmt.Errorf("could not withdraw reward: %w", err)
		}

		rewards = rewards.Add(reward...)
	}

	return rewards, nil
}

func (k Keeper) withdrawReward(ctx context.Context, accAddr sdk.AccAddress, validator string) (sdk.Coins, error) {
	validatorAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(validator)
	if err != nil {
		return sdk.Coins{}, err
	}

	amount, err := k.distributionKeeper.WithdrawDelegationRewards(ctx, accAddr, validatorAddr)
	if err != nil {
		return sdk.Coins{}, err
	}

	return amount, nil
}

// getDelegations returns a list of validators for which the given account has delegations.
func (k Keeper) getDelegations(ctx context.Context, accAddr sdk.AccAddress) ([]string, error) {
	var validators []string

	if err := k.stakingKeeper.IterateDelegations(
		ctx, accAddr,
		func(_ int64, del stakingtypes.DelegationI) (stop bool) {
			validators = append(validators, del.GetValidatorAddr())
			return false
		},
	); err != nil {
		return nil, err
	}

	return validators, nil
}

// getDelegationAmounts returns a list of delegations including validator address, whether it is in the top N and the
// delegated amount.
func (k Keeper) getDelegationAmounts(ctx context.Context, accAddr sdk.AccAddress, addEmpty bool) ([]ValidatorAmount, error) {
	var amounts ValidatorAmounts
	topValidators, err := k.getTopValidators(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get top validators: %w", err)
	}

	if err = k.stakingKeeper.IterateDelegations(
		ctx, accAddr,
		func(_ int64, del stakingtypes.DelegationI) (stop bool) {
			validatorAddress, _ := sdk.ValAddressFromBech32(del.GetValidatorAddr())
			validator, _ := k.stakingKeeper.Validator(ctx, validatorAddress)
			amounts = append(amounts, ValidatorAmount{
				validator: del.GetValidatorAddr(),
				amount:    validator.TokensFromShares(del.GetShares()),
				inTopN:    topValidators.contains(validator.GetOperator()),
			})

			return false
		},
	); err != nil {
		return nil, err
	}

	if addEmpty {
		var empty []ValidatorAmount
		for _, topValidator := range topValidators {
			if !amounts.contains(topValidator.GetOperator()) {
				empty = append(empty, ValidatorAmount{
					validator: topValidator.GetOperator(),
					amount:    math.LegacyZeroDec(),
					inTopN:    true,
				})
			}
		}

		amounts = append(amounts, empty...)
	}

	return amounts, nil
}

// getDelegationSum returns the sum of an account's delegations
func (k Keeper) getDelegationSum(ctx context.Context, accAddr sdk.AccAddress) (math.LegacyDec, error) {
	sum := math.LegacyZeroDec()

	var (
		validatorAddress sdk.ValAddress
		validator        stakingtypes.ValidatorI
		innerErr         error
	)
	if err := k.stakingKeeper.IterateDelegations(
		ctx, accAddr,
		func(_ int64, del stakingtypes.DelegationI) (stop bool) {
			validatorAddress, innerErr = sdk.ValAddressFromBech32(del.GetValidatorAddr())
			if innerErr != nil {
				return true
			}
			validator, innerErr = k.stakingKeeper.Validator(ctx, validatorAddress)
			if innerErr != nil {
				return true
			}

			sum = sum.Add(validator.TokensFromShares(del.GetShares()))
			return false
		},
	); err != nil {
		return math.LegacyDec{}, err
	}

	if innerErr != nil {
		return math.LegacyDec{}, innerErr
	}

	return sum, nil
}

type TopValidators []stakingtypes.Validator

func (tv TopValidators) contains(operatorAddress string) bool {
	for _, validator := range tv {
		if validator.GetOperator() == operatorAddress {
			return true
		}
	}

	return false
}

func (k Keeper) getTopValidators(ctx context.Context) (TopValidators, error) {
	validators, err := k.stakingKeeper.GetBondedValidatorsByPower(ctx)
	if err != nil {
		return nil, err
	}

	topNValidators := k.GetParams(ctx).TopNValidators
	if len(validators) > int(topNValidators) {
		validators = validators[:topNValidators]
	}

	return validators, nil
}
