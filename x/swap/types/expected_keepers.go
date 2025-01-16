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

type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SpendableCoin(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	// Methods imported from bank should be defined here

	SendCoins(ctx context.Context, sendingAddress, recipientAddress sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, sendingModule, recipientModule string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, name string, amt sdk.Coins) error
	MintCoins(ctx context.Context, name string, amt sdk.Coins) error

	GetSupply(ctx context.Context, denom string) sdk.Coin
}

type BlockspeedKeeper interface {
	GetBlocksPerSecond(context.Context) (math.LegacyDec, error)
}

type DexKeeper interface {
	cache.Cache

	AddLiquidity(ctx context.Context, address sdk.AccAddress, denom string, amount math.Int) (math.Int, error)
	CalculateParity(ctx context.Context, kCoin string) (*math.LegacyDec, string, error)
	CalculatePrice(ctx context.Context, denomFrom, denomTo string) (math.LegacyDec, error)
	ExecuteSell(ctx dextypes.TradeContext) (dextypes.TradeResult, error)
	GetLiquidityByAddress(ctx context.Context, denom, address string) math.Int
	GetFullLiquidityBase(ctx context.Context, denomOther string) math.LegacyDec
	GetFullLiquidityOther(ctx context.Context, denom string) math.LegacyDec
	GetLiquiditySum(ctx context.Context, denom string) math.Int
	GetValueInBase(ctx context.Context, denom string, amount math.LegacyDec) (math.LegacyDec, error)
	RemoveAllLiquidityForModule(ctx context.Context, denom, module string) error
	RemoveLiquidityForAddress(ctx context.Context, accAddress sdk.AccAddress, denom string, amount math.Int) error
	SimulateTradeForReserve(ctx dextypes.TradeContext) (dextypes.TradeSimulationResult, error)
}

type DenomKeeper interface {
	GetArbitrageDenoms(ctx context.Context) []*denomtypes.ArbitrageDenom
	GetRatio(ctx context.Context, denom string) (denomtypes.Ratio, error)
	IsKCoin(ctx context.Context, denom string) bool
	KCoins(ctx context.Context) []string
	MaxSupply(ctx context.Context, kCoin string) math.Int
	MaxBurnAmount(ctx context.Context, kCoin string) math.Int
	MaxMintAmount(ctx context.Context, kCoin string) math.Int
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}
