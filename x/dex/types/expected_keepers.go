package types

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	// Methods imported from account should be defined here

	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SpendableCoin(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	// Methods imported from bank should be defined here

	SendCoins(ctx context.Context, fromAddress, toAddress sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error

	BurnCoins(ctx context.Context, name string, amt sdk.Coins) error
	MintCoins(ctx context.Context, name string, amt sdk.Coins) error

	GetSupply(ctx context.Context, denom string) sdk.Coin
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}

type DenomKeeper interface {
	CalculatePrice(ctx context.Context, denomGiving, denomReceiving string) (math.LegacyDec, error)
	ConvertToExponent(ctx context.Context, denom string, amount math.LegacyDec, targetExponent uint64) (math.LegacyDec, error)
	Denoms(ctx context.Context) []string
	ExtraVirtualLiquidity(ctx context.Context, denom string) math.Int
	GetAllRatios(ctx context.Context) []denomtypes.Ratio
	GetAuthority() string
	GetCAssetByBaseName(ctx context.Context, baseDenom string) (denomtypes.CAsset, error)
	GetHighestUSDReference(ctx context.Context) (string, error)
	GetRatio(ctx context.Context, denom string) (denomtypes.Ratio, error)
	GetPriceInUSD(ctx context.Context, denom string) (math.LegacyDec, error)
	GetValueIn(ctx context.Context, denomFrom, denomTo string, amount math.LegacyDec) (math.LegacyDec, error)
	GetValueInFromUSD(ctx context.Context, denom string, amount math.LegacyDec) (math.LegacyDec, error)
	GetValueInBase(ctx context.Context, denom string, amount math.LegacyDec) (math.LegacyDec, error)
	GetValueInUSD(ctx context.Context, denom string, amount math.LegacyDec) (math.LegacyDec, error)
	IsCollateralDenom(ctx context.Context, denom string) bool
	IsKCoin(ctx context.Context, denom string) bool
	IsNativeDenom(ctx context.Context, denom string) bool
	IsValidDenom(ctx context.Context, denom string) bool
	KCoins(ctx context.Context) (kCoins []string)
	MaxBurnAmount(ctx context.Context, denom string) math.Int
	MaxMintAmount(ctx context.Context, denom string) math.Int
	MinLiquidity(ctx context.Context, denom string) math.Int
	MinOrderSize(ctx context.Context, denom string) math.Int
	ReferenceDenoms(ctx context.Context, kCoin string) []string
	RemoveDenom(ctx context.Context, denom string) error
	SetRatio(ctx context.Context, ratio denomtypes.Ratio)
}

type BlockspeedKeeper interface {
	GetSecondsPerBlock(context.Context) math.LegacyDec
}
