package types

import (
	"context"
	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
	SpendableCoin(context.Context, sdk.AccAddress, string) sdk.Coin
	// Methods imported from bank should be defined here
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error

	BurnCoins(ctx context.Context, name string, amt sdk.Coins) error
	MintCoins(ctx context.Context, name string, amt sdk.Coins) error
	GetSupply(ctx context.Context, denom string) sdk.Coin
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}

type DistributionKeeper interface {
	CalculateDelegationRewards(ctx context.Context, val stakingtypes.ValidatorI, del stakingtypes.DelegationI, endingPeriod uint64) (rewards sdk.DecCoins, err error)
	IncrementValidatorPeriod(ctx context.Context, val stakingtypes.ValidatorI) (uint64, error)
	WithdrawDelegationRewards(context.Context, sdk.AccAddress, sdk.ValAddress) (sdk.Coins, error)
}

type StakingKeeper interface {
	IterateDelegations(ctx context.Context, delegator sdk.AccAddress,
		fn func(index int64, delegation stakingtypes.DelegationI) (stop bool)) error

	Delegate(context.Context, sdk.AccAddress, math.Int, stakingtypes.BondStatus, stakingtypes.Validator, bool) (math.LegacyDec, error)
	GetBondedValidatorsByPower(context.Context) ([]stakingtypes.Validator, error)
	GetDelegatorDelegations(ctx context.Context, delegator sdk.AccAddress, maxRetrieve uint16) (delegations []stakingtypes.Delegation, err error)

	GetValidator(ctx context.Context, addr sdk.ValAddress) (validator stakingtypes.Validator, err error)
	Validator(context.Context, sdk.ValAddress) (stakingtypes.ValidatorI, error)
	ValidatorAddressCodec() address.Codec
	ValidateUnbondAmount(context.Context, sdk.AccAddress, sdk.ValAddress, math.Int) (math.LegacyDec, error)
	Undelegate(context.Context, sdk.AccAddress, sdk.ValAddress, math.LegacyDec) (time.Time, math.Int, error)
}
