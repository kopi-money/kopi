package types

import (
	"context"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI // only used for simulation
	// Methods imported from account should be defined here

	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
	SpendableCoin(context.Context, sdk.AccAddress, string) sdk.Coin
	// Methods imported from bank should be defined here

	GetSupply(ctx context.Context, denom string) sdk.Coin
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	SendCoins(ctx context.Context, senderAddr, recipientAddr sdk.AccAddress, amt sdk.Coins) error

	BurnCoins(ctx context.Context, name string, amt sdk.Coins) error
	MintCoins(ctx context.Context, name string, amt sdk.Coins) error
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

	Validator(context.Context, sdk.ValAddress) (stakingtypes.ValidatorI, error)
	ValidatorAddressCodec() address.Codec
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}

type BlockspeedKeeper interface {
	BlocksPerYear(ctx context.Context) (math.LegacyDec, error)
	GetSecondsPerBlock(ctx context.Context) math.LegacyDec
}

type DexKeeper interface {
	AddLiquidity(context.Context, sdk.AccAddress, string, math.Int) (math.Int, error)
	CalculateParity(context.Context, string) (*math.LegacyDec, string, error)
	CalculatePrice(context.Context, string, string) (math.LegacyDec, error)
	ExecuteBuy(dextypes.TradeContext) (dextypes.TradeResult, error)
	ExecuteSell(dextypes.TradeContext) (dextypes.TradeResult, error)
	GetHighestUSDReference(ctx context.Context) (string, error)
	GetLiquidityByAddress(ctx context.Context, denom, address string) math.Int
	GetPriceInUSD(ctx context.Context, denom string) (math.LegacyDec, error)
	GetValueIn(ctx context.Context, denomFrom, denomTo string, amount math.LegacyDec) (math.LegacyDec, error)
	GetValueInUSD(ctx context.Context, denomFrom string, amount math.LegacyDec) (math.LegacyDec, error)
	NewOrdersCaches(ctx context.Context) *dextypes.OrdersCaches
	RemoveLiquidityForAddress(context.Context, sdk.AccAddress, string, math.Int) error
}

type DenomKeeper interface {
	GetArbitrageDenoms(context.Context) []denomtypes.ArbitrageDenom
	GetCAsset(context.Context, string) (denomtypes.CAsset, error)
	GetCAssetByBaseName(context.Context, string) (denomtypes.CAsset, error)
	GetArbitrageDenomByCAsset(context.Context, string) (denomtypes.ArbitrageDenom, error)
	GetArbitrageDenomByName(context.Context, string) (denomtypes.ArbitrageDenom, error)
	IsBorrowableDenom(context.Context, string) bool
	IsCAsset(context.Context, string) bool
	IsCollateralDenom(context.Context, string) bool
	IsValidDenom(context.Context, string) bool
	SetRatio(context.Context, denomtypes.Ratio)
}

type MMKeeper interface {
	AddCollateral(context.Context, sdk.AccAddress, string, math.Int) (math.Int, error)
	Borrow(context.Context, sdk.AccAddress, string, math.Int) (math.Int, math.Int, error)
	CalculateBorrowableAmount(context.Context, string, string) (math.LegacyDec, error)
	CalcWithdrawableCollateralAmount(context.Context, string, string) (math.LegacyDec, error)
	CalculateNewCAssetAmount(context.Context, denomtypes.CAsset, math.Int) (math.Int, error)
	CalculateCAssetRedemptionValue(context.Context, denomtypes.CAsset) math.LegacyDec
	CalculateCreditLineUsage(context.Context, string) (math.LegacyDec, error)
	CalculateInterestRateForDenom(context.Context, string) math.LegacyDec
	CreateRedemptionRequest(context.Context, sdk.AccAddress, denomtypes.CAsset, math.Int, math.LegacyDec) error
	ConvertToBaseAmount(context.Context, denomtypes.CAsset, math.LegacyDec) math.LegacyDec
	Deposit(context.Context, sdk.AccAddress, denomtypes.CAsset, math.Int) (math.Int, error)
	GetCollateralForDenomForAddressWithDefault(context.Context, string, string) math.Int
	GetLoanValue(ctx context.Context, denom, address string) math.LegacyDec
	GetMinimumRedemptionFee(context.Context) math.LegacyDec
	Repay(context.Context, string, string, math.Int) error
	WithdrawCollateral(context.Context, sdk.AccAddress, string, math.Int) (math.Int, error)
}
