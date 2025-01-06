package keeper

import (
	"context"
	"testing"

	"cosmossdk.io/math"

	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/kopi-money/kopi/cache"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/runtime"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/kopi-money/kopi/x/tokenfactory/keeper"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func TokenfactoryKeeper(t *testing.T) (keeper.Keeper, context.Context) {
	dexKeeper, ctx, keys := DexKeeper(t)

	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	k := keeper.NewKeeper(
		keys.cdc,
		runtime.NewKVStoreService(keys.tof),
		log.NewNopLogger(),
		dexKeeper.AccountKeeper,
		dexKeeper.BankKeeper.(types.BankKeeper),
		dexKeeper.DenomKeeper,
		authority.String(),
	)
	cache.AddCache(k)

	// Initialize params
	require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.SetParams(innerCtx, TestParams())
	}))

	return k, ctx
}

func TestParams() types.Params {
	return types.Params{
		CreationFee:     types.CreationFee,
		ReserveFee:      types.ReserveFee,
		MinimumPoolSize: math.NewInt(100),
	}
}

func SetupTokenfactoryMsgServer(t *testing.T) (keeper.Keeper, types.MsgServer, context.Context) {
	k, ctx := TokenfactoryKeeper(t)
	addFunds(ctx, k.BankKeeper.(bankkeeper.BaseKeeper), t)
	return k, keeper.NewMsgServerImpl(k), ctx
}

func CreateFactoryDenom(ctx context.Context, msgServer types.MsgServer, creator, name string, exponent uint64) (string, error) {
	var factoryDenomHash string
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		response, err := msgServer.CreateDenom(innerCtx, &types.MsgCreateDenom{
			Creator:  creator,
			Name:     name,
			Exponent: exponent,
		})

		if err != nil {
			return err
		}

		factoryDenomHash = response.FullName
		return nil
	})

	return factoryDenomHash, err
}

func MintFactoryDenom(ctx context.Context, msgServer types.MsgServer, creator, factoryDenomHash, targetAddress, amount string) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := msgServer.MintDenom(innerCtx, &types.MsgMintDenom{
			Creator:              creator,
			FullFactoryDenomName: factoryDenomHash,
			TargetAddress:        targetAddress,
			Amount:               amount,
		})
		return err
	})
}

func BurnFactoryDenom(ctx context.Context, msgServer types.MsgServer, creator, factoryDenomHash, amount string) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := msgServer.BurnDenom(innerCtx, &types.MsgBurnDenom{
			Creator:              creator,
			FullFactoryDenomName: factoryDenomHash,
			Amount:               amount,
		})
		return err
	})
}

func CreatePool(ctx context.Context, msgServer types.MsgServer, creator, factoryDenomHash, factoryDenomAmount, kCoin, kCoinAmount, poolFee string, unlockSeconds uint64) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := msgServer.CreatePool(innerCtx, &types.MsgCreatePool{
			Creator:              creator,
			FullFactoryDenomName: factoryDenomHash,
			FactoryDenomAmount:   factoryDenomAmount,
			KCoin:                kCoin,
			KCoinAmount:          kCoinAmount,
			PoolFee:              poolFee,
			UnlockInSeconds:      unlockSeconds,
		})
		return err
	})
}

func DissolvePool(ctx context.Context, msgServer types.MsgServer, creator, factoryDenomHash string) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := msgServer.DissolvePool(innerCtx, &types.MsgDissolvePool{
			Creator:              creator,
			FullFactoryDenomName: factoryDenomHash,
		})
		return err
	})
}

func UpdateLiquidityPoolSettings(ctx context.Context, msgServer types.MsgServer, creator, factoryDenomHash, poolFee string, unlockSeconds uint64) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := msgServer.UpdateLiquidityPoolSettings(innerCtx, &types.MsgUpdateLiquidityPoolSettings{
			Creator:              creator,
			FullFactoryDenomName: factoryDenomHash,
			PoolFee:              poolFee,
			UnlockInSeconds:      unlockSeconds,
		})
		return err
	})
}

func AddFactoryLiquidity(ctx context.Context, msgServer types.MsgServer, creator, factoryDenomHash, factoryDenomAmount string) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := msgServer.AddLiquidity(innerCtx, &types.MsgAddLiquidity{
			Creator:              creator,
			FullFactoryDenomName: factoryDenomHash,
			FactoryDenomAmount:   factoryDenomAmount,
		})
		return err
	})
}

func UnlockLiquidity(ctx context.Context, msgServer types.MsgServer, creator, factoryDenomHash, factoryDenomAmount string) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := msgServer.UnlockLiquidity(innerCtx, &types.MsgUnlockLiquidity{
			Creator:              creator,
			FullFactoryDenomName: factoryDenomHash,
			FactoryDenomAmount:   factoryDenomAmount,
		})
		return err
	})
}

func FactoryDenomSell(ctx context.Context, msgServer types.MsgServer, creator, factoryDenomHash, denomGiving, denomReceiving, amount, maxPrice string, allowIncomplete bool) (*types.MsgTradeResponse, error) {
	var response *types.MsgTradeResponse
	var err error

	err = cache.Transact(ctx, func(innerCtx context.Context) error {
		response, err = msgServer.Sell(innerCtx, &types.MsgSell{
			Creator:              creator,
			FullFactoryDenomName: factoryDenomHash,
			DenomGiving:          denomGiving,
			DenomReceiving:       denomReceiving,
			Amount:               amount,
			MaxPrice:             maxPrice,
			AllowIncomplete:      allowIncomplete,
		})
		return err
	})

	return response, err
}

func FactoryDenomBuy(ctx context.Context, msgServer types.MsgServer, creator, factoryDenomHash, denomGiving, denomReceiving, amount, maxPrice string, allowIncomplete bool) (*types.MsgTradeResponse, error) {
	var response *types.MsgTradeResponse
	var err error

	err = cache.Transact(ctx, func(innerCtx context.Context) error {
		response, err = msgServer.Buy(innerCtx, &types.MsgBuy{
			Creator:              creator,
			FullFactoryDenomName: factoryDenomHash,
			DenomGiving:          denomGiving,
			DenomReceiving:       denomReceiving,
			Amount:               amount,
			MaxPrice:             maxPrice,
			AllowIncomplete:      allowIncomplete,
		})
		return err
	})

	return response, err
}

func FactoryDenomBuyback(ctx context.Context, msgServer types.MsgServer, creator, factoryDenomHash, amount string) error {
	var err error

	err = cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err = msgServer.Buyback(innerCtx, &types.MsgBuyback{
			Creator:              creator,
			FullFactoryDenomName: factoryDenomHash,
			BuybackAmount:        amount,
		})
		return err
	})

	return err
}
