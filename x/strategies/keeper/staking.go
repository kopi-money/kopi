package keeper

import (
	"context"
	"fmt"

	"github.com/kopi-money/kopi/x/strategies/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/kopi-money/constants"
)

func (k Keeper) withdrawRewardsAndStake(ctx context.Context, accAddr sdk.AccAddress, strategy string, pseudoRandomNumber int) (string, error) {
	rewards, err := k.withdrawRewards(ctx, accAddr)
	if err != nil {
		return "", fmt.Errorf("withdraw rewards: %w", err)
	}

	var validator string
	validator, err = k.stake(ctx, accAddr, rewards.AmountOf(constants.BaseCurrency), strategy, pseudoRandomNumber)
	if err != nil {
		return "", fmt.Errorf("restake rewards: %w", err)
	}

	return validator, nil
}

func (k Keeper) withdrawRewards(ctx context.Context, accAddr sdk.AccAddress) (sdk.Coins, error) {
	delegations, err := k.getDelegations(ctx, accAddr)
	if err != nil {
		return nil, fmt.Errorf("get delegations: %w", err)
	}

	rewards := sdk.NewCoins()
	for _, delegation := range delegations {
		var reward sdk.Coins
		reward, err = k.withdrawReward(ctx, accAddr, delegation)
		if err != nil {
			return nil, fmt.Errorf("withdraw reward: %w", err)
		}

		rewards = rewards.Add(reward...)
	}

	return rewards, nil
}

func (k Keeper) withdrawReward(ctx context.Context, accAddr sdk.AccAddress, validator string) (sdk.Coins, error) {
	k.Logger().Info(fmt.Sprintf("WR %v %v", accAddr.String(), validator))

	validatorAddr, err := k.StakingKeeper.ValidatorAddressCodec().StringToBytes(validator)
	if err != nil {
		return sdk.Coins{}, err
	}

	amount, err := k.DistributionKeeper.WithdrawDelegationRewards(ctx, accAddr, validatorAddr)
	if err != nil {
		return sdk.Coins{}, err
	}

	return amount, nil
}

func (k Keeper) getDelegations(ctx context.Context, accAddr sdk.AccAddress) ([]string, error) {
	var validators []string

	if err := k.StakingKeeper.IterateDelegations(
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

func (k Keeper) stake(ctx context.Context, accAddr sdk.AccAddress, amount math.Int, strategy string, pseudoRandomNumber int) (string, error) {
	validators, err := k.StakingKeeper.GetBondedValidatorsByPower(ctx)
	if err != nil {
		return "", fmt.Errorf("get validators: %w", err)
	}

	var validator stakingtypes.Validator
	if isRandomStakingStrategy(strategy) {
		validator = getPseudoRandomValidator(ctx, validators, strategy, pseudoRandomNumber)
	} else {
		validator, err = getValidatorFromList(validators, strategy)
		if err != nil {
			return "", err
		}
	}

	if _, err = k.StakingKeeper.Delegate(ctx, accAddr, amount, stakingtypes.Unbonded, validator, true); err != nil {
		return "", fmt.Errorf("delegate: %w", err)
	}

	return validator.OperatorAddress, nil
}

func (k Keeper) getAmountStaked(ctx context.Context, address sdk.AccAddress) (math.Int, error) {
	// 1k should be enough...
	delegations, err := k.StakingKeeper.GetDelegatorDelegations(ctx, address, 1_000)
	if err != nil {
		return math.Int{}, err
	}

	sum := math.LegacyZeroDec()
	for _, delegation := range delegations {
		sum = sum.Add(delegation.Shares)
	}

	return sum.TruncateInt(), nil
}

// Taken from x/staking/DelegationTotalRewards
func (k Keeper) getStakingRewards(ctx context.Context, address sdk.AccAddress) (math.LegacyDec, error) {
	total := sdk.DecCoins{}

	if err := k.StakingKeeper.IterateDelegations(
		ctx, address,
		func(_ int64, del stakingtypes.DelegationI) (stop bool) {
			valAddr, err := sdk.ValAddressFromBech32(del.GetValidatorAddr())
			if err != nil {
				panic(err)
			}

			val, err := k.StakingKeeper.Validator(ctx, valAddr)
			if err != nil {
				panic(err)
			}

			endingPeriod, err := k.DistributionKeeper.IncrementValidatorPeriod(ctx, val)
			if err != nil {
				panic(err)
			}

			delReward, err := k.DistributionKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)
			if err != nil {
				panic(err)
			}

			total = total.Add(delReward...)
			return false
		},
	); err != nil {
		return math.LegacyDec{}, err
	}

	return total.AmountOf(constants.BaseCurrency), nil
}

func getValidatorFromList(validators []stakingtypes.Validator, address string) (stakingtypes.Validator, error) {
	for _, validator := range validators {
		if validator.OperatorAddress == address {
			return validator, nil
		}
	}

	return stakingtypes.Validator{}, types.ErrNonExistingValidator
}

func getPseudoRandomValidator(ctx context.Context, validators []stakingtypes.Validator, strategy string, pseudoRandomNumber int) stakingtypes.Validator {
	switch strategy {
	case "top5":
		if len(validators) > 5 {
			validators = validators[:5]
		}
	case "top10":
		if len(validators) > 10 {
			validators = validators[:10]
		}
	case "bottom5":
		if len(validators) > 5 {
			validators = validators[len(validators)-5:]
		}
	case "bottom10":
		if len(validators) > 10 {
			validators = validators[len(validators)-10:]
		}
	}

	index := int(sdk.UnwrapSDKContext(ctx).BlockHeight()) + pseudoRandomNumber
	index %= len(validators)
	return validators[index]
}

func isValidStakingStrategy(strategy string) bool {
	switch strategy {
	case "top5", "top10", "bottom5", "bottom10", "random":
		return true
	default:
		_, err := sdk.ValAddressFromBech32(strategy)
		return err == nil
	}
}

func isRandomStakingStrategy(strategy string) bool {
	switch strategy {
	case "top5", "top10", "bottom5", "bottom10", "random":
		return true
	default:
		return false
	}
}

func checkForStakingStrategy(string1, string2 string) error {
	if string1 == "" {
		return fmt.Errorf("string1 is empty")
	}

	if !isValidStakingStrategy(string1) {
		return fmt.Errorf("invalid staking strategy: %v", string1)
	}

	if string2 != "" {
		return fmt.Errorf("string2 has to be empty")
	}

	return nil
}

func getStakingStrategy(string1, string2 string) string {
	if string1 != "" {
		return string1
	} else {
		return string2
	}
}
