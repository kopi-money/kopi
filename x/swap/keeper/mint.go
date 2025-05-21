package keeper

import (
	"context"
	"errors"
	"fmt"
	"github.com/kopi-money/kopi/trading"

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
			return fmt.Errorf("mint denom: %w", err)
		}
	}

	return nil
}

// CheckMint checks the parity of a given kCoin. If it is above 1, new coins are minted and sold in favor of
// the base currency.
func (k Keeper) CheckMint(ctx context.Context, kCoin string, maxMintAmount math.Int) error {
	parity, _, err := k.DenomKeeper.CalculateParity(ctx, kCoin)
	if err != nil {
		return fmt.Errorf("calculate parity: %w", err)
	}

	// parity can be nil at initialization of the chain when not all currencies have liquidity. It is an edge case.
	if parity == nil {
		return nil
	}

	if parity.LTE(k.mintThreshold(ctx)) {
		return nil
	}

	maxMintAmount = k.adjustToParity(ctx, *parity, maxMintAmount.ToLegacyDec()).TruncateInt()
	mintAmount := k.adjustForSupplyCap(ctx, kCoin, maxMintAmount)
	if mintAmount.LTE(math.OneInt()) {
		return nil
	}

	mintCoins := sdk.NewCoins(sdk.NewCoin(kCoin, mintAmount))
	if err = k.BankKeeper.MintCoins(ctx, types.ModuleName, mintCoins); err != nil {
		return fmt.Errorf("mint new kcoin %v: %w", kCoin, err)
	}

	moduleAddress := k.AccountKeeper.GetModuleAccount(ctx, types.ModuleName).GetAddress()

	tradeCtx := dextypes.TradeContext{
		TradeAmount:         mintAmount,
		Context:             ctx,
		CoinSource:          moduleAddress.String(),
		CoinTarget:          moduleAddress.String(),
		TradeDenomGiving:    kCoin,
		TradeDenomReceiving: constants.BaseCurrency,
		TradeBalances:       dexkeeper.NewTradeBalances(),
		ExcludeFromDiscount: true,
		ProtocolTrade:       true,
	}

	if _, err = k.DexKeeper.ExecuteSell(tradeCtx); err != nil {
		k.Logger().Info(err.Error())

		if isError(err, []error{
			trading.ErrTradeAmountTooSmall,
			dextypes.ErrNoLiquidityGiving,
			dextypes.ErrNoLiquidityReceiving,
			dextypes.ErrNotEnoughFunds,
		}) {
			return nil
		}

		return fmt.Errorf("execute incomplete trade: %w", err)
	}

	if err = tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper); err != nil {
		return fmt.Errorf("settle trade balances: %w", err)
	}

	if err = k.burnFunds(ctx, constants.BaseCurrency); err != nil {
		return fmt.Errorf("burn funds: %w", err)
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

func isError(err error, targets []error) bool {
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}

	return false
}

// adjustToParity adjusts the mint/burn amount according to the deviation from parity. For example, if the parity is
// 110%, the mint amount will be 10% higher than the set value. The parity factor additionally increases that value.
func (k Keeper) adjustToParity(ctx context.Context, parity math.LegacyDec, mintBurnAmount math.LegacyDec) math.LegacyDec {
	if parity.LT(math.LegacyOneDec()) {
		parity = math.LegacyOneDec().Quo(parity)
	}

	parity = parity.Sub(math.LegacyOneDec())
	parity = parity.Mul(k.parityFactor(ctx))
	parity = math.LegacyMinDec(parity, math.LegacyOneDec())
	parity = parity.Add(math.LegacyOneDec())

	return mintBurnAmount.Mul(parity)
}
