package keeper

import (
	"context"
	"math"
	"strconv"
	"sync"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/kopi-money/kopi/utils"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/stretchr/testify/require"
)

const (
	Alice = "kopi1zwfsl2deqq0cgajfzn4ts03d6rmv5z7z9q6at5"
	Bob   = "kopi1cgxt4umyzmuupaem0t4azuvg5ca02mtm42cyxa"
	Carol = "kopi1a622gyh8e95mkxhumtv7j3umky32vjq0c84zuv"
)

var initConfig sync.Once

func initSDKConfig() {
	initConfig.Do(func() {
		// Set prefixes
		accountPubKeyPrefix := utils.Bech32PrefixAccAddr + "pub"
		validatorAddressPrefix := utils.Bech32PrefixAccAddr + "valoper"
		validatorPubKeyPrefix := utils.Bech32PrefixAccAddr + "valoperpub"
		consNodeAddressPrefix := utils.Bech32PrefixAccAddr + "valcons"
		consNodePubKeyPrefix := utils.Bech32PrefixAccAddr + "valconspub"

		// Set and seal config
		config := sdk.GetConfig()
		config.SetBech32PrefixForAccount(utils.Bech32PrefixAccAddr, accountPubKeyPrefix)
		config.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
		config.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
		config.Seal()
	})
}

func Pow(amount int64) int64 {
	fac := int64(math.Pow(10, float64(utils.DecimalPlaces)))
	return amount * fac
}

func IntString(amount int64) string {
	return strconv.Itoa(int(amount))
}

func PowInt64String(amount int64) string {
	return IntString(PowInt64(amount))
}

func PowInt64(amount int64) int64 {
	return sdkmath.NewInt(Pow(amount)).Int64()
}

func PowDec(amount int64) sdkmath.LegacyDec {
	return sdkmath.LegacyNewDec(Pow(amount))
}

func addFunds(ctx context.Context, k bankkeeper.BaseKeeper, t *testing.T) {
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
			AddFunds(t, ctx, k, denom, address, int64(100_000_000_000))
		}
	}
}

type TestBankKeeper interface {
	MintCoins(context.Context, string, sdk.Coins) error
	SendCoinsFromModuleToAccount(context.Context, string, sdk.AccAddress, sdk.Coins) error
}

func AddFunds(t *testing.T, ctx context.Context, k TestBankKeeper, denom, address string, amount int64) {
	coin := sdk.NewCoin(denom, sdkmath.LegacyNewDec(amount).RoundInt())
	coins := sdk.NewCoins(coin)
	err := k.MintCoins(ctx, dextypes.PoolReserve, coins)
	require.NoError(t, err)
	addr, err := sdk.AccAddressFromBech32(address)
	require.NoError(t, err)
	require.NoError(t, k.SendCoinsFromModuleToAccount(ctx, dextypes.PoolReserve, addr, coins))
}
