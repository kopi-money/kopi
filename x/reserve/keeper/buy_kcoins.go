package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/cache"
	"github.com/kopi-money/kopi/trading"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/reserve/types"
)

func (k Keeper) HandleKCoinBuyback(ctx context.Context) {
	addressBurner := k.AccountKeeper.GetModuleAccount(ctx, types.Burner).GetAddress()
	addressBuying := k.AccountKeeper.GetModuleAccount(ctx, types.BuyingKCoins).GetAddress()
	spendableCoins := k.BankKeeper.SpendableCoins(ctx, addressBuying)

	kCoin := k.getBlockKCoin(ctx)
	for _, coin := range spendableCoins {
		k.handleKCoinBuyback(ctx, coin, addressBuying.String(), addressBurner.String(), kCoin)
	}
}

// getBlockKCoin returns a different kCoin at each block so that not the same one is bought each block
func (k Keeper) getBlockKCoin(ctx context.Context) string {
	kCoins := k.DenomKeeper.KCoins(ctx)
	height := int(sdk.UnwrapSDKContext(ctx).BlockHeight())
	return kCoins[height%len(kCoins)]
}

func (k Keeper) handleKCoinBuyback(ctx context.Context, coin sdk.Coin, addressBuying, addressBurner, kCoin string) {
	if k.DenomKeeper.IsKCoin(ctx, coin.Denom) {
		k.burnKCoin(ctx, coin)
	} else {
		k.buyKCoin(ctx, coin, addressBuying, addressBurner, kCoin)
	}
}

func (k Keeper) burnKCoin(ctx context.Context, coin sdk.Coin) {
	_ = k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.BuyingKCoins, types.Burner, sdk.NewCoins(coin))

	amountUSD, _ := k.DenomKeeper.GetValueInUSD(ctx, coin.Denom, coin.Amount.ToLegacyDec())

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("kcoin_burn",
			sdk.Attribute{Key: "kcoin", Value: coin.Denom},
			sdk.Attribute{Key: "amount", Value: coin.Amount.String()},
			sdk.Attribute{Key: "amount_usd", Value: amountUSD.String()},
		),
	)
}

func (k Keeper) buyKCoin(ctx context.Context, sellingCoin sdk.Coin, addressBuying, addressBurner, kCoin string) {
	sellAmount, has := k.getSellAmount(ctx, sellingCoin.Denom)
	if !has {
		return
	}

	if sellingCoin.Amount.LT(sellAmount) {
		return
	}

	var tradeResult trading.TradeResult
	if err := cache.TransactWithNewMultiStore(ctx, func(innerCtx context.Context) error {
		var (
			zero = math.ZeroInt()
			err  error
		)

		tradeResult, err = k.DexKeeper.ExecuteSell(dextypes.TradeContext{
			Context:             ctx,
			TradeAmount:         sellAmount,
			TradeDenomGiving:    sellingCoin.Denom,
			TradeDenomReceiving: kCoin,
			ProtocolTrade:       true,
			MinimumTradeAmount:  &zero,
			CoinSource:          addressBuying,
			CoinTarget:          addressBurner,
		})
		return err
	}); err != nil {
		k.Logger().Error(err.Error())
		return
	}

	amountUSD, _ := k.DenomKeeper.GetValueInUSD(ctx, kCoin, tradeResult.AmountReceived().ToLegacyDec())

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("kcoin_buyback",
			sdk.Attribute{Key: "sold_denom", Value: sellingCoin.Denom},
			sdk.Attribute{Key: "kcoin", Value: kCoin},
			sdk.Attribute{Key: "amount_sold", Value: tradeResult.AmountGiven().String()},
			sdk.Attribute{Key: "amount_bought", Value: tradeResult.AmountReceived().String()},
			sdk.Attribute{Key: "amount_bought_usd", Value: amountUSD.String()},
		),
	)
}
