package types

import (
	"fmt"
	"github.com/kopi-money/constants"

	"cosmossdk.io/math"
)

var (
	OfferFee                        = math.LegacyNewDecWithPrec(5, 3) // 0.005, 0.5%
	ReserveFeeShare                 = math.LegacyNewDecWithPrec(5, 1) // 50%
	MinimumPoolSize                 = math.NewInt(1000_000000)        // 1000
	MinimumPoolFee                  = math.LegacyNewDecWithPrec(1, 3) // 0.001
	MaximumPoolFee                  = math.LegacyNewDecWithPrec(5, 2) // 0.05
	MinimumPoolMovingValue          = math.NewInt(5000_000000)        // 5000
	MinimumUnlockingInSeconds int64 = 86400
	MaximumVestingUnlockSteps int64 = 100
)

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		MinimumUnlockInSeconds:    MinimumUnlockingInSeconds,
		MaximumVestingUnlockSteps: MaximumVestingUnlockSteps,
		ReserveFeeShare:           ReserveFeeShare,
		MinimumPoolSize:           MinimumPoolSize,
		MinimumPoolFee:            MinimumPoolFee,
		MaximumPoolFee:            MaximumPoolFee,
		OfferFee:                  OfferFee,
		ChangeSecondsDescription:  constants.SecondsPerDay * 7,
		ChangeSecondsWebsite:      constants.SecondsPerDay * 7,
		ChangeSecondsImage:        constants.SecondsPerDay * 7,
		PoolTresholdSeconds:       constants.SecondsPerDay * 7,
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateShare(p.ReserveFeeShare); err != nil {
		return fmt.Errorf("invalid reserve fee share: %w", err)
	}

	if err := validateShare(p.MinimumPoolFee); err != nil {
		return fmt.Errorf("invalid minimum pool fee: %w", err)
	}

	if err := validateCategories(p.Categories.Categories); err != nil {
		return fmt.Errorf("invalid categories: %w", err)
	}

	if err := validateMaximumVestingSteps(p.MaximumVestingUnlockSteps); err != nil {
		return fmt.Errorf("invalid maximum vesting unlock steps: %w", err)
	}

	if err := validatePoolFees(p.MinimumPoolFee, p.MaximumPoolFee); err != nil {
		return fmt.Errorf("invalid pool fees: %w", err)
	}

	if err := validateOfferFee(p.OfferFee); err != nil {
		return fmt.Errorf("invalid offer fee: %w", err)
	}

	return nil
}

func validateOfferFee(offerFee math.LegacyDec) error {
	if offerFee.IsNegative() {
		return fmt.Errorf("offer fee cannot be negative")
	}

	if offerFee.GTE(math.LegacyOneDec()) {
		return fmt.Errorf("offer fee must not be greater than or equal to one")
	}

	return nil
}

func validateShare(d any) error {
	v, ok := d.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", d)
	}

	if v.IsNil() {
		return fmt.Errorf("value is nil")
	}

	if v.IsNegative() {
		return fmt.Errorf("fee must not be smaller than 0")
	}

	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("fee must not be greater than 1")
	}

	return nil
}

func validateCategories(categories []Category) error {
	seen := make(map[uint64]bool)
	names := make(map[string]bool)
	seenLocal := false

	for _, category := range categories {
		if _, has := seen[category.Index]; has {
			return fmt.Errorf("duplicate category index: %v", category.Index)
		}

		if _, has := names[category.Name]; has {
			return fmt.Errorf("duplicate category name: %v", category.Index)
		}

		if category.CreationPrice.IsNegative() {
			return fmt.Errorf("negative creation price: %v", category.CreationPrice)
		}

		seen[category.Index] = true
		names[category.Name] = true

		if category.IsIbc {
			if seenLocal {
				return fmt.Errorf("only one category can be ibc")
			}

			seenLocal = true
		}
	}

	return nil
}

func validateMaximumVestingSteps(maximumVestingSteps int64) error {
	if maximumVestingSteps < 1 {
		return fmt.Errorf("invalid maximum vesting steps: %d", maximumVestingSteps)
	}

	return nil
}

func validatePoolFees(minimumPoolFee, maximumPoolFee math.LegacyDec) error {
	if minimumPoolFee.GT(maximumPoolFee) {
		return fmt.Errorf("minimum pool fee is greater than maximum pool fee")
	}

	if err := validatePoolFee(minimumPoolFee); err != nil {
		return fmt.Errorf("invalid minimum pool fee: %w", err)
	}

	if err := validatePoolFee(maximumPoolFee); err != nil {
		return fmt.Errorf("invalid maximum pool fee: %w", err)
	}

	return nil
}

func validatePoolFee(poolFee math.LegacyDec) error {
	if poolFee.IsNegative() {
		return fmt.Errorf("must not be negative")
	}

	if poolFee.GT(math.LegacyOneDec()) {
		return fmt.Errorf("must not be larger than 1")
	}

	return nil
}
