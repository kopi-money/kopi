package keeper

import (
	"context"
	"sync"
	"testing"

	"github.com/kopi-money/kopi/cache"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/stretchr/testify/require"
)

const (
	Alice = "kopi1zwfsl2deqq0cgajfzn4ts03d6rmv5z7z9q6at5"
	Bob   = "kopi1cgxt4umyzmuupaem0t4azuvg5ca02mtm42cyxa"
	Carol = "kopi1a622gyh8e95mkxhumtv7j3umky32vjq0c84zuv"
	Dave  = "kopi1ktr8krut00d7yg43nr7tfhksqgape4789gqgfc"
)

var initConfig sync.Once

func initSDKConfig() {
	initConfig.Do(func() {
		// Set prefixes
		accountPubKeyPrefix := constants.Bech32PrefixAccAddr + "pub"
		validatorAddressPrefix := constants.Bech32PrefixAccAddr + "valoper"
		validatorPubKeyPrefix := constants.Bech32PrefixAccAddr + "valoperpub"
		consNodeAddressPrefix := constants.Bech32PrefixAccAddr + "valcons"
		consNodePubKeyPrefix := constants.Bech32PrefixAccAddr + "valconspub"

		// Set and seal config
		config := sdk.GetConfig()
		config.SetBech32PrefixForAccount(constants.Bech32PrefixAccAddr, accountPubKeyPrefix)
		config.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
		config.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
		config.Seal()
	})
}

func addFunds(ctx context.Context, k TestBankKeeper, t *testing.T) {
	addresses := []string{Alice, Bob, Carol}
	denoms := []string{
		"ukopi",
		"ukusd",
		"uwusdc",
		"ukbtc",
		"uwbtc",
		"ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5",
	}

	for _, address := range addresses {
		for _, denom := range denoms {
			AddFunds(ctx, t, k, denom, address, int64(100_000_000_000))
		}
	}
}

type TestBankKeeper interface {
	MintCoins(context.Context, string, sdk.Coins) error
	SendCoinsFromModuleToAccount(context.Context, string, sdk.AccAddress, sdk.Coins) error
}

func AddFunds(ctx context.Context, t *testing.T, k TestBankKeeper, denom, address string, amount int64) {
	coin := sdk.NewCoin(denom, sdkmath.LegacyNewDec(amount).RoundInt())
	coins := sdk.NewCoins(coin)
	err := k.MintCoins(ctx, dextypes.PoolReserve, coins)
	require.NoError(t, err)
	addr, err := sdk.AccAddressFromBech32(address)
	require.NoError(t, err)
	require.NoError(t, k.SendCoinsFromModuleToAccount(ctx, dextypes.PoolReserve, addr, coins))
}

type SetLiquidityBankKeeper interface {
	TestBankKeeper

	BurnCoins(ctx context.Context, name string, amt sdk.Coins) error
	SendCoinsFromModuleToModule(context.Context, string, string, sdk.Coins) error
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}

func SetLiquidity(ctx context.Context, k SetLiquidityBankKeeper, dexkeeper dexkeeper.Keeper, t *testing.T, pool map[string]int64) {
	acc := dexkeeper.AccountKeeper.GetModuleAccount(ctx, dextypes.PoolLiquidity)
	existingCoins := k.SpendableCoins(ctx, acc.GetAddress())

	require.NoError(t, k.SendCoinsFromModuleToModule(ctx, dextypes.PoolLiquidity, dextypes.PoolReserve, existingCoins))
	require.NoError(t, k.BurnCoins(ctx, dextypes.PoolReserve, existingCoins))

	for denom, amount := range pool {
		AddFunds(ctx, t, k, denom, acc.GetAddress().String(), amount)

		require.NoError(t, cache.Transact(ctx, func(innerCtx context.Context) error {
			dexkeeper.SetLiquidity(innerCtx, denom, dextypes.Liquidity{Address: Alice, Amount: sdkmath.NewInt(amount)})
			return nil
		}))
	}
}
