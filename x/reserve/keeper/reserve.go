package keeper

import (
	"context"
	"fmt"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/kopi-money/kopi/x/reserve/types"

	denomtypes "github.com/kopi-money/kopi/x/denominations/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	mmtypes "github.com/kopi-money/kopi/x/mm/types"
)

// BeginBlockCheckReserve checks whether the reserve has funds that have not been added to the dex yet and if yes, adds those
// funds to the dex. When the denomination of coins is virtual, it is checked whether the kCoin is above
// parity. When not, those coins are not added to the dex. First, the base currency is handled, after that all other
// currencies.
func (k Keeper) BeginBlockCheckReserve(ctx context.Context) error {
	address := k.AccountKeeper.GetModuleAccount(ctx, dextypes.PoolReserve).GetAddress()
	coins := k.BankKeeper.SpendableCoins(ctx, address)

	if err := k.handleBaseLiquidity(ctx, address, coins.AmountOf(constants.BaseCurrency)); err != nil {
		return fmt.Errorf("could not handle base liquidity: %w", err)
	}

	for _, coin := range coins {
		if coin.Denom == constants.BaseCurrency {
			continue
		}

		// Not adding liquidity for denom that has not (yet) been whitelisted
		if !k.DenomKeeper.IsValidDenom(ctx, coin.Denom) {
			continue
		}

		if err := k.checkReserveForDenom(ctx, address, coin); err != nil {
			return fmt.Errorf("error checking reserve for %v: %w", coin.Denom, err)
		}
	}

	return nil
}

func (k Keeper) handleBaseLiquidity(ctx context.Context, address sdk.AccAddress, baseAmount math.Int) error {
	acc := k.AccountKeeper.GetModuleAccount(ctx, dextypes.PoolLiquidity)
	baseLiquidity := k.BankKeeper.SpendableCoin(ctx, acc.GetAddress(), constants.BaseCurrency).Amount
	minimumLiquidity := math.NewInt(constants.MinimumBaseLiquidity)

	// Make sure the DEX always has at least 1000 XKP for liquidity
	if baseLiquidity.LT(minimumLiquidity) {
		newCoins := sdk.NewCoins(sdk.NewCoin(constants.BaseCurrency, minimumLiquidity))
		if err := k.BankKeeper.MintCoins(ctx, types.ModuleName, newCoins); err != nil {
			return fmt.Errorf("could not mint new coins: %w", err)
		}

		if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, dextypes.PoolReserve, newCoins); err != nil {
			return fmt.Errorf("could not send from module account to reserve: %w", err)
		}

		baseAmount = baseAmount.Add(minimumLiquidity)
	}

	if baseAmount.GT(math.ZeroInt()) {
		baseCoin := sdk.NewCoin(constants.BaseCurrency, baseAmount)
		if err := k.checkReserveForDenom(ctx, address, baseCoin); err != nil {
			return fmt.Errorf("error checking reserve for base currency: %w", err)
		}
	}

	return nil
}

func (k Keeper) checkReserveForDenom(ctx context.Context, address sdk.AccAddress, coin sdk.Coin) error {
	if coin.Amount.IsZero() {
		return nil
	}

	// If the denom is a borrowable denom, part of the reserve is sent to the money market to incentivize deposits
	if cAsset, err := k.DenomKeeper.GetCAssetByBaseName(ctx, coin.Denom); err == nil {
		coin = k.sendToMoneyMarket(ctx, coin, cAsset)
	}

	// If the coins are kCoins, they are burned
	coin, err := k.burnKCoinReserve(ctx, coin)
	if err != nil {
		return fmt.Errorf("burn kcoin reserve: %w", err)
	}

	if coin.Amount.IsPositive() {
		stakerShareFactor := k.getTradeFeeShareStakers(ctx, coin.Denom)
		stakerShareAmount := stakerShareFactor.Mul(coin.Amount.ToLegacyDec()).TruncateInt()
		if stakerShareAmount.IsPositive() {
			rewardCoins := sdk.NewCoins(sdk.NewCoin(coin.Denom, stakerShareAmount))

			if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, dextypes.PoolReserve, authtypes.FeeCollectorName, rewardCoins); err != nil {
				return fmt.Errorf("send coins to distribution: %w", err)
			}

			coin.Amount = coin.Amount.Sub(stakerShareAmount)
		}

		if coin.Amount.IsPositive() {
			if _, err = k.DexKeeper.AddLiquidity(ctx, address, coin.Denom, coin.Amount); err != nil {
				return fmt.Errorf("add liquidity: %w", err)
			}
		}
	}

	return nil
}

func (k Keeper) burnKCoinReserve(ctx context.Context, coin sdk.Coin) (sdk.Coin, error) {
	if !k.DenomKeeper.IsKCoin(ctx, coin.Denom) {
		return coin, nil
	}

	if coin.Amount.LTE(math.ZeroInt()) {
		return coin, nil
	}

	kCoinBurnShare := k.getKCoinBurnShare(ctx)
	burnAmount := coin.Amount.ToLegacyDec().Mul(kCoinBurnShare).TruncateInt()

	burnCoin := sdk.NewCoins(sdk.NewCoin(coin.Denom, burnAmount))
	if err := k.BankKeeper.BurnCoins(ctx, dextypes.PoolReserve, burnCoin); err != nil {
		return sdk.Coin{}, err
	}

	coin.Amount = coin.Amount.Sub(burnAmount)
	return coin, nil
}

func (k Keeper) sendToMoneyMarket(ctx context.Context, coin sdk.Coin, cAsset denomtypes.CAsset) sdk.Coin {
	cAssetValue := k.MMKeeper.CalculateCAssetValue(ctx, cAsset)
	sendAmount := math.LegacyNewDecFromInt(coin.Amount).Mul(cAsset.DexFeeShare)

	// The cAsset's value should be increased by 1% at max
	maxSendAmont := cAssetValue.Mul(math.LegacyNewDecWithPrec(1, 2))
	sendAmount = math.LegacyMinDec(maxSendAmont, sendAmount)

	sendAmountInt := sendAmount.TruncateInt()
	if sendAmountInt.GT(math.ZeroInt()) {
		coins := sdk.NewCoins(sdk.NewCoin(coin.Denom, sendAmountInt))
		_ = k.BankKeeper.SendCoinsFromModuleToModule(ctx, dextypes.PoolReserve, mmtypes.PoolVault, coins)
	}

	coin.Amount = coin.Amount.Sub(sendAmountInt)
	return coin
}
