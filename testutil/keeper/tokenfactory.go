package keeper

import (
	"context"
	denomkeeper "github.com/kopi-money/kopi/x/denominations/keeper"
	"testing"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/cache"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

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
		dexKeeper.DenomKeeper.(denomkeeper.Keeper),
		dexKeeper,
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
		MinimumPoolSize: math.NewInt(100),
		Categories: types.Categories{
			Categories: []types.Category{
				{
					Index:         0,
					Name:          "General",
					CreationPrice: math.NewInt(1_000_000),
				},
				{
					Index:         1,
					Name:          "Business",
					CreationPrice: math.NewInt(1_000_000),
				},
				{
					Index:         2,
					Name:          "IBC",
					CreationPrice: math.NewInt(1_000_000),
					IsIbc:         true,
				},
			},
		},
		ReserveFeeShare: math.LegacyNewDecWithPrec(5, 1),
	}
}

func SetupTokenfactoryMsgServer(t *testing.T) (keeper.Keeper, types.MsgServer, context.Context) {
	k, ctx := TokenfactoryKeeper(t)
	addFunds(ctx, k.BankKeeper.(bankkeeper.BaseKeeper), t)
	return k, keeper.NewMsgServerImpl(k), ctx
}

func CreateFactoryDenom(ctx context.Context, msgServer types.MsgServer, creator, name, symbol string, exponent uint64) (string, error) {
	var factoryDenomHash string
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		response, err := msgServer.CreateDenom(innerCtx, &types.MsgCreateDenom{
			Creator:  creator,
			Name:     name,
			Exponent: exponent,
			Symbol:   symbol,
			Mintable: true,
		})

		if err != nil {
			return err
		}

		factoryDenomHash = response.FullName
		return nil
	})

	return factoryDenomHash, err
}

func CreateFactoryDenomFromLocal(ctx context.Context, msgServer types.MsgServer, creator, name, localName, symbol string, categoryIndex, exponent uint64) (string, error) {
	var factoryDenomHash string
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		response, err := msgServer.CreateDenom(innerCtx, &types.MsgCreateDenom{
			Creator:       creator,
			Name:          name,
			Exponent:      exponent,
			LocalName:     localName,
			Symbol:        symbol,
			CategoryIndex: categoryIndex,
			Mintable:      false,
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

func AddFactoryLiquidity(ctx context.Context, msgServer types.MsgServer, creator, factoryDenomHash, factoryDenomAmount, maximumKCoinAmount string) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := msgServer.AddLiquidity(innerCtx, &types.MsgAddLiquidity{
			Creator:              creator,
			FullFactoryDenomName: factoryDenomHash,
			FactoryDenomAmount:   factoryDenomAmount,
			MaximumKcoinAmount:   maximumKCoinAmount,
		})
		return err
	})
}

func AddOneSidedFactoryLiquidity(ctx context.Context, msgServer types.MsgServer, creator, factoryDenomHash, factoryDenomAmount string) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := msgServer.AddFactoryLiquidity(innerCtx, &types.MsgAddFactoryLiquidity{
			Creator:              creator,
			FullFactoryDenomName: factoryDenomHash,
			Amount:               factoryDenomAmount,
		})
		return err
	})
}

func AddOneSidedKCoinLiquidity(ctx context.Context, msgServer types.MsgServer, creator, factoryDenomHash, kCoinAmount string) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		_, err := msgServer.AddKCoinLiquidity(innerCtx, &types.MsgAddKCoinLiquidity{
			Creator:              creator,
			FullFactoryDenomName: factoryDenomHash,
			Amount:               kCoinAmount,
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

func FactoryDenomSell(ctx context.Context, msgServer types.MsgServer, creator, factoryDenomHash, denomGiving, denomReceiving, amount, maxPrice string, feeIncluded bool) (*types.MsgTradeResponse, error) {
	var (
		response *types.MsgTradeResponse
		mp       *types.MaxPrice
		err      error
	)

	if maxPrice != "" {
		mp = &types.MaxPrice{
			MaxPrice:    maxPrice,
			FeeIncluded: feeIncluded,
		}
	}

	err = cache.Transact(ctx, func(innerCtx context.Context) error {
		response, err = msgServer.Sell(innerCtx, &types.MsgSell{
			Creator:              creator,
			FullFactoryDenomName: factoryDenomHash,
			DenomGiving:          denomGiving,
			DenomReceiving:       denomReceiving,
			Amount:               amount,
			MaxPrice:             mp,
		})
		return err
	})

	return response, err
}

func FactoryDenomBuy(ctx context.Context, msgServer types.MsgServer, creator, factoryDenomHash, denomGiving, denomReceiving, amount, maxPrice string, feeIncluded bool) (*types.MsgTradeResponse, error) {
	var (
		response *types.MsgTradeResponse
		mp       *types.MaxPrice
		err      error
	)

	if maxPrice != "" {
		mp = &types.MaxPrice{
			MaxPrice:    maxPrice,
			FeeIncluded: feeIncluded,
		}
	}

	err = cache.Transact(ctx, func(innerCtx context.Context) error {
		response, err = msgServer.Buy(innerCtx, &types.MsgBuy{
			Creator:              creator,
			FullFactoryDenomName: factoryDenomHash,
			DenomGiving:          denomGiving,
			DenomReceiving:       denomReceiving,
			Amount:               amount,
			MaxPrice:             mp,
		})
		return err
	})

	return response, err
}

func FactoryDenomBuyback(ctx context.Context, msgServer types.MsgServer, creator, factoryDenomHash, amount string) (*types.MsgBuybackResponse, error) {
	var (
		res *types.MsgBuybackResponse
		err error
	)

	if err = cache.Transact(ctx, func(innerCtx context.Context) error {
		res, err = msgServer.Buyback(innerCtx, &types.MsgBuyback{
			Creator:              creator,
			FullFactoryDenomName: factoryDenomHash,
			BuybackAmount:        amount,
		})
		return err
	}); err != nil {
		return nil, err
	}

	return res, nil
}
