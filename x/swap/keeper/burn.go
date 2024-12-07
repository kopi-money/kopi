package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/kopi-money/kopi/constants"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/swap/types"
)

// Burn is called at the end of each block to check whether the prices of the kCoins are lower than their
// "real" counterparts. If yes, funds for the base currency are minted, the kCoin is bought and received
// funds are burned such as to lower the supply of the kCoin and slightly increase its price. The amount
// that is minted is limited depending on the currency to not mint too much per block.
func (k Keeper) Burn(ctx context.Context) error {
	for _, kCoin := range k.DenomKeeper.KCoins(ctx) {
		maxBurnAmount := k.DenomKeeper.MaxBurnAmount(ctx, kCoin)
		if err := k.CheckBurn(ctx, kCoin, maxBurnAmount); err != nil {
			return fmt.Errorf("could not burn denom %v: %w", kCoin, err)
		}
	}

	return nil
}

func (k Keeper) CheckBurn(ctx context.Context, kCoin string, maxBurnAmount math.Int) error {
	parity, referenceDenom, err := k.DexKeeper.CalculateParity(ctx, kCoin)
	if err != nil {
		return fmt.Errorf("could not calculate parity: %w", err)
	}

	// parity can be nil at initialization of the chain when not all currencies have liquidity. It is an edge case.
	if parity == nil {
		return nil
	}

	if parity.GT(k.burnThreshold(ctx)) {
		return nil
	}

	maxBurnAmountBase, err := k.DexKeeper.GetValueInBase(ctx, referenceDenom, maxBurnAmount.ToLegacyDec())
	if err != nil {
		return fmt.Errorf("could not convert to maxBurnAmountBase: %w", err)
	}

	referenceRatio, _ := k.DenomKeeper.GetRatio(ctx, referenceDenom)
	mintAmountBase := k.calcBaseMintAmount(ctx, referenceRatio.Ratio, kCoin)
	mintAmountBase = math.MinInt(mintAmountBase, maxBurnAmountBase.TruncateInt())
	if mintAmountBase.LTE(math.ZeroInt()) {
		return nil
	}

	mintCoins := sdk.NewCoins(sdk.NewCoin(kCoin, mintAmountBase))
	if err = k.BankKeeper.MintCoins(ctx, types.ModuleName, mintCoins); err != nil {
		return fmt.Errorf("could not mint coins: %w", err)
	}

	// New coins of the base currency are minted, used to buy the kCoin and burn
	if err = k.mintTradeBurn(ctx, kCoin, mintAmountBase); err != nil {
		return fmt.Errorf("could not mintTradeBurn: %w", err)
	}

	return nil
}

func (k Keeper) calcBaseMintAmount(ctx context.Context, referenceRatio math.LegacyDec, kCoin string) math.Int {
	liqBase := k.DexKeeper.GetFullLiquidityBase(ctx, kCoin)
	liqKCoin := k.DexKeeper.GetFullLiquidityOther(ctx, kCoin)
	constantProductRoot, _ := liqBase.Mul(liqKCoin).Quo(referenceRatio).ApproxSqrt()
	mintAmount := constantProductRoot.Sub(liqBase)

	mintAmount = mintAmount.Mul(k.BlockspeedKeeper.GetBlocksPerSecond(ctx))

	return mintAmount.TruncateInt()
}

// This function mints new XKP, buys the kCoin and then burns the tokens it has bought.
func (k Keeper) mintTradeBurn(ctx context.Context, kCoin string, mintAmountBase math.Int) error {
	mintCoins := sdk.NewCoins(sdk.NewCoin(constants.BaseCurrency, mintAmountBase))
	if err := k.BankKeeper.MintCoins(ctx, types.ModuleName, mintCoins); err != nil {
		return fmt.Errorf("could not mint new XKP: %w", err)
	}

	address := k.AccountKeeper.GetModuleAccount(ctx, types.ModuleName).GetAddress()

	tradeCtx := dextypes.TradeContext{
		Context:             ctx,
		TradeAmount:         mintAmountBase,
		CoinSource:          address.String(),
		CoinTarget:          address.String(),
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: kCoin,
		ExcludeFromDiscount: true,
		ProtocolTrade:       true,
		TradeBalances:       dexkeeper.NewTradeBalances(),
	}

	if _, err := k.DexKeeper.ExecuteSell(tradeCtx); err != nil {
		if errors.Is(err, dextypes.ErrTradeAmountTooSmall) {
			return nil
		}
		if errors.Is(err, dextypes.ErrNotEnoughLiquidity) {
			return nil
		}

		return fmt.Errorf("could not execute trade: %w", err)
	}

	if err := tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper); err != nil {
		return fmt.Errorf("could not settle trade balances: %w", err)
	}

	if err := k.burnFunds(ctx, kCoin); err != nil {
		return fmt.Errorf("could not burn funds: %w", err)
	}

	return nil
}

func (k Keeper) burnFunds(ctx context.Context, denom string) error {
	burnableAmount := k.getUsableAmount(ctx, denom, types.ModuleName)
	if burnableAmount.LTE(math.ZeroInt()) {
		return nil
	}

	if denom == constants.BaseCurrency {
		stakingShare := k.GetParams(ctx).StakingShare
		if stakingShare.IsNil() {
			stakingShare = math.LegacyZeroDec()
		}

		rewards := stakingShare.Mul(burnableAmount.ToLegacyDec()).TruncateInt()
		if rewards.GT(math.ZeroInt()) {
			rewardCoins := sdk.NewCoins(sdk.NewCoin(constants.BaseCurrency, rewards))
			if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, distributiontypes.ModuleName, rewardCoins); err != nil {
				return fmt.Errorf("could not send coins to distribution: %w", err)
			}

			burnableAmount = burnableAmount.Sub(rewards)
		}
	}

	burnCoins := sdk.NewCoins(sdk.NewCoin(denom, burnableAmount))
	if err := k.BankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
		return err
	}

	return nil
}
