package types

import (
	"context"
	"cosmossdk.io/math"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI // only used for simulation
	// Methods imported from account should be defined here

	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SpendableCoin(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	// Methods imported from bank should be defined here

	SendCoins(ctx context.Context, fromAddress, toAddress sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error

	BurnCoins(ctx context.Context, name string, amt sdk.Coins) error
	MintCoins(ctx context.Context, name string, amt sdk.Coins) error

	GetSupply(ctx context.Context, denom string) sdk.Coin

	IterateAllBalances(ctx context.Context, cb func(sdk.AccAddress, sdk.Coin) bool)
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}

type DenomKeeper interface {
	CreateRatioFromReference(ctx context.Context, referenceFactor math.LegacyDec, referenceDenom string, exponent uint64) (math.LegacyDec, error)
	DexAddDenom(ctx context.Context, denom denomtypes.DexDenom, ratio denomtypes.Ratio) error
	IsKCoin(ctx context.Context, denom string) bool
	IsValidDenom(ctx context.Context, denom string) bool
	GetHighestUSDReference(context.Context) (string, error)
	GetValueIn(ctx context.Context, denomFrom, denomTo string, amount math.LegacyDec) (math.LegacyDec, error)
	GetValueInUSD(ctx context.Context, denomFrom string, amount math.LegacyDec) (math.LegacyDec, error)
}

type DexKeeper interface {
	AddLiquidityWithCompound(context.Context, sdk.AccAddress, string, math.Int, bool) error
}
