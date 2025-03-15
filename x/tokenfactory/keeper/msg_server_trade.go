package keeper

import (
	"context"
	"fmt"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/constant_product"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

type CalcAmount func(math.LegacyDec, math.LegacyDec, math.LegacyDec, math.LegacyDec, bool) math.LegacyDec

func (k msgServer) Sell(ctx context.Context, msg *types.MsgSell) (*types.MsgTradeResponse, error) {
	return k.Keeper.Sell(ctx, TradeData{
		factoryDenom:    msg.FullFactoryDenomName,
		creator:         msg.Creator,
		denomGiving:     msg.DenomGiving,
		denomReceiving:  msg.DenomReceiving,
		maxPrice:        msg.MaxPrice,
		tradeAmount:     msg.Amount,
		allowIncomplete: msg.AllowIncomplete,
	})
}

func (k Keeper) Sell(ctx context.Context, tradeData TradeData) (*types.MsgTradeResponse, error) {
	acc, _ := sdk.AccAddressFromBech32(tradeData.creator)

	factoryDenom, has := k.GetDenomByFullName(ctx, tradeData.factoryDenom)
	if !has {
		return nil, types.ErrDenomDoesNotExists
	}

	pool, has := k.liquidityPools.Get(ctx, factoryDenom.FullName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	if tradeData.denomGiving == tradeData.denomReceiving {
		return nil, types.ErrSameDenom
	}

	amountToGiveGross, err := dexkeeper.ParseAmount(tradeData.tradeAmount)
	if err != nil {
		return nil, fmt.Errorf("could not parse amount: %w", err)
	}

	if k.BankKeeper.SpendableCoin(ctx, acc, tradeData.denomGiving).Amount.LT(amountToGiveGross) {
		return nil, types.ErrInsufficientFunds
	}

	var (
		adjustPrice      AdjustMaxPrice
		feeDataReceiving = newFeeData()
		feeDataGiving    = newFeeData()
	)

	if tradeData.denomGiving == pool.KCoin {
		adjustPrice = DecreaseMaxPrice
	} else {
		adjustPrice = IncreaseMaxPrice
	}

	amountToGiveGross, _, err = k.HandleMaxPrice(ctx, tradeData, pool, amountToGiveGross, adjustPrice, constant_product.CalculateMaximumGivingOneStep)
	if err != nil {
		return nil, err
	}

	// If the trade is to sell a kCoin, the trade fee is subtracted from the amount to be sold.
	amountToGiveNet := amountToGiveGross
	if tradeData.denomGiving == pool.KCoin {
		feeDataGiving = k.calculateFees(ctx, pool, amountToGiveGross)
		amountToGiveNet = amountToGiveGross.Sub(feeDataGiving.Fee())
	}

	if amountToGiveNet.LT(math.NewInt(1000)) {
		return nil, types.ErrTradeAmountTooSmall
	}

	coins := sdk.NewCoins(sdk.NewCoin(tradeData.denomGiving, amountToGiveGross))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFactoryLiquidity, coins); err != nil {
		return nil, fmt.Errorf("could not send coins from account to liquidity pool: %w", err)
	}

	// If the trade is to sell a factory token, i.e. to buy a kCoin, the trade fee is subtracted from the amount to be received.
	amountToReceiveGross := constantProductSell(pool, tradeData.denomGiving, amountToGiveNet)
	amountToReceiveNet := amountToReceiveGross
	if tradeData.denomReceiving == pool.KCoin {
		feeDataReceiving = k.calculateFees(ctx, pool, amountToReceiveGross)
		amountToReceiveNet = amountToReceiveGross.Sub(feeDataReceiving.Fee())
	}

	feeData := getFeeData(feeDataGiving, feeDataReceiving)
	if feeData.feeReserve.IsPositive() {
		coins = sdk.NewCoins(sdk.NewCoin(pool.KCoin, feeData.feeReserve))
		if err = k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolFactoryLiquidity, dextypes.PoolReserve, coins); err != nil {
			return nil, fmt.Errorf("could not send reserve fee to module: %w", err)
		}
	}

	pool.FactoryDenomAmount = getNewFactoryAmount(pool, tradeData.denomGiving, amountToGiveNet, amountToReceiveNet)
	pool.KCoinAmount = getNewKCoinAmount(pool, tradeData.denomGiving, amountToGiveNet, amountToReceiveNet)
	pool.KCoinAmount = pool.KCoinAmount.Add(feeData.feePool)
	k.liquidityPools.Set(ctx, factoryDenom.FullName, pool)

	coins = sdk.NewCoins(sdk.NewCoin(tradeData.denomReceiving, amountToReceiveNet))
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolFactoryLiquidity, acc, coins); err != nil {
		return nil, fmt.Errorf("could not send coins from liquidity pool to account: %w", err)
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("factory_trade",
			sdk.Attribute{Key: "denom_from", Value: tradeData.denomGiving},
			sdk.Attribute{Key: "denom_to", Value: tradeData.denomReceiving},
			sdk.Attribute{Key: "amount_given", Value: amountToGiveGross.String()},
			sdk.Attribute{Key: "amount_received", Value: amountToReceiveNet.String()},
			sdk.Attribute{Key: "fee_pool", Value: feeData.feePool.String()},
			sdk.Attribute{Key: "fee_reserve", Value: feeData.feeReserve.String()},
			sdk.Attribute{Key: "address", Value: tradeData.creator},
		),
	)

	var price math.LegacyDec
	if tradeData.denomReceiving == pool.KCoin {
		price = amountToReceiveNet.ToLegacyDec().Quo(amountToGiveGross.ToLegacyDec()) // C
	} else {
		price = amountToGiveGross.ToLegacyDec().Quo(amountToReceiveNet.ToLegacyDec()) // C
	}

	return &types.MsgTradeResponse{
		AmountGivenGross:    amountToGiveGross.String(),
		AmountGivenNet:      amountToGiveGross.Sub(feeDataGiving.Fee()).String(),
		AmountReceivedGross: amountToReceiveNet.Add(feeDataReceiving.Fee()).String(),
		AmountReceivedNet:   amountToReceiveNet.String(),
		Fee:                 feeData.Fee().String(),
		FeePool:             feeData.feePool.String(),
		FeeReserve:          feeData.feeReserve.String(),
		Price:               price.String(),
	}, nil
}

func (k msgServer) Buy(ctx context.Context, msg *types.MsgBuy) (*types.MsgTradeResponse, error) {
	return k.Keeper.Buy(ctx, TradeData{
		factoryDenom:    msg.FullFactoryDenomName,
		creator:         msg.Creator,
		denomGiving:     msg.DenomGiving,
		denomReceiving:  msg.DenomReceiving,
		maxPrice:        msg.MaxPrice,
		tradeAmount:     msg.Amount,
		allowIncomplete: msg.AllowIncomplete,
	})
}

func (k Keeper) Buy(ctx context.Context, tradeData TradeData) (*types.MsgTradeResponse, error) {
	acc, _ := sdk.AccAddressFromBech32(tradeData.creator)

	factoryDenom, has := k.GetDenomByFullName(ctx, tradeData.factoryDenom)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	pool, has := k.liquidityPools.Get(ctx, factoryDenom.FullName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	if tradeData.denomGiving == tradeData.denomReceiving {
		return nil, types.ErrSameDenom
	}

	amountToReceiveNet, err := dexkeeper.ParseAmount(tradeData.tradeAmount)
	if err != nil {
		return nil, fmt.Errorf("could not parse amount: %w", err)
	}

	var (
		adjustPrice      AdjustMaxPrice
		feeDataReceiving = newFeeData()
		feeDataGiving    = newFeeData()
		amountChanged    bool
	)

	// If the trade is to buy a kCoin, the amount to be bought has to be larger than requested because the amount used
	// for the trade fee has to be bought as well.
	amountToReceiveGross := amountToReceiveNet
	if tradeData.denomReceiving == pool.KCoin {
		feeDataReceiving = k.calculateFees(ctx, pool, amountToReceiveNet)
		amountToReceiveGross = amountToReceiveNet.Add(feeDataReceiving.Fee())
	}

	if tradeData.denomGiving == pool.KCoin {
		adjustPrice = IncreaseMaxPrice
	} else {
		adjustPrice = DecreaseMaxPrice
	}

	amountToReceiveGross, amountChanged, err = k.HandleMaxPrice(ctx, tradeData, pool, amountToReceiveGross, adjustPrice, constant_product.CalculateMaximumReceiving)
	if err != nil {
		return nil, err
	}

	// If the trade amount has changed given the max price, the calculated amount is used as gross amount. The trade fee
	// is subtracted to determine the net amount that will be received by the user.
	if amountChanged && tradeData.denomReceiving == pool.KCoin {
		feeDataReceiving = k.calculateFees(ctx, pool, amountToReceiveGross)
		amountToReceiveNet = amountToReceiveGross.Sub(feeDataReceiving.Fee())
	}

	if amountToReceiveGross.LT(math.NewInt(1000)) {
		return nil, types.ErrTradeAmountTooSmall
	}

	amountToGiveNet, err := constantProductBuy(pool, tradeData.denomGiving, amountToReceiveGross)
	if err != nil {
		return nil, err
	}

	// If the trade is to buy a factory token, the amount to give has to be larger than the calculated amount as to
	// cover the trade we.
	amountToGiveGross := amountToGiveNet
	if tradeData.denomGiving == pool.KCoin {
		feeDataGiving = k.calculateFees(ctx, pool, amountToGiveNet)
		amountToGiveGross = amountToGiveNet.Add(feeDataGiving.Fee())
	}

	if k.BankKeeper.SpendableCoin(ctx, acc, tradeData.denomGiving).Amount.LT(amountToGiveGross) {
		return nil, types.ErrInsufficientFunds
	}

	coins := sdk.NewCoins(sdk.NewCoin(tradeData.denomGiving, amountToGiveGross))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFactoryLiquidity, coins); err != nil {
		return nil, fmt.Errorf("send coins from account to liquidity pool: %w", err)
	}

	feeData := getFeeData(feeDataGiving, feeDataReceiving)
	if feeData.feeReserve.IsPositive() {
		coins = sdk.NewCoins(sdk.NewCoin(pool.KCoin, feeData.feeReserve))
		if err = k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolFactoryLiquidity, dextypes.PoolReserve, coins); err != nil {
			return nil, fmt.Errorf("could not send reserve fee to module: %w", err)
		}
	}

	pool.FactoryDenomAmount = getNewFactoryAmount(pool, tradeData.denomGiving, amountToGiveNet, amountToReceiveNet)
	pool.KCoinAmount = getNewKCoinAmount(pool, tradeData.denomGiving, amountToGiveNet, amountToReceiveNet)
	pool.KCoinAmount = pool.KCoinAmount.Add(feeData.feePool)
	k.liquidityPools.Set(ctx, factoryDenom.FullName, pool)

	coins = sdk.NewCoins(sdk.NewCoin(tradeData.denomReceiving, amountToReceiveNet))
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolFactoryLiquidity, acc, coins); err != nil {
		return nil, fmt.Errorf("send coins from liquidity pool to account: %w", err)
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("factory_trade",
			sdk.Attribute{Key: "denom_from", Value: tradeData.denomGiving},
			sdk.Attribute{Key: "denom_to", Value: tradeData.denomReceiving},
			sdk.Attribute{Key: "amount_given", Value: amountToGiveGross.String()},
			sdk.Attribute{Key: "amount_received", Value: amountToReceiveNet.String()},
			sdk.Attribute{Key: "fee_pool", Value: feeData.feePool.String()},
			sdk.Attribute{Key: "fee_reserve", Value: feeData.feeReserve.String()},
			sdk.Attribute{Key: "address", Value: tradeData.creator},
		),
	)

	var price math.LegacyDec
	if tradeData.denomReceiving == pool.KCoin {
		price = amountToReceiveNet.ToLegacyDec().Quo(amountToGiveGross.ToLegacyDec()) // C
	} else {
		price = amountToGiveGross.ToLegacyDec().Quo(amountToReceiveNet.ToLegacyDec()) // C
	}

	return &types.MsgTradeResponse{
		AmountGivenGross:    amountToGiveGross.String(),
		AmountGivenNet:      amountToGiveGross.Sub(feeDataGiving.Fee()).String(),
		AmountReceivedGross: amountToReceiveNet.Add(feeDataReceiving.Fee()).String(),
		AmountReceivedNet:   amountToReceiveNet.String(),
		Fee:                 feeData.Fee().String(),
		FeePool:             feeData.feePool.String(),
		FeeReserve:          feeData.feeReserve.String(),
		Price:               price.String(),
	}, nil
}

func (k Keeper) HandleMaxPrice(ctx context.Context, tradeData TradeData, pool types.LiquidityPool, amount math.Int, adjustMaxPrice AdjustMaxPrice, calculate constant_product.CalculateMaximumAmountOneStep) (math.Int, bool, error) {
	if tradeData.maxPrice == "" {
		return amount, false, nil
	}

	maxPrice, err := getMaxPrice(tradeData.maxPrice)
	if err != nil {
		return math.Int{}, false, err
	}

	var (
		priceTradeAmount math.Int
		amountChanged    bool
	)

	if tradeData.denomGiving != pool.KCoin {
		maxPrice = math.LegacyOneDec().Quo(maxPrice)
	}

	priceTradeAmount, err = k.calculateMaxAmount(ctx, pool, tradeData.denomGiving, maxPrice, pool.PoolFee, adjustMaxPrice, calculate)
	if err != nil {
		return math.Int{}, false, err
	}

	if priceTradeAmount.LT(amount) {
		if priceTradeAmount.IsNegative() || !tradeData.allowIncomplete {
			return math.Int{}, false, types.ErrMarketPriceTooHigh
		}

		amountChanged = true
		amount = priceTradeAmount
	}

	if !amount.IsPositive() {
		return math.Int{}, false, types.ErrEmptyTrade
	}

	return amount, amountChanged, nil
}

func getFeeData(feeDataGiving, feeDataReceiving FeeData) FeeData {
	if feeDataGiving.Fee().IsPositive() {
		return feeDataGiving
	} else {
		return feeDataReceiving
	}
}

type TradeData struct {
	factoryDenom    string
	creator         string
	denomGiving     string
	denomReceiving  string
	maxPrice        string
	tradeAmount     string
	allowIncomplete bool

	calcAmountToGive    CalcAmount
	calcAmountToReceive CalcAmount
}

func NewTradeData(factoryDenom, creator, denomGiving, denomReceiving, maxPrice, tradeAmount string, allowIncomplete bool) TradeData {
	return TradeData{
		factoryDenom:        factoryDenom,
		creator:             creator,
		denomGiving:         denomGiving,
		denomReceiving:      denomReceiving,
		maxPrice:            maxPrice,
		tradeAmount:         tradeAmount,
		allowIncomplete:     allowIncomplete,
		calcAmountToGive:    nil,
		calcAmountToReceive: nil,
	}
}

type FeeData struct {
	feePool    math.Int
	feeReserve math.Int
}

func newFeeData() FeeData {
	return FeeData{
		feePool:    math.ZeroInt(),
		feeReserve: math.ZeroInt(),
	}
}

func (fd FeeData) Fee() math.Int {
	return fd.feePool.Add(fd.feeReserve)
}

func (k Keeper) calculateFees(ctx context.Context, pool types.LiquidityPool, tradeAmount math.Int) FeeData {
	reserveFeeAmount := k.GetParams(ctx).ReserveFee.Mul(tradeAmount.ToLegacyDec()).TruncateInt()
	poolFeeAmount := pool.PoolFee.Mul(tradeAmount.ToLegacyDec()).TruncateInt()

	return FeeData{
		feePool:    poolFeeAmount,
		feeReserve: reserveFeeAmount,
	}
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

func constantProductSell(pool types.LiquidityPool, denomGiving string, amount math.Int) math.Int {
	liqFrom, liqTo := getLiquidity(pool, denomGiving)
	amountDec, _, _ := constant_product.ConstantProductTradeSell(liqFrom, liqTo, amount.ToLegacyDec(), math.LegacyZeroDec())
	return amountDec.TruncateInt()
}

func constantProductBuy(pool types.LiquidityPool, denomGiving string, amount math.Int) (math.Int, error) {
	liqFrom, liqTo := getLiquidity(pool, denomGiving)
	amountDec, _, err := constant_product.ConstantProductTradeBuy(liqFrom, liqTo, amount.ToLegacyDec(), math.LegacyZeroDec())
	if err != nil {
		return math.Int{}, types.ErrCannotBuyAmount
	}

	return amountDec.TruncateInt(), nil
}

type AdjustMaxPrice func(math.LegacyDec, math.LegacyDec) math.LegacyDec

func IncreaseMaxPrice(maxPrice, tradeFee math.LegacyDec) math.LegacyDec {
	return maxPrice.Quo(math.LegacyOneDec().Sub(tradeFee))
}

func DecreaseMaxPrice(maxPrice, tradeFee math.LegacyDec) math.LegacyDec {
	return maxPrice.Mul(math.LegacyOneDec().Sub(tradeFee))
}

func (k Keeper) calculateMaxAmount(ctx context.Context, pool types.LiquidityPool, denomFrom string, maxPrice, poolFee math.LegacyDec, adjustMaxPrice AdjustMaxPrice, calculate constant_product.CalculateMaximumAmountOneStep) (math.Int, error) {
	liqFrom, liqTo := getLiquidity(pool, denomFrom)
	tradeFee := k.getTradeFee(ctx, poolFee)
	maxPrice = adjustMaxPrice(maxPrice, tradeFee)

	priceTradeAmount, err := calculate(liqFrom, liqTo, maxPrice)
	if err != nil {
		return math.Int{}, err
	}

	return priceTradeAmount.TruncateInt(), nil
}

func getLiquidity(pool types.LiquidityPool, denomFrom string) (math.LegacyDec, math.LegacyDec) {
	var liqFrom, liqTo math.Int

	if denomFrom == pool.KCoin {
		liqFrom, liqTo = pool.KCoinAmount, pool.FactoryDenomAmount
	} else {
		liqFrom, liqTo = pool.FactoryDenomAmount, pool.KCoinAmount
	}

	return liqFrom.ToLegacyDec(), liqTo.ToLegacyDec()
}

func getMaxPrice(maxPriceString string) (math.LegacyDec, error) {
	maxPriceString = strings.ReplaceAll(maxPriceString, ",", "")
	maxPrice, err := math.LegacyNewDecFromStr(maxPriceString)
	if err != nil {
		return math.LegacyDec{}, types.ErrInvalidPriceFormat
	}

	if !maxPrice.IsPositive() {
		return math.LegacyDec{}, fmt.Errorf("max price is not positive")
	}

	return maxPrice, nil
}
