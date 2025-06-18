package keeper

import (
	"context"
	"errors"
	"fmt"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/kopi-money/kopi/trading"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
			return fmt.Errorf("burn denom %v: %w", kCoin, err)
		}
	}

	return nil
}

func (k Keeper) CheckBurn(ctx context.Context, kCoin string, maxBurnAmount math.Int) error {
	parity, referenceDenom, err := k.DenomKeeper.CalculateParity(ctx, kCoin)
	if err != nil {
		return fmt.Errorf("calculate parity: %w", err)
	}

	// parity can be nil at initialization of the chain when not all currencies have liquidity. It is an edge case.
	if parity == nil {
		return nil
	}

	if parity.GT(k.burnThreshold(ctx)) {
		return nil
	}

	mintAmountBase, err := k.DenomKeeper.GetValueInBase(ctx, referenceDenom, maxBurnAmount.ToLegacyDec())
	if err != nil {
		return fmt.Errorf("convert to mintAmountBase: %w", err)
	}

	mintAmountBase = k.adjustToParity(ctx, *parity, mintAmountBase)

	// New coins of the base currency are minted, used to buy the kCoin and burn
	if err = k.mintTradeBurn(ctx, kCoin, mintAmountBase.TruncateInt()); err != nil {
		return fmt.Errorf("mintTradeBurn: %w", err)
	}

	return nil
}

// This function mints new XKP, buys the kCoin and then burns the tokens it has bought.
func (k Keeper) mintTradeBurn(ctx context.Context, kCoin string, mintAmountBase math.Int) error {
	mintCoins := sdk.NewCoins(sdk.NewCoin(constants.BaseCurrency, mintAmountBase))
	if err := k.BankKeeper.MintCoins(ctx, types.ModuleName, mintCoins); err != nil {
		return fmt.Errorf("mint new XKP: %w", err)
	}

	address := k.AccountKeeper.GetModuleAccount(ctx, types.ModuleName).GetAddress()

	tradeCtx := dextypes.TradeContext{
		Context:             ctx,
		TradeAmount:         mintAmountBase,
		CoinSource:          address.String(),
		CoinTarget:          address.String(),
		TradeDenomGiving:    constants.BaseCurrency,
		TradeDenomReceiving: kCoin,
		TradeBalances:       dexkeeper.NewTradeBalances(),
		ExcludeFromDiscount: true,
		ProtocolTrade:       true,
	}

	if _, err := k.DexKeeper.ExecuteSell(tradeCtx); err != nil {
		if errors.Is(err, trading.ErrTradeAmountTooSmall) {
			return nil
		}
		if errors.Is(err, dextypes.ErrNoLiquidityGiving) {
			return nil
		}
		if errors.Is(err, dextypes.ErrNoLiquidityReceiving) {
			return nil
		}

		return fmt.Errorf("execute trade: %w", err)
	}

	if err := tradeCtx.TradeBalances.Settle(ctx, k.BankKeeper); err != nil {
		return fmt.Errorf("settle trade balances: %w", err)
	}

	if err := k.burnFunds(ctx, kCoin); err != nil {
		return fmt.Errorf("burn funds: %w", err)
	}

	return nil
}

func (k Keeper) burnFunds(ctx context.Context, denom string) error {
	burnableAmount := k.getUsableAmount(ctx, denom, types.ModuleName)
	if !burnableAmount.IsPositive() {
		return nil
	}

	if denom == constants.BaseCurrency {
		stakingShare := k.GetParams(ctx).StakingShare
		if stakingShare.IsNil() {
			stakingShare = math.LegacyZeroDec()
		}

		rewards := stakingShare.Mul(burnableAmount.ToLegacyDec()).TruncateInt()
		if rewards.IsPositive() {
			rewardCoins := sdk.NewCoins(sdk.NewCoin(constants.BaseCurrency, rewards))
			if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, authtypes.FeeCollectorName, rewardCoins); err != nil {
				return fmt.Errorf("send coins to distribution: %w", err)
			}

			burnableAmount = burnableAmount.Sub(rewards)
		}
	}

	if burnableAmount.IsPositive() {
		burnCoins := sdk.NewCoins(sdk.NewCoin(denom, burnableAmount))
		if err := k.BankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
			return err
		}
	}

	return nil
}
