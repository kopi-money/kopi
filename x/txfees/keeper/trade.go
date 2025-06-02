package keeper

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/cache"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/kopi-money/kopi/constants"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/txfees/types"
)

func (k Keeper) Trade(ctx context.Context) {
	moduleAcc := k.accountKeeper.GetModuleAccount(ctx, types.ModuleName).GetAddress()
	available := k.bankKeeper.SpendableCoins(ctx, moduleAcc)

	for _, feeDenom := range k.denomKeeper.FeeDenoms(ctx) {
		if err := k.tradeFeeDenom(ctx, moduleAcc, available, feeDenom); err != nil {
			k.Logger().Error(fmt.Sprintf("failed to trade fee denom %v: %v", feeDenom.Denom, err.Error()))
		}
	}
}

func (k Keeper) tradeFeeDenom(ctx context.Context, moduleAddress sdk.AccAddress, available sdk.Coins, feeDenom denomtypes.FeeDenom) error {
	if available.AmountOf(feeDenom.Denom).LT(feeDenom.MinimumTradeAmount) {
		return nil
	}

	target := k.accountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
	tradeCtx := dextypes.TradeContext{
		TradeAmount:         feeDenom.MinimumTradeAmount,
		CoinSource:          moduleAddress.String(),
		CoinTarget:          target.String(),
		TradeDenomGiving:    feeDenom.Denom,
		TradeDenomReceiving: constants.BaseCurrency,
		TradeBalances:       dexkeeper.NewTradeBalances(),
		ExcludeFromDiscount: true,
		ProtocolTrade:       true,
	}

	return cache.TransactWithNewMultiStore(ctx, func(innerCtx context.Context) error {
		tradeCtx.Context = innerCtx
		result, err := k.dexKeeper.ExecuteSell(tradeCtx)
		if err != nil {
			return fmt.Errorf("failed to execute sell: %w", err)
		}

		if err = tradeCtx.TradeBalances.Settle(ctx, k.bankKeeper); err != nil {
			return fmt.Errorf("failed to settle trade balances: %w", err)
		}

		sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
			sdk.NewEvent("fee_income_traded",
				sdk.Attribute{Key: "denom", Value: feeDenom.Denom},
				sdk.Attribute{Key: "amount_given", Value: result.AmountGiven().String()},
				sdk.Attribute{Key: "amount_received", Value: result.AmountReceived().String()},
			),
		)

		return err
	})
}
