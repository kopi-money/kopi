package keeper

import (
	"context"
	"errors"
	"fmt"

	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) BuyKCoins(ctx context.Context) error {
	for _, kCoin := range k.DenomKeeper.KCoins(ctx) {
		if err := k.buyKCoin(ctx, kCoin); err != nil {
			return fmt.Errorf("could not sell kcoin  %s: %w", kCoin, err)
		}
	}

	return nil
}

func (k Keeper) buyKCoin(ctx context.Context, kCoin string) error {
	parity, referenceDenom, err := k.DexKeeper.CalculateParity(ctx, kCoin)
	if err != nil {
		return fmt.Errorf("could not calculate parity: %w", err)
	}

	if parity == nil {
		return nil
	}

	if parity.GT(k.buyThreshold(ctx)) {
		return nil
	}

	moduleAddr := k.AccountKeeper.GetModuleAccount(ctx, types.PoolReserve).GetAddress()
	tradeAmount := k.DenomKeeper.MaxMintAmount(ctx, kCoin)
	amountLiquidity := k.DexKeeper.GetLiquidityByAddress(ctx, referenceDenom, moduleAddr.String())
	if amountLiquidity.LT(tradeAmount) {
		return nil
	}

	if err = k.DexKeeper.RemoveLiquidityForAddress(ctx, moduleAddr, referenceDenom, tradeAmount); err != nil {
		return fmt.Errorf("could not remove liquidity for %s: %w", referenceDenom, err)
	}

	tradeCtx := types.TradeContext{
		Context:             ctx,
		CoinSource:          moduleAddr.String(),
		CoinTarget:          moduleAddr.String(),
		TradeAmount:         tradeAmount,
		TradeDenomGiving:    referenceDenom,
		TradeDenomReceiving: kCoin,
		ExcludeFromDiscount: true,
		ProtocolTrade:       true,
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	if _, err = k.DexKeeper.ExecuteSell(tradeCtx); err != nil {
		if errors.Is(err, types.ErrTradeAmountTooSmall) {
			return nil
		}
		if errors.Is(err, types.ErrNotEnoughLiquidity) {
			return nil
		}

		return fmt.Errorf("could not execute incomplete trade: %w", err)
	}

	if err = k.readdLiquidity(ctx, moduleAddr, kCoin, referenceDenom); err != nil {
		return err
	}

	return nil
}
