package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/kopi-money/kopi/utils"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/swap/types"
	"github.com/pkg/errors"
)

// Burn is called at the end of each block to check whether the prices of the kCoins are lower than their
// "real" counterparts. If yes, funds for the base currency are minted, the kCoin is bought and received
// funds are burned such as to lower the supply of the kCoin and slightly increase its price. The amount
// that is minted is limited depending on the currency to not mint too much per block.
func (k Keeper) Burn(ctx context.Context, eventManager sdk.EventManagerI) error {
	for _, kCoin := range k.DenomKeeper.KCoins(ctx) {
		maxBurnAmount := k.DenomKeeper.MaxBurnAmount(ctx, kCoin)
		if err := k.CheckBurn(ctx, eventManager, kCoin, maxBurnAmount); err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not burn denom %v", kCoin))
		}
	}

	return nil
}

func (k Keeper) CheckBurn(ctx context.Context, eventManager sdk.EventManagerI, kCoin string, maxBurnAmount math.Int) error {
	parity, referenceDenom, err := k.DexKeeper.CalculateParity(ctx, kCoin)
	if err != nil {
		return errors.Wrap(err, "could not calculate parity")
	}

	// parity can be nil at initialization of the chain when not all currencies have liquidity. It is an edge case.
	if parity == nil || parity.GT(math.LegacyOneDec()) {
		return nil
	}

	referenceRatio, _ := k.DexKeeper.GetRatio(ctx, referenceDenom)
	if referenceRatio.Ratio == nil || referenceRatio.Ratio.GT(math.LegacyOneDec()) {
		return nil
	}

	mintAmountBase := k.calcBaseMintAmount(ctx, referenceDenom, kCoin, maxBurnAmount)
	if mintAmountBase.LTE(math.ZeroInt()) {
		return nil
	}

	// Liquidity of the kCoin is removed if present
	liq := k.DexKeeper.GetLiquidityByAddress(ctx, kCoin, dextypes.PoolReserve)
	if liq.GT(math.ZeroInt()) {
		if err = k.DexKeeper.RemoveAllLiquidityForModule(ctx, eventManager, kCoin, dextypes.PoolReserve); err != nil {
			return errors.Wrap(err, "could not remove all liquidity for module")
		}
	}

	// New coins of the base currency are minted, used to buy the kCoin and burn
	if err = k.mintTradeBurn(ctx, eventManager, kCoin, mintAmountBase); err != nil {
		return errors.Wrap(err, "could not mintTradeBurn")
	}

	return nil
}

func (k Keeper) calcBaseMintAmount(ctx context.Context, referenceDenom, kCoin string, maxBurnAmount math.Int) math.Int {
	liqReference := k.DexKeeper.GetFullLiquidityOther(ctx, referenceDenom)
	liqVirtual := k.DexKeeper.GetFullLiquidityOther(ctx, kCoin)

	amountDiff := liqVirtual.Sub(liqReference)
	if amountDiff.LTE(math.LegacyZeroDec()) {
		return math.ZeroInt()
	}

	amountReference := math.MinInt(amountDiff.RoundInt(), maxBurnAmount)
	mintAmount, _, _, _ := k.DexKeeper.SimulateTradeForReserve(ctx, referenceDenom, utils.BaseCurrency, amountReference)

	return mintAmount
}

// This function mints new XKP, buys the kCoin and then burns the tokens it has bought.
func (k Keeper) mintTradeBurn(ctx context.Context, eventManager sdk.EventManagerI, kCoin string, mintAmountBase math.Int) error {
	k.Logger().Info(fmt.Sprintf("MTB %v, minting %v ukopi", kCoin, mintAmountBase))

	mintCoins := sdk.NewCoins(sdk.NewCoin(utils.BaseCurrency, mintAmountBase))
	if err := k.BankKeeper.MintCoins(ctx, types.ModuleName, mintCoins); err != nil {
		return errors.Wrap(err, "could not mint new XKP")
	}

	address := k.AccountKeeper.GetModuleAccount(ctx, types.ModuleName).GetAddress()

	options := dextypes.TradeOptions{
		GivenAmount:         mintAmountBase,
		CoinSource:          address,
		CoinTarget:          address,
		MaxPrice:            nil,
		TradeDenomStart:     utils.BaseCurrency,
		TradeDenomEnd:       kCoin,
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

		return errors.Wrap(err, "could not execute trade")
	}

	eventManager.EmitEvent(
		sdk.NewEvent("arbitrage_trade",
			sdk.Attribute{Key: "denom_from", Value: options.TradeDenomStart},
			sdk.Attribute{Key: "denom_to", Value: options.TradeDenomEnd},
			sdk.Attribute{Key: "amount_used", Value: amountUsed.String()},
			sdk.Attribute{Key: "amount_received", Value: amountReceived.String()},
		))

	burnedAmount, err := k.burnFunds(ctx, eventManager, kCoin)
	if err != nil {
		return errors.Wrap(err, "could not burn funds")
	}

	eventManager.EmitEvent(
		sdk.NewEvent("swap_coins_minted",
			sdk.Attribute{Key: "denom", Value: utils.BaseCurrency},
			sdk.Attribute{Key: "amount", Value: mintAmountBase.String()},
		),
	)

	eventManager.EmitEvent(
		sdk.NewEvent("swap_coins_burned",
			sdk.Attribute{Key: "denom", Value: kCoin},
			sdk.Attribute{Key: "amount", Value: burnedAmount.String()},
		),
	)

	return nil
}

func (k Keeper) burnFunds(ctx context.Context, eventManager sdk.EventManagerI, denom string) (math.Int, error) {
	burnableAmount := k.getUsableAmount(ctx, denom, types.ModuleName)

	if denom == utils.BaseCurrency {
		rewards := k.GetParams(ctx).StakingShare.Mul(math.LegacyNewDecFromInt(burnableAmount))
		if rewards.GT(math.LegacyZeroDec()) {
			rewardCoins := sdk.NewCoins(sdk.NewCoin(utils.BaseCurrency, rewards.RoundInt()))
			if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, distributiontypes.ModuleName, rewardCoins); err != nil {
				return math.Int{}, errors.Wrap(err, "could not send coins to distribution")
			}

			eventManager.EmitEvent(
				sdk.NewEvent("swap_reward",
					sdk.Attribute{Key: "denom", Value: utils.BaseCurrency},
					sdk.Attribute{Key: "amount", Value: rewards.RoundInt().String()},
				),
			)

			burnableAmount = burnableAmount.Sub(rewards.RoundInt())
		}
	}

	burnCoins := sdk.NewCoins(sdk.NewCoin(denom, burnableAmount))
	if err := k.BankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
		return burnableAmount, err
	}

	eventManager.EmitEvent(
		sdk.NewEvent("swap_coins_burned",
			sdk.Attribute{Key: "denom", Value: denom},
			sdk.Attribute{Key: "amount", Value: burnableAmount.String()},
		),
	)

	return burnableAmount, nil
}
