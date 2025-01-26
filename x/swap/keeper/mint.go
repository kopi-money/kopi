package keeper

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/constants"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/swap/types"
)

// Mint is called at the end of each block to check whether the prices of the kCoins are higher than their
// "real" counterparts. If yes, funds for the kCoin are minted, the kCoin is sold for the base
// currency and received funds are burned such as to increase the supply of the kCoin and slightly decrease
// its price. The amount that is minted is limited depending on the currency to not mint too much per block.
func (k Keeper) Mint(ctx context.Context) error {
	for _, kCoin := range k.DenomKeeper.KCoins(ctx) {
		maxMintAmount := k.DenomKeeper.MaxMintAmount(ctx, kCoin)
		if err := k.CheckMint(ctx, kCoin, maxMintAmount); err != nil {
			return fmt.Errorf("could not mint denom: %w", err)
		}
	}

	return nil
}

// CheckMint checks the parity of a given kCoin. If it is above 1, new coins are minted and sold in favor of
// the base currency.
func (k Keeper) CheckMint(ctx context.Context, kCoin string, maxMintAmount math.Int) error {
	parity, referenceDenom, err := k.DexKeeper.CalculateParity(ctx, kCoin)
	if err != nil {
		return fmt.Errorf("could not calculate parity: %w", err)
	}

	// parity can be nil at initialization of the chain when not all currencies have liquidity. It is an edge case.
	if parity == nil {
		return nil
	}

	if parity.LTE(k.mintThreshold(ctx)) {
		return nil
	}

	referenceRatio, err := k.DenomKeeper.GetRatio(ctx, referenceDenom)
	if err != nil {
		return fmt.Errorf("ratio: %w", err)
	}

	if !referenceRatio.Ratio.IsPositive() {
		return fmt.Errorf("ratio (%v) is not positive", referenceDenom)
	}

	mintAmount := k.adjustForSupplyCap(ctx, kCoin, maxMintAmount)
	if mintAmount.LTE(math.OneInt()) {
		return nil
	}

	mintCoins := sdk.NewCoins(sdk.NewCoin(kCoin, mintAmount))
	if err = k.BankKeeper.MintCoins(ctx, types.ModuleName, mintCoins); err != nil {
		return fmt.Errorf("could not mint new kcoin %v: %w", kCoin, err)
	}

	moduleAddress := k.AccountKeeper.GetModuleAccount(ctx, types.ModuleName).GetAddress()

	tradeCtx := dextypes.TradeContext{
		Context:             ctx,
		CoinSource:          moduleAddress.String(),
		CoinTarget:          moduleAddress.String(),
		TradeAmount:         mintAmount,
		TradeDenomGiving:    kCoin,
		TradeDenomReceiving: constants.BaseCurrency,
		ExcludeFromDiscount: true,
		ProtocolTrade:       true,
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	if _, err = k.DexKeeper.ExecuteSell(tradeCtx); err != nil {
		k.Logger().Info(err.Error())

		if errors.Is(err, dextypes.ErrTradeAmountTooSmall) {
			return nil
		}
		if errors.Is(err, dextypes.ErrNotEnoughLiquidity) {
			return nil
		}

		return fmt.Errorf("could not execute incomplete trade: %w", err)
	}

	if err = tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper); err != nil {
		return fmt.Errorf("could not settle trade balances: %w", err)
	}

	if err = k.burnFunds(ctx, constants.BaseCurrency); err != nil {
		return fmt.Errorf("could not burn funds: %w", err)
	}

	return nil
}

func (k Keeper) adjustForSupplyCap(ctx context.Context, kCoin string, amountToAdd math.Int) math.Int {
	supply := k.BankKeeper.GetSupply(ctx, kCoin).Amount
	maximumSupply := k.DenomKeeper.MaxSupply(ctx, kCoin)

	maximumAddableAmount := maximumSupply.Sub(supply.Add(amountToAdd))
	amountToAdd = math.MinInt(maximumAddableAmount, amountToAdd)

	return amountToAdd
}

func (k Keeper) getUsableAmount(ctx context.Context, denom, module string) math.Int {
	address := k.AccountKeeper.GetModuleAccount(ctx, module).GetAddress()
	return k.BankKeeper.SpendableCoin(ctx, address, denom).Amount
}
