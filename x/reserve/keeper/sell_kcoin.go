package keeper

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) SellKCoins(ctx context.Context) error {
	for _, kCoin := range k.DenomKeeper.KCoins(ctx) {
		if err := k.sellKCoin(ctx, kCoin); err != nil {
			return fmt.Errorf("could not sell kcoin  %s: %w", kCoin, err)
		}
	}

	return nil
}

func (k Keeper) sellKCoin(ctx context.Context, kCoin string) error {
	parity, referenceDenom, err := k.DexKeeper.CalculateParity(ctx, kCoin)
	if err != nil {
		return fmt.Errorf("could not calculate parity: %w", err)
	}

	if parity == nil {
		return nil
	}

	if parity.LT(k.sellThreshold(ctx)) {
		return nil
	}

	moduleAddr := k.AccountKeeper.GetModuleAccount(ctx, types.PoolReserve).GetAddress()
	tradeAmount := k.DenomKeeper.MaxBurnAmount(ctx, kCoin)
	amountLiquidity := k.DexKeeper.GetLiquidityByAddress(ctx, kCoin, moduleAddr.String())
	if amountLiquidity.LT(tradeAmount) {
		return nil
	}

	if err = k.DexKeeper.RemoveLiquidityForAddress(ctx, moduleAddr, kCoin, tradeAmount); err != nil {
		return fmt.Errorf("could not remove liquidity for %s: %w", kCoin, err)
	}

	tradeCtx := types.TradeContext{
		Context:             ctx,
		CoinSource:          moduleAddr.String(),
		CoinTarget:          moduleAddr.String(),
		TradeAmount:         tradeAmount,
		TradeDenomGiving:    kCoin,
		TradeDenomReceiving: referenceDenom,
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

func (k Keeper) readdLiquidity(ctx context.Context, moduleAddr sdk.AccAddress, kCoin, referenceDenom string) error {
	spendableBalance := k.BankKeeper.SpendableCoins(ctx, moduleAddr)
	balanceKCoin := spendableBalance.AmountOf(kCoin)
	balanceReferenceDenom := spendableBalance.AmountOf(referenceDenom)

	if balanceKCoin.IsPositive() {
		if _, err := k.DexKeeper.AddLiquidity(ctx, moduleAddr, kCoin, balanceKCoin); err != nil {
			return fmt.Errorf("could not add liquidity for %s: %w", kCoin, err)
		}
	}

	if balanceReferenceDenom.IsPositive() {
		if _, err := k.DexKeeper.AddLiquidity(ctx, moduleAddr, referenceDenom, balanceReferenceDenom); err != nil {
			return fmt.Errorf("could not add liquidity for %s: %w", balanceReferenceDenom, err)
		}
	}

	return nil
}
