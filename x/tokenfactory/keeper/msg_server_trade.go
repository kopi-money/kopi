package keeper

import (
	"context"
	"fmt"

	reservetypes "github.com/kopi-money/kopi/x/reserve/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/trading"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

type CalcAmount func(math.LegacyDec, math.LegacyDec, math.LegacyDec, math.LegacyDec, bool) math.LegacyDec

func (k msgServer) Sell(ctx context.Context, msg *types.MsgSell) (*types.MsgTradeResponse, error) {
	return k.handleTrade(ctx, msg, trading.SellCallbacks())
}

func (k msgServer) Buy(ctx context.Context, msg *types.MsgBuy) (*types.MsgTradeResponse, error) {
	return k.handleTrade(ctx, msg, trading.BuyCallbacks())
}

func (k msgServer) handleTrade(ctx context.Context, msg types.MsgTrade, callbacks trading.Callbacks) (*types.MsgTradeResponse, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.GetFullFactoryDenomName())
	if !has {
		return nil, types.ErrDenomDoesNotExists
	}

	pool, has := k.liquidityPools.Get(ctx, factoryDenom.FullName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	tradeAmount, err := trading.ParseAmount(msg.GetAmount())
	if err != nil {
		return nil, err
	}

	var maxPrice *trading.MaxPriceData
	if msg.GetMaxPrice() != nil {
		mpValue, err := math.LegacyNewDecFromStr(msg.GetMaxPrice().MaxPrice)
		if err != nil {
			return nil, types.ErrInvalidMaxPriceFormat
		}

		maxPrice = &trading.MaxPriceData{
			MaxPrice:    mpValue,
			FeeIncluded: msg.GetMaxPrice().FeeIncluded,
		}
	}

	minimumTradeAmount, err := trading.ParseMinimumTradeAmount(msg.GetMinimumTradeAmount())
	if err != nil {
		return nil, fmt.Errorf("invalid minimum trade amount(%v): %w", msg.GetMinimumTradeAmount(), err)
	}

	tradeContext := types.TradeContext{
		Context: ctx,

		MaxPrice:           maxPrice,
		TradeAmount:        tradeAmount,
		Callbacks:          callbacks,
		MinimumTradeAmount: minimumTradeAmount,
		Pool:               pool,
		DenomGiving:        factoryDenom.ReplaceWithFactoryTradeDenom(msg.GetDenomGiving()),
		DenomReceiving:     factoryDenom.ReplaceWithFactoryTradeDenom(msg.GetDenomReceiving()),
		Creator:            msg.GetCreator(),
	}

	return k.Trade(tradeContext, factoryDenom)
}

func (k Keeper) Trade(ctx types.TradeContext, factoryDenom types.FactoryDenom) (*types.MsgTradeResponse, error) {
	acc, _ := sdk.AccAddressFromBech32(ctx.Creator)

	if ctx.DenomGiving == ctx.DenomReceiving {
		return nil, types.ErrSameDenom
	}

	tradeData := ctx.ToTradeData()

	var err error
	tradeData.TradeAmount, err = trading.HandleMaxPrice(tradeData, trading.DecreaseMaxPrice)
	if err != nil {
		return nil, err
	}

	tradeResult, err := trading.Trade(tradeData)
	if err != nil {
		return nil, err
	}

	if k.BankKeeper.SpendableCoin(ctx, acc, ctx.DenomGiving).Amount.LT(tradeResult.AmountGiven()) {
		return nil, types.ErrInsufficientFunds
	}

	coins := sdk.NewCoins(sdk.NewCoin(ctx.DenomGiving, tradeResult.AmountGiven()))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFactoryLiquidity, coins); err != nil {
		return nil, fmt.Errorf("send coins from account to liquidity pool: %w", err)
	}

	kCoinAmountBefore := ctx.Pool.KCoinAmount
	ctx.Pool.FactoryDenomAmount = getNewFactoryAmount(ctx.Pool, ctx.DenomGiving, tradeResult.AmountGiven(), tradeResult.AmountReceived())
	ctx.Pool.KCoinAmount = getNewKCoinAmount(ctx.Pool, ctx.DenomGiving, tradeResult.AmountGiven(), tradeResult.AmountReceived())

	var feeAmount math.Int
	if ctx.DenomGiving == ctx.Pool.KCoin {
		feeAmount = tradeResult.FeeGiving()
	} else {
		feeAmount = tradeResult.FeeReceiving()
	}

	feeAmountReserve, feeAmountPool, err := k.handleReserveFee(ctx, &ctx.Pool, feeAmount)
	if err != nil {
		return nil, fmt.Errorf("handle reserve fee: %w", err)
	}

	k.liquidityPools.Set(ctx, factoryDenom.FullName, ctx.Pool)

	coins = sdk.NewCoins(sdk.NewCoin(ctx.DenomReceiving, tradeResult.AmountReceived()))
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolFactoryLiquidity, acc, coins); err != nil {
		return nil, fmt.Errorf("send coins from liquidity pool to account: %w", err)
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("factory_trade",
			sdk.Attribute{Key: "denom_from", Value: ctx.DenomGiving},
			sdk.Attribute{Key: "denom_to", Value: ctx.DenomReceiving},
			sdk.Attribute{Key: "amount_given", Value: tradeResult.AmountGiven().String()},
			sdk.Attribute{Key: "amount_received", Value: tradeResult.AmountReceived().String()},
			sdk.Attribute{Key: "pool_size", Value: kCoinAmountBefore.String()},
			sdk.Attribute{Key: "fee_pool", Value: feeAmountPool.String()},
			sdk.Attribute{Key: "fee_reserve", Value: feeAmountReserve.String()},
			sdk.Attribute{Key: "address", Value: ctx.Creator},
		),
	)

	price, _ := tradeResult.PricePaidRounded()
	priceKCoin := getPriceKCoin(price, ctx.DenomGiving == ctx.Pool.KCoin)

	return &types.MsgTradeResponse{
		AmountGivenGross:    tradeResult.AmountGiven().String(),
		AmountGivenNet:      tradeResult.AmountGivenNet().String(),
		AmountReceivedGross: tradeResult.AmountReceivedGross().String(),
		AmountReceivedNet:   tradeResult.AmountReceived().String(),
		Fee:                 feeAmountPool.Add(feeAmountReserve).String(),
		FeePool:             feeAmountPool.String(),
		FeeReserve:          feeAmountReserve.String(),
		Price:               price.String(),
		PriceKcoin:          priceKCoin.String(),
	}, nil
}

func (k Keeper) handleReserveFee(ctx context.Context, pool *types.LiquidityPool, feeAmount math.Int) (math.Int, math.Int, error) {
	reserveFeeShare := k.getReserveFeeShare(ctx)
	feeAmountReserve := feeAmount.ToLegacyDec().Mul(reserveFeeShare).TruncateInt()
	feeAmountPool := feeAmount.Sub(feeAmountReserve)

	if feeAmountReserve.IsPositive() {
		coins := sdk.NewCoins(sdk.NewCoin(pool.KCoin, feeAmountReserve))
		if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolFactoryLiquidity, reservetypes.BuyingKCoins, coins); err != nil {
			return math.Int{}, math.Int{}, fmt.Errorf("send reserve fee to module: %w", err)
		}

		pool.KCoinAmount = pool.KCoinAmount.Sub(feeAmountReserve)
	}

	return feeAmountReserve, feeAmountPool, nil
}

func getNewFactoryAmount(pool types.LiquidityPool, denomFrom string, tradeAmountGross, amountToReceive math.Int) math.Int {
	if pool.KCoin == denomFrom {
		// kCoin -> FactoryDenom => Factory denom decreases
		return pool.FactoryDenomAmount.Sub(amountToReceive)
	} else {
		// FactoryDenom -> kCoin => Factory denom increases
		return pool.FactoryDenomAmount.Add(tradeAmountGross)
	}
}

func getNewKCoinAmount(pool types.LiquidityPool, denomFrom string, tradeAmountGross, amountToReceive math.Int) math.Int {
	if pool.KCoin == denomFrom {
		// kCoin -> FactoryDenom => kCoin amount increases
		return pool.KCoinAmount.Add(tradeAmountGross)
	} else {
		// FactoryDenom -> kCoin => kCoin denom decreases
		return pool.KCoinAmount.Sub(amountToReceive)
	}
}
