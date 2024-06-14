package types

import (
	"context"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/cache"
	dextypes "github.com/kopi-money/kopi/x/dex/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI // only used for simulation
	// Methods imported from account should be defined here

	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
}

type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	// Methods imported from bank should be defined here

	SendCoins(ctx context.Context, sendingAddress, recipientAddress sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, sendingModule, recipientModule string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, name string, amt sdk.Coins) error
	MintCoins(ctx context.Context, name string, amt sdk.Coins) error

	GetSupply(ctx context.Context, denom string) sdk.Coin
}

type DexKeeper interface {
	cache.Cache

	AddLiquidity(ctx context.Context, eventManager sdk.EventManagerI, address sdk.AccAddress, denom string, amount math.Int) error
	CalculateParity(ctx context.Context, kCoin string) (*math.LegacyDec, string, error)
	CalculatePrice(ctx context.Context, denomFrom, denomTo string) (math.LegacyDec, error)
	ExecuteTrade(ctx dextypes.TradeContext) (math.Int, math.Int, math.Int, math.Int, math.Int, error)
	GetLiquidityByAddress(ctx context.Context, denom, address string) math.Int
	GetFullLiquidityBase(ctx context.Context, denomOther string) math.LegacyDec
	GetFullLiquidityOther(ctx context.Context, denom string) math.LegacyDec
	GetLiquiditySum(ctx context.Context, denom string) math.Int
	GetRatio(ctx context.Context, denom string) (dextypes.Ratio, error)
	RemoveAllLiquidityForModule(ctx context.Context, eventManager sdk.EventManagerI, denom, module string) error
	RemoveLiquidityForAddress(ctx context.Context, eventManager sdk.EventManagerI, accAddress sdk.AccAddress, denom string, amount math.Int) error
	SimulateTradeForReserve(ctx dextypes.TradeContext) (math.Int, math.LegacyDec, math.LegacyDec, error)
}

type DenomKeeper interface {
	IsKCoin(ctx context.Context, denom string) bool
	MaxSupply(ctx context.Context, kCoin string) math.Int
	MaxBurnAmount(ctx context.Context, kCoin string) math.Int
	MaxMintAmount(ctx context.Context, kCoin string) math.Int
	KCoins(ctx context.Context) []string
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}
