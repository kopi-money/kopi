package types

import (
	"context"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/cache"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	dextypes "github.com/kopi-money/kopi/x/dex/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI // only used for simulation
	// Methods imported from account should be defined here
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SpendableCoin(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	// Methods imported from bank should be defined here

	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	SendCoins(ctx context.Context, senderAddr, recipientAddr sdk.AccAddress, amt sdk.Coins) error

	BurnCoins(ctx context.Context, name string, amt sdk.Coins) error
	MintCoins(ctx context.Context, name string, amt sdk.Coins) error
	GetSupply(ctx context.Context, denom string) sdk.Coin
}

type BlockspeedKeeper interface {
	BlocksPerYear(ctx context.Context) (math.LegacyDec, error)
}

type DenomKeeper interface {
	Denoms(ctx context.Context) []string
	GetCAssets(context.Context) []*denomtypes.CAsset
	GetCAssetByBaseName(context.Context, string) (*denomtypes.CAsset, error)
	GetCAssetByName(context.Context, string) (*denomtypes.CAsset, error)
	GetCollateralDenom(context.Context, string) *denomtypes.CollateralDenom
	GetCollateralDenoms(context.Context) []*denomtypes.CollateralDenom
	GetDepositCap(context.Context, string) (math.Int, error)
	GetLTV(ctx context.Context, denom string) (math.LegacyDec, error)
	IsValidCollateralDenom(context.Context, string) bool
}
type DexKeeper interface {
	cache.Cache

	CalculatePrice(ctx context.Context, denomFrom, denomTo string) (math.LegacyDec, error)
	ExecuteBuy(cctx dextypes.TradeContext) (dextypes.TradeResult, error)
	GetDenomValue(ctx context.Context, denom string) (math.LegacyDec, error)
	GetLiquidityByAddress(ctx context.Context, denom, address string) math.Int
	GetAllOrdersByAddress(ctx context.Context, address string) []dextypes.Order
	GetValueInUSD(ctx context.Context, denom string, amount math.LegacyDec) (math.LegacyDec, error)
	GetHighestUSDReference(ctx context.Context) (string, error)
	GetValueInBase(ctx context.Context, denom string, amount math.LegacyDec) (math.LegacyDec, error)
	GetValueIn(ctx context.Context, denomFrom, denomTo string, amount math.LegacyDec) (math.LegacyDec, error)
	NewOrdersCaches(ctx context.Context) *dextypes.OrdersCaches
	SimulateSell(ctx dextypes.TradeContext) (dextypes.TradeSimulationResult, error)
}
