package keeper

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/math"

	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/strategies/types"
)

func (k Keeper) getArbitrageDenom(ctx context.Context, denom string) types.ArbitrageDenom {
	arbitrageDenom, has := k.arbitrageDenoms.Get(ctx, denom)
	if has {
		return arbitrageDenom
	}

	return types.ArbitrageDenom{
		Name:        denom,
		KCoinAmount: math.ZeroInt(),
	}
}

func (k Keeper) updateArbitrageDenomBalance(ctx context.Context, denom string, change math.Int) {
	arbitrageDenom := k.getArbitrageDenom(ctx, denom)
	arbitrageDenom.KCoinAmount = arbitrageDenom.KCoinAmount.Add(change)
	k.arbitrageDenoms.Set(ctx, denom, arbitrageDenom)
}

func (k Keeper) HandleArbitrageDenoms(ctx context.Context) error {
	arbitrageDenoms := k.DenomKeeper.GetArbitrageDenoms(ctx)
	if len(arbitrageDenoms) == 0 {
		return nil
	}

	arbitrageDenoms = shiftDenomOrder(ctx, arbitrageDenoms)
	for _, arbitrageDenom := range arbitrageDenoms {
		if err := k.handleArbitrageDenom(ctx, arbitrageDenom); err != nil {
			return fmt.Errorf("could not handle arbitrage denom %v: %w", arbitrageDenom.DexDenom, err)
		}
	}

	return nil
}

// shiftDenomOrder changes the execution order of the arbitrage denoms. If the first element in the list would always be
// executed first, it would mean that denom has more chances to use arbitrage opportunities than the other denoms. By
// flipping the order depending on the block height, every denom gets the chance to be first, result in a fair
// execution order.
func shiftDenomOrder(ctx context.Context, arbitrageDenoms []denomtypes.ArbitrageDenom) []denomtypes.ArbitrageDenom {
	shift := int(sdk.UnwrapSDKContext(ctx).BlockHeight()) % len(arbitrageDenoms)

	result := make([]denomtypes.ArbitrageDenom, len(arbitrageDenoms))
	for i, arbitrageDenom := range arbitrageDenoms {
		newIndex := (i + shift) % len(arbitrageDenoms)
		result[newIndex] = arbitrageDenom
	}

	return result
}

func (k Keeper) handleArbitrageDenom(ctx context.Context, arbitrageDenom denomtypes.ArbitrageDenom) error {
	parity, _, err := k.DexKeeper.CalculateParity(ctx, arbitrageDenom.KCoin)
	if err != nil {
		return fmt.Errorf("calculate parity: %w", err)
	}

	address := k.AccountKeeper.GetModuleAccount(ctx, types.PoolArbitrage).GetAddress()
	tradeBalances := dexkeeper.NewTradeBalances()
	balance := k.BankKeeper.SpendableCoins(ctx, address)

	// The kCoin is bought when its parity has fallen below the given threshold.
	if parity.LT(arbitrageDenom.BuyThreshold) {
		amountCAsset := balance.AmountOf(arbitrageDenom.CAsset)
		if amountCAsset.IsPositive() {
			zeroInt := math.ZeroInt()
			tradeCtx := dextypes.TradeContext{
				Context:             ctx,
				CoinSource:          address.String(),
				CoinTarget:          address.String(),
				TradeAmount:         arbitrageDenom.BuyTradeAmount,
				TradeDenomGiving:    arbitrageDenom.CAsset,
				TradeDenomReceiving: arbitrageDenom.KCoin,
				TradeBalances:       tradeBalances,
				MinimumTradeAmount:  &zeroInt,
				ProtocolTrade:       true,
			}

			var tradeResult dextypes.TradeResult
			tradeResult, err = k.DexKeeper.ExecuteBuy(tradeCtx)
			if err != nil {
				if errors.Is(err, dextypes.ErrTradeAmountTooSmall) {
					return nil
				}
				if errors.Is(err, dextypes.ErrNotEnoughLiquidity) {
					return nil
				}
				if errors.Is(err, dextypes.ErrNotEnoughFunds) {
					return nil
				}

				return fmt.Errorf("could not execute incomplete trade: %w", err)
			}

			k.updateArbitrageDenomBalance(ctx, arbitrageDenom.DexDenom, tradeResult.AmountReceived)
		}
	}

	// The kCoin is sold when its parity is above the given threshold
	// kCoin > cAsset
	if parity.GT(arbitrageDenom.SellThreshold) {
		amount := k.getArbitrageDenom(ctx, arbitrageDenom.DexDenom).KCoinAmount
		amount = math.MinInt(amount, arbitrageDenom.SellTradeAmount)

		if amount.IsPositive() {
			tradeCtx := dextypes.TradeContext{
				Context:                ctx,
				CoinSource:             address.String(),
				CoinTarget:             address.String(),
				TradeAmount:            amount,
				MaximumAvailableAmount: k.getArbitrageDenom(ctx, arbitrageDenom.DexDenom).KCoinAmount,
				TradeDenomGiving:       arbitrageDenom.KCoin,
				TradeDenomReceiving:    arbitrageDenom.CAsset,
				TradeBalances:          tradeBalances,
				ProtocolTrade:          true,
			}

			var tradeResult dextypes.TradeResult
			tradeResult, err = k.DexKeeper.ExecuteSell(tradeCtx)
			if err != nil {
				if errors.Is(err, dextypes.ErrTradeAmountTooSmall) {
					return nil
				}
				if errors.Is(err, dextypes.ErrNotEnoughLiquidity) {
					return nil
				}
				if errors.Is(err, dextypes.ErrNotEnoughFunds) {
					return nil
				}

				return fmt.Errorf("could not execute incomplete trade: %w", err)
			}

			k.updateArbitrageDenomBalance(ctx, arbitrageDenom.DexDenom, tradeResult.AmountGiven.Neg())
		}
	}

	if err = tradeBalances.Settle(ctx, k.BankKeeper); err != nil {
		return fmt.Errorf("could not settle trades: %w", err)
	}

	return nil
}
