package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/dex/types"
	mmtypes "github.com/kopi-money/kopi/x/mm/types"
	"github.com/pkg/errors"
)

func (k Keeper) BeginBlockCheckReserve(ctx context.Context, eventManager sdk.EventManagerI, blockHeight int64) error {
	// It's not necessary to add micro amounts to the dex, thus we wait a while to add bigger chunks
	if blockHeight%1000 == 1 {
		firstBlock := blockHeight == 1
		return k.CheckReserve(ctx, eventManager, firstBlock)
	}

	return nil
}

// CheckReserve checks whether the reserve has funds that have not been added to the dex yet and if yes, adds those
// funds to the dex. When the denomination of coins is virtual, it is checked whether the kCoinination is above
// parity. When not, those coins are not added to the dex. First, the base currency is handled, after that all other
// currencies.
func (k Keeper) CheckReserve(ctx context.Context, eventManager sdk.EventManagerI, firstBlock bool) error {
	address := k.AccountKeeper.GetModuleAccount(ctx, types.PoolReserve).GetAddress()
	coins := k.BankKeeper.SpendableCoins(ctx, address)

	found, base := coins.Find(utils.BaseCurrency)
	if found {
		if err := k.checkReserveForDenom(ctx, eventManager, address, base, firstBlock); err != nil {
			return errors.Wrap(err, "error checking reserve for base currency")
		}
	}

	for _, coin := range coins {
		if coin.Denom == utils.BaseCurrency {
			continue
		}

		// Preventing issue where protocol wants to add liquidity before denom has been whitelisted
		if k.DenomKeeper.IsValidDenom(ctx, coin.Denom) {
			continue
		}

		if err := k.checkReserveForDenom(ctx, eventManager, address, coin, false); err != nil {
			return errors.Wrap(err, fmt.Sprintf("error checking reserve for %v", coin.Denom))
		}
	}

	return nil
}

func (k Keeper) checkReserveForDenom(ctx context.Context, eventManager sdk.EventManagerI, address sdk.AccAddress, coin sdk.Coin, firstBlock bool) error {
	if coin.Amount.Equal(math.ZeroInt()) {
		return nil
	}

	// If the denom is a borrowable denom, part of the reserve is sent to the money market to incentivize deposits
	if CAsset, _ := k.DenomKeeper.GetCAssetByBaseName(ctx, coin.Denom); CAsset != nil {
		amountSent := k.sendToMoneyMarket(ctx, coin, CAsset.DexFeeShare)
		coin.Amount = coin.Amount.Sub(amountSent)
	}

	// If the coins are kCoins, they are burned
	if k.DenomKeeper.IsKCoin(ctx, coin.Denom) {
		if err := k.BankKeeper.BurnCoins(ctx, types.PoolReserve, sdk.NewCoins(coin)); err != nil {
			return errors.Wrap(err, "could not burn coins")
		}

		return nil
	}

	if err := k.AddLiquidity(ctx, eventManager, address, coin.Denom, coin.Amount); err != nil {
		return errors.Wrap(err, "could not add liquidity")
	}

	return nil
}

func (k Keeper) sendToMoneyMarket(ctx context.Context, coin sdk.Coin, dexFeeShare math.LegacyDec) math.Int {
	sendAmount := math.LegacyNewDecFromInt(coin.Amount).Mul(dexFeeShare).RoundInt()

	coins := sdk.NewCoins(sdk.NewCoin(coin.Denom, sendAmount))
	_ = k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolReserve, mmtypes.PoolVault, coins)

	return sendAmount
}
