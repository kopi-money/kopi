package keeper

import (
	"context"
	"fmt"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/strategies/types"
)

func (k msgServer) ArbitrageDeposit(ctx context.Context, msg *types.MsgArbitrageDeposit) (*types.Void, error) {
	address, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	amount, err := parseAmount(msg.Amount, false)
	if err != nil {
		return nil, err
	}

	var cAsset *denomtypes.CAsset
	if cAsset, _ = k.DenomKeeper.GetCAssetByBaseName(ctx, msg.Denom); cAsset != nil {
		var cAssetAmount math.Int
		cAssetAmount, err = k.MMKeeper.Deposit(ctx, address, cAsset, amount)
		if err != nil {
			return nil, fmt.Errorf("could not deposit into c asset: %w", err)
		}

		amount = cAssetAmount
	} else {
		cAsset, err = k.DenomKeeper.GetCAsset(ctx, msg.Denom)
		if err != nil {
			return nil, fmt.Errorf("could not find cAsset by name: %v", msg.Denom)
		}
	}

	if amount.IsZero() {
		return nil, fmt.Errorf("cAsset amount is zero")
	}

	arbitrageDenom, err := k.DenomKeeper.GetArbitrageDenomByCAsset(ctx, cAsset.DexDenom)
	if err != nil {
		return nil, err
	}

	coins := sdk.NewCoins(sdk.NewCoin(arbitrageDenom.CAsset, amount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, address, types.PoolArbitrage, coins); err != nil {
		return nil, fmt.Errorf("could not send coins to module: %w", err)
	}

	calculateValue := k.calculateArbitrageTokenValue(ctx, arbitrageDenom)
	newTokens, err := k.calculateNewStrategyAssetAmount(ctx, arbitrageDenom.DexDenom, amount, calculateValue)
	if err != nil {
		return nil, fmt.Errorf("could not calculate new strategy asset amount: %w", err)
	}

	if newTokens.LTE(math.ZeroInt()) {
		return nil, types.ErrZeroMint
	}

	coins = sdk.NewCoins(sdk.NewCoin(arbitrageDenom.DexDenom, newTokens))
	if err = k.BankKeeper.MintCoins(ctx, types.PoolArbitrage, coins); err != nil {
		return nil, err
	}

	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolArbitrage, address, coins); err != nil {
		return nil, fmt.Errorf("could not send coins to module: %w", err)
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("arbitrage_deposit",
			sdk.Attribute{Key: "address", Value: msg.Creator},
			sdk.Attribute{Key: "amount", Value: newTokens.String()},
			sdk.Attribute{Key: "denom", Value: arbitrageDenom.DexDenom},
		),
	)

	return &types.Void{}, nil
}

func (k msgServer) ArbitrageRedeem(ctx context.Context, msg *types.MsgArbitrageRedeem) (*types.Void, error) {
	address, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	amount, err := parseAmount(msg.Amount, false)
	if err != nil {
		return nil, err
	}

	arbitrageDenom, err := k.DenomKeeper.GetArbitrageDenomByName(ctx, msg.Denom)
	if err != nil {
		return nil, err
	}

	spendableCoins := k.BankKeeper.SpendableCoins(ctx, address).AmountOf(arbitrageDenom.DexDenom)
	if spendableCoins.LT(amount) {
		err = fmt.Errorf("requested amount (%v%v) is smaller than available amount (%v%v)",
			amount.String(), arbitrageDenom.DexDenom, spendableCoins.String(), arbitrageDenom.DexDenom)
		return nil, err
	}

	moduleAccount := k.AccountKeeper.GetModuleAccount(ctx, types.PoolArbitrage)
	available := k.BankKeeper.SpendableCoin(ctx, moduleAccount.GetAddress(), arbitrageDenom.CAsset).Amount
	if available.IsZero() {
		return nil, types.ErrEmptyVault
	}

	calculateValue := k.calculateArbitrageTokenValue(ctx, arbitrageDenom)
	payoutAmountGross, burnAmount, err := k.calculateRedemptionAmount(ctx, arbitrageDenom, amount, available, calculateValue, msg.AllowIncomplete)
	if err != nil {
		return nil, fmt.Errorf("could not calculate redemption amount: %w", err)
	}

	coins := sdk.NewCoins(sdk.NewCoin(arbitrageDenom.DexDenom, burnAmount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, address, types.PoolArbitrage, coins); err != nil {
		return nil, fmt.Errorf("could not send aasset coins to module: %w", err)
	}

	payoutAmountNet, err := k.handleRedemptionFee(ctx, arbitrageDenom, payoutAmountGross)
	if err != nil {
		return nil, fmt.Errorf("could not handle redemption fee: %w", err)
	}

	coins = sdk.NewCoins(sdk.NewCoin(arbitrageDenom.DexDenom, burnAmount))
	if err = k.BankKeeper.BurnCoins(ctx, types.PoolArbitrage, coins); err != nil {
		return nil, fmt.Errorf("could not burn aasset coins: %w", err)
	}

	coins = sdk.NewCoins(sdk.NewCoin(arbitrageDenom.CAsset, payoutAmountNet))
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolArbitrage, address, coins); err != nil {
		return nil, fmt.Errorf("could not send casset coins to user: %w", err)
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("arbitrage_redemption",
			sdk.Attribute{Key: "address", Value: msg.Creator},
			sdk.Attribute{Key: "amount", Value: msg.Amount},
			sdk.Attribute{Key: "denom", Value: arbitrageDenom.DexDenom},
		),
	)

	return &types.Void{}, nil
}

func (k Keeper) calculateArbitrageTokenValue(ctx context.Context, arbitrageDenom *denomtypes.ArbitrageDenom) CalculateValue {
	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolArbitrage)

	return CalculateValue{
		func() (math.LegacyDec, error) {
			supply := k.BankKeeper.SpendableCoin(ctx, acc.GetAddress(), arbitrageDenom.CAsset)
			return supply.Amount.ToLegacyDec(), nil
		},
		func() (math.LegacyDec, error) {
			amountKCoin := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress()).AmountOf(arbitrageDenom.KCoin)
			valueKCoin, err := k.DexKeeper.GetValueIn(ctx, arbitrageDenom.KCoin, arbitrageDenom.CAsset, amountKCoin.ToLegacyDec())
			if err != nil {
				return math.LegacyDec{}, fmt.Errorf("could not convert kcoin value to casset value: %w", err)
			}

			return valueKCoin, nil
		},
	}
}

func parseAmount(amountStr string, canBeZero bool) (math.Int, error) {
	amountStr = strings.ReplaceAll(amountStr, ",", "")
	amount, ok := math.NewIntFromString(amountStr)
	if !ok {
		return math.Int{}, types.ErrInvalidAmountFormat
	}

	if amount.LT(math.ZeroInt()) {
		return math.Int{}, types.ErrNegativeAmount
	}

	if !canBeZero && amount.IsZero() {
		return math.Int{}, types.ErrZeroAmount
	}

	return amount, nil
}

func (k Keeper) checkSpendableCoins(ctx context.Context, address sdk.AccAddress, denom string, amount math.Int) error {
	spendableCoins := k.BankKeeper.SpendableCoins(ctx, address).AmountOf(denom)
	if spendableCoins.IsNil() || amount.GT(spendableCoins) {
		return types.ErrNotEnoughFunds
	}

	return nil
}

func (k Keeper) handleRedemptionFee(ctx context.Context, arbitrageDenom *denomtypes.ArbitrageDenom, payoutAmountGross math.Int) (math.Int, error) {
	if payoutAmountGross.LTE(math.ZeroInt()) {
		return math.ZeroInt(), nil
	}

	redemptionFee := arbitrageDenom.RedemptionFee.Mul(payoutAmountGross.ToLegacyDec()).TruncateInt()
	payoutAmountNet := payoutAmountGross.Sub(redemptionFee)

	msg := fmt.Sprintf("Gross: %v, Net: %v, Fee: %v", payoutAmountGross.String(), payoutAmountNet.String(), redemptionFee.String())
	k.Logger().Info(msg)

	redemptionFeeProtocolShare := arbitrageDenom.RedemptionFeeReserveShare.Mul(redemptionFee.ToLegacyDec()).TruncateInt()
	coins := sdk.NewCoins(sdk.NewCoin(arbitrageDenom.CAsset, redemptionFeeProtocolShare))
	if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolArbitrage, dextypes.PoolReserve, coins); err != nil {
		return math.ZeroInt(), err
	}

	return payoutAmountNet, nil
}
