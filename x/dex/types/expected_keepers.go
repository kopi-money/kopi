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
	Denoms(ctx context.Context) []string
	GetCAssetByBaseName(ctx context.Context, baseDenom string) (*denomtypes.CAsset, error)
	InitialVirtualLiquidityFactor(ctx context.Context, denom string) (math.LegacyDec, error)
	IsNativeDenom(ctx context.Context, denom string) bool
	IsValidDenom(ctx context.Context, denom string) bool
	IsKCoin(ctx context.Context, denom string) bool
	MinLiquidity(ctx context.Context, denom string) math.Int
	MinOrderSize(ctx context.Context, denom string) math.Int
	ReferenceDenoms(ctx context.Context, kCoin string) []string
	KCoins(ctx context.Context) (kCoins []string)
}
