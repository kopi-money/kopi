package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/utils"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/swap/types"
	"github.com/pkg/errors"
)

// Mint is called at the end of each block to check whether the prices of the kCoins are higher than their
// "real" counterparts. If yes, funds for the kCoin are minted, the kCoin is sold for the base
// currency and received funds are burned such as to increase the supply of the kCoin and slightly decrease
// its price. The amount that is minted is limited depending on the currency to not mint too much per block.
func (k Keeper) Mint(ctx context.Context, eventManager sdk.EventManagerI) error {
	for _, kCoin := range k.DenomKeeper.KCoins(ctx) {
		maxMintAmount := k.DenomKeeper.MaxMintAmount(ctx, kCoin)
		if err := k.CheckMint(ctx, eventManager, kCoin, maxMintAmount); err != nil {
			return errors.Wrap(err, "could not mint denom")
		}
	}

	return nil
}

// CheckMint checks the parity of a given kCoin. If it is above 1, new coins are minted and sold in favor of
// the base currency.
func (k Keeper) CheckMint(ctx context.Context, eventManager sdk.EventManagerI, kCoin string, maxMintAmount math.Int) error {
	parity, referenceDenom, err := k.DexKeeper.CalculateParity(ctx, kCoin)
	if err != nil {
		return errors.Wrap(err, "could not calculate parity")
	}

	if parity == nil || parity.LT(math.LegacyOneDec()) {
		return nil
	}

	referenceRatio, _ := k.DexKeeper.GetRatio(ctx, referenceDenom)
	calculatedMintAmount := k.calcKCoinMintAmount(ctx, kCoin, *referenceRatio.Ratio)
	mintAmount := math.MinInt(calculatedMintAmount, maxMintAmount)

	mintAmount = k.adjustForSupplyCap(ctx, kCoin, mintAmount)
	if mintAmount.LTE(math.ZeroInt()) {
		return nil
	}

	// maxMintAmount is given in the denom of the kCoin's reference denom, which is why it's converted to
	// the kCoin
	mintAmount, _, _, err = k.DexKeeper.SimulateTradeForReserve(ctx, referenceDenom, kCoin, mintAmount)
	if err != nil {
		return errors.Wrap(err, "could not simulate trade")
	}

	if mintAmount.LT(math.OneInt()) {
		return nil
	}

	mintCoins := sdk.NewCoins(sdk.NewCoin(kCoin, mintAmount))
	if err = k.BankKeeper.MintCoins(ctx, types.ModuleName, mintCoins); err != nil {
		return errors.Wrap(err, "could not mint coins")
	}

	address := k.AccountKeeper.GetModuleAccount(ctx, types.ModuleName).GetAddress()

	options := dextypes.TradeOptions{
		CoinSource:          address,
		CoinTarget:          address,
		GivenAmount:         mintAmount,
		MaxPrice:            nil,
		TradeDenomStart:     kCoin,
		TradeDenomEnd:       utils.BaseCurrency,
		AllowIncomplete:     true,
		ExcludeFromDiscount: true,
		ProtocolTrade:       true,
	}

	amountUsed, amountReceived, _, _, err := k.DexKeeper.ExecuteTrade(ctx, eventManager, options)
	if err != nil {
		if errors.Is(err, dextypes.ErrTradeAmountTooSmall) {
			return nil
		}
		if errors.Is(err, dextypes.ErrNotEnoughLiquidity) {
			return nil
		}

		return errors.Wrap(err, "could not execute incomplete trade")
	}

	burnedAmount, err := k.burnFunds(ctx, eventManager, utils.BaseCurrency)
	if err != nil {
		return errors.Wrap(err, "could not burn funds")
	}

	eventManager.EmitEvent(
		sdk.NewEvent("arbitrage_trade",
			sdk.Attribute{Key: "denom_from", Value: options.TradeDenomStart},
			sdk.Attribute{Key: "denom_to", Value: options.TradeDenomEnd},
			sdk.Attribute{Key: "amount_used", Value: amountUsed.String()},
			sdk.Attribute{Key: "amount_received", Value: amountReceived.String()},
		),
	)

	eventManager.EmitEvent(
		sdk.NewEvent("swap_coins_minted",
			sdk.Attribute{Key: "denom", Value: kCoin},
			sdk.Attribute{Key: "amount", Value: mintAmount.String()},
		),
	)

	eventManager.EmitEvent(
		sdk.NewEvent("swap_coins_burned",
			sdk.Attribute{Key: "denom", Value: utils.BaseCurrency},
			sdk.Attribute{Key: "amount", Value: burnedAmount.String()},
		),
	)

	return nil
}

func (k Keeper) adjustForSupplyCap(ctx context.Context, kCoin string, amountToAdd math.Int) math.Int {
	supply := k.BankKeeper.GetSupply(ctx, kCoin).Amount
	maximumSupply := k.DenomKeeper.MaxSupply(ctx, kCoin)

	maximumAddableAmount := maximumSupply.Sub(supply.Add(amountToAdd))
	amountToAdd = math.MinInt(maximumAddableAmount, amountToAdd)

	return amountToAdd
}

func (k Keeper) calcKCoinMintAmount(ctx context.Context, kCoin string, referenceRatio math.LegacyDec) math.Int {
	liqBase := k.DexKeeper.GetFullLiquidityBase(ctx, kCoin)
	liqVirtual := k.DexKeeper.GetFullLiquidityOther(ctx, kCoin)
	constantProduct := liqBase.Mul(liqVirtual)
	newLiqVirtual, _ := constantProduct.Quo(referenceRatio).ApproxSqrt()
	mintAmount := newLiqVirtual.Sub(liqVirtual)
	return mintAmount.TruncateInt()
}

func (k Keeper) getUsableAmount(ctx context.Context, denom, module string) math.Int {
	address := k.AccountKeeper.GetModuleAccount(ctx, module).GetAddress()
	coins := k.BankKeeper.SpendableCoins(ctx, address)

	for _, coin := range coins {
		if coin.Denom == denom {
			return coin.Amount
		}
	}

	return math.ZeroInt()
}
