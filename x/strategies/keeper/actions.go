package keeper

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/strategies/types"
)

func isNoAmountAction(actionType int64) bool {
	for _, noAmountActionType := range types.NoAmountActions {
		if noAmountActionType == actionType {
			return true
		}
	}

	return false
}

func (k Keeper) CheckActions(ctx context.Context, address string, actions []types.Action) error {
	for actionIndex, action := range actions {
		if err := k.CheckAction(ctx, address, action); err != nil {
			return fmt.Errorf("invalid action[%d]: %w", actionIndex, err)
		}
	}

	return nil
}

func (k Keeper) CheckAction(ctx context.Context, address string, action types.Action) error {
	if !isNoAmountAction(action.ActionType) {
		if !types.RegexPercentage.MatchString(action.Amount) {
			integer, ok := math.NewIntFromString(action.Amount)

			if !ok {
				return fmt.Errorf("given amount either has to be valid integer or percentage, was: %v", action.Amount)
			}

			if integer.IsZero() {
				return types.ErrZeroAmount
			}
		}
	}

	switch action.ActionType {
	case types.ActionWithdrawRewards:
		if action.String1 != "" {
			return fmt.Errorf("string1 has to be empty")
		}

		if action.String2 != "" {
			return fmt.Errorf("string2 has to be empty")
		}

		if action.Amount != "" {
			return fmt.Errorf("amount has to be empty")
		}

	case types.ActionWithdrawRewardsAndStake:
		if action.String1 == "" {
			return fmt.Errorf("string1 is empty")
		}

		if !isValidStakingStrategy(action.String1) {
			return fmt.Errorf("invalid staking strategy: '%v'", action.String1)
		}

		if action.String2 != "" {
			return fmt.Errorf("string2 has to be empty")
		}

		if action.Amount != "" {
			return fmt.Errorf("amount has to be empty")
		}

	case types.ActionStake:
		if err := checkForStakingStrategy(action.String1, action.String2); err != nil {
			if err = checkForStakingStrategy(action.String2, action.String1); err != nil {
				return fmt.Errorf("either string1 or string2 has  to be valid staking strategy")
			}
		}

	case types.ActionWithdrawAutomationFunds, types.ActionDepositAutomationFunds:
		if action.String1 != "" {
			return fmt.Errorf("string1 has to be empty")
		}

		if action.String2 != "" {
			return fmt.Errorf("string2 has to be empty")
		}

	case types.ActionBuy, types.ActionSell:
		if !k.DenomKeeper.IsValidDenom(ctx, action.String1) {
			return fmt.Errorf("invalid part2: %v", action.String1)
		}

		if !k.DenomKeeper.IsValidDenom(ctx, action.String2) {
			return fmt.Errorf("invalid part3: %v", action.String2)
		}

		if action.String1 == action.String2 {
			return fmt.Errorf("same denom twice")
		}

		amount, ok := math.NewIntFromString(action.Amount)
		if !ok {
			return fmt.Errorf("given amount was no valid integer, was: %v", action.Amount)
		}

		if amount.LT(math.ZeroInt()) {
			return fmt.Errorf("amount cannot be negative")
		}

		if action.MinimumTradeAmount != "" {
			var minimumTradeAmount math.Int
			minimumTradeAmount, ok = math.NewIntFromString(action.MinimumTradeAmount)
			if !ok {
				return fmt.Errorf("invalid minimum trade amount")
			}

			if minimumTradeAmount.GT(amount) {
				return fmt.Errorf("minimum trade amount must not be larger than trade amount")
			}
		}

	case types.ActionDeposit:
		if action.String2 != "" {
			return fmt.Errorf("string1 has to be empty")
		}

		if !k.DenomKeeper.IsBorrowableDenom(ctx, action.String1) {
			return fmt.Errorf("invalid denom: %v", action.String1)
		}

	case types.ActionRedeem:
		if action.String2 != "" {
			return fmt.Errorf("string1 has to be empty")
		}

		if !k.DenomKeeper.IsCAsset(ctx, action.String1) {
			return fmt.Errorf("invalid denom: %v", action.String1)
		}

	case types.ActionCollateralAdd, types.ActionCollateralWithdraw:
		if action.String2 != "" {
			return fmt.Errorf("string1 has to be empty")
		}

		if !k.DenomKeeper.IsCollateralDenom(ctx, action.String1) {
			return fmt.Errorf("invalid denom: %v", action.String1)
		}

	case types.ActionLoanBorrow, types.ActionLoanRepay:
		if action.String2 != "" {
			return fmt.Errorf("string1 has to be empty")
		}

		if !k.DenomKeeper.IsBorrowableDenom(ctx, action.String1) {
			return fmt.Errorf("invalid denom: %v", action.String1)
		}

	case types.ActionLiquidityAdd, types.ActionLiquidityWithdraw:
		if action.String2 != "" {
			return fmt.Errorf("string1 has to be empty")
		}

		if !k.DenomKeeper.IsValidDenom(ctx, action.String1) {
			return fmt.Errorf("invalid denom: %v", action.String1)
		}

	case types.ActionSendCoins:
		if !k.DenomKeeper.IsValidDenom(ctx, action.String1) {
			return fmt.Errorf("invalid denom: %v", action.String1)
		}

		if _, err := sdk.AccAddressFromBech32(action.String2); err != nil {
			return fmt.Errorf("given address was no valid address: %v", action.String2)
		}

		if action.String2 == address {
			return fmt.Errorf("sender and receiver must not be equal")
		}

	default:
		return fmt.Errorf("invalid action type: %v", action.ActionType)
	}

	return nil
}

func (k Keeper) ExecuteAction(ctx context.Context, address sdk.AccAddress, action types.Action, automationIndex, automationExecutionIndex, actionIndex int) error {
	tradeBalances := dexkeeper.NewTradeBalances()

	amount1, amount2, volume, string1, string2, err := k.executeAction(ctx, address, action, tradeBalances, automationIndex, automationExecutionIndex, actionIndex)
	if err != nil && !errorIsOf(err, types.ValidErrors) {
		return fmt.Errorf("could not execute action: %w", err)
	}

	if err == nil {
		if err = tradeBalances.Settle(ctx, k.BankKeeper); err != nil {
			return fmt.Errorf("could not settle balances: %w", err)
		}
	}

	event := sdk.NewEvent(
		"automation_action_executed",
		sdk.Attribute{Key: "automation_index", Value: strconv.Itoa(automationIndex)},
		sdk.Attribute{Key: "action_index", Value: strconv.Itoa(actionIndex)},
		sdk.Attribute{Key: "action_type", Value: strconv.Itoa(int(action.ActionType))},
		sdk.Attribute{Key: "cost", Value: strconv.Itoa(int(k.GetParams(ctx).AutomationFeeAction))},
	)

	if err != nil && errorIsOf(err, types.ValidErrors) {
		event = event.AppendAttributes(sdk.Attribute{
			Key:   "error",
			Value: err.Error(),
		})
	}

	if err == nil {
		if !amount1.IsNil() {
			event = event.AppendAttributes(sdk.Attribute{
				Key:   "amount1",
				Value: strconv.Itoa(int(amount1.Int64())),
			})
		}

		if !amount2.IsNil() {
			event = event.AppendAttributes(sdk.Attribute{
				Key:   "amount2",
				Value: strconv.Itoa(int(amount2.Int64())),
			})
		}

		if string1 != "" {
			event = event.AppendAttributes(sdk.Attribute{
				Key:   "string1",
				Value: string1,
			})
		}

		if string2 != "" {
			event = event.AppendAttributes(sdk.Attribute{
				Key:   "string2",
				Value: string2,
			})
		}

		if !volume.IsNil() {
			event = event.AppendAttributes(sdk.Attribute{
				Key:   "volume",
				Value: volume.TruncateInt().String(),
			})
		}
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(event)
	return err
}

func (k Keeper) executeAction(
	ctx context.Context,
	address sdk.AccAddress,
	action types.Action,
	tradeBalances dextypes.TradeBalances,
	automationIndex, automationExecutionIndex, actionIndex int) (amount1, amount2 math.Int, volume math.LegacyDec, string1, string2 string, err error) {

	string1 = action.String1
	string2 = action.String2

	if action.ActionType != types.ActionSell && action.ActionType != types.ActionBuy {
		if action.MinimumTradeAmount != "" {
			err = fmt.Errorf("minimum_trade_amount has to be empty if action is no trade action, was: '%v", action.MinimumTradeAmount)
			return
		}
	}

	switch action.ActionType {
	case types.ActionSell, types.ActionBuy:
		var denomGiving, denomReceiving string
		if action.ActionType == types.ActionSell {
			denomGiving = action.String1
			denomReceiving = action.String2
		} else {
			denomGiving = action.String2
			denomReceiving = action.String1
		}

		amount1, err = k.getAmountWallet(ctx, address, action.String1, action.Amount)
		if err != nil {
			return
		}

		var minimumTradeAmount *math.Int
		minimumTradeAmount, err = stringToInt(action.MinimumTradeAmount)
		if err != nil {
			return
		}

		tradeCtx := dextypes.TradeContext{
			Context:             ctx,
			TradeAmount:         amount1,
			TradeDenomGiving:    denomGiving,
			TradeDenomReceiving: denomReceiving,
			MinimumTradeAmount:  minimumTradeAmount,
			CoinSource:          address.String(),
			CoinTarget:          address.String(),
			DiscountAddress:     address.String(),
			TradeBalances:       tradeBalances,
		}

		var tradeResult dextypes.TradeResult
		if action.ActionType == types.ActionSell {
			tradeResult, err = k.DexKeeper.ExecuteSell(tradeCtx)
		} else {
			tradeResult, err = k.DexKeeper.ExecuteBuy(tradeCtx)
		}

		if err != nil {
			return
		}

		if action.ActionType == types.ActionSell {
			amount1 = tradeResult.AmountGiven
			amount2 = tradeResult.AmountReceived
		} else {
			amount2 = tradeResult.AmountGiven
			amount1 = tradeResult.AmountReceived
		}

		volume, err = k.DexKeeper.GetValueInUSD(ctx, denomReceiving, amount2.ToLegacyDec())

	case types.ActionDeposit:
		amount1, err = k.getAmountWallet(ctx, address, action.String1, action.Amount)
		if err != nil {
			return
		}

		var cAsset denomtypes.CAsset
		cAsset, err = k.DenomKeeper.GetCAsset(ctx, action.String1)
		if err != nil {
			err = fmt.Errorf("could not get c asset: %w", err)
			return
		}

		amount2, err = k.MMKeeper.Deposit(ctx, address, cAsset, amount1)
		string2 = cAsset.DexDenom

	case types.ActionRedeem:
		amount1, err = k.getAmountWallet(ctx, address, action.String1, action.Amount)
		if err != nil {
			return
		}

		var cAsset denomtypes.CAsset
		cAsset, err = k.DenomKeeper.GetCAsset(ctx, action.String1)
		if err != nil {
			err = fmt.Errorf("could not get c asset: %w", err)
			return
		}

		fee := k.MMKeeper.GetMinimumRedemptionFee(ctx)
		err = k.MMKeeper.CreateRedemptionRequest(ctx, address, cAsset, amount1, fee)

	case types.ActionLoanBorrow:
		amount1, err = k.getAmountBorrowable(ctx, address, action.String1, action.Amount)
		if err != nil {
			err = fmt.Errorf("could not get borrowable amount: %w", err)
			return
		}

		amount1, amount2, err = k.MMKeeper.Borrow(ctx, address, action.String1, amount1)

	case types.ActionLoanRepay:
		amount1, err = k.getAmountWallet(ctx, address, action.String1, action.Amount)
		if err != nil {
			return
		}

		err = k.MMKeeper.Repay(ctx, action.String1, address.String(), amount1)
		amount2 = k.MMKeeper.GetLoanValue(ctx, action.String1, address.String()).TruncateInt()

	case types.ActionCollateralAdd:
		amount1, err = k.getAmountWallet(ctx, address, action.String1, action.Amount)
		if err != nil {
			return
		}

		amount2, err = k.MMKeeper.AddCollateral(ctx, address, action.String1, amount1)

	case types.ActionCollateralWithdraw:
		amount1, err = k.getAmountWithdrawableCollateral(ctx, address, action.String1, action.Amount)
		if err != nil {
			err = fmt.Errorf("could not get withdrawable collateral amount: %w", err)
			return
		}

		amount2, err = k.MMKeeper.WithdrawCollateral(ctx, address, action.String1, amount1)

	case types.ActionLiquidityAdd:
		amount1, err = k.getAmountWallet(ctx, address, action.String1, action.Amount)
		if err != nil {
			return
		}

		amount2, err = k.DexKeeper.AddLiquidity(ctx, address, action.String1, amount1)

	case types.ActionLiquidityWithdraw:
		if !k.DenomKeeper.IsValidDenom(ctx, action.String1) {
			err = denomtypes.ErrInvalidDexAsset
		}

		if err == nil {
			amount1 = k.getAmountLiquidity(ctx, address, action.String1, action.Amount)
			err = k.DexKeeper.RemoveLiquidityForAddress(ctx, address, action.String1, amount1)

			if err == nil {
				amount2 = k.DexKeeper.GetLiquidityByAddress(ctx, action.String1, address.String())
			}
		}

	case types.ActionSendCoins:
		receiver, _ := sdk.AccAddressFromBech32(action.String2)

		amount1, err = k.getAmountWallet(ctx, address, action.String1, action.Amount)
		if err != nil {
			return
		}

		coins := sdk.NewCoins(sdk.NewCoin(action.String1, amount1))
		if err = k.BankKeeper.SendCoins(ctx, address, receiver, coins); err != nil {
			err = fmt.Errorf("could not send coins: %w", err)
			return
		}

	case types.ActionWithdrawRewardsAndStake:
		pseudoRandomNumber := automationIndex + automationExecutionIndex + actionIndex

		var validator string
		validator, err = k.withdrawRewardsAndStake(ctx, address, action.String1, pseudoRandomNumber)
		if err != nil {
			err = fmt.Errorf("could not withdraw and stake: %w", err)
			return
		}

		string2 = validator

	case types.ActionWithdrawRewards:
		var rewards sdk.Coins
		rewards, err = k.withdrawRewards(ctx, address)
		if err != nil {
			err = fmt.Errorf("could not withdraw and stake: %w", err)
			return
		}

		amount1 = rewards.AmountOf(constants.BaseCurrency)
		amount2, err = k.getAmountStaked(ctx, address)

	case types.ActionStake:
		amount1, err = k.getAmountWallet(ctx, address, constants.BaseCurrency, action.Amount)
		if err != nil {
			return
		}

		pseudoRandomNumber := automationIndex + automationExecutionIndex + actionIndex
		strategy := getStakingStrategy(action.String1, action.String2)

		var validator string
		validator, err = k.stake(ctx, address, amount1, strategy, pseudoRandomNumber)
		if err != nil {
			err = fmt.Errorf("could not stake: %w", err)
			return
		}

		amount2, err = k.getAmountStaked(ctx, address)
		string2 = validator

	case types.ActionDepositAutomationFunds:
		amount1, err = k.getAmountWallet(ctx, address, constants.BaseCurrency, action.Amount)
		if err != nil {
			return
		}

		err = k.depositAutomationFunds(ctx, amount1, address.String())
		if err != nil {
			return
		}

		amount2 = k.GetAutomationFunds(ctx, address.String())

	case types.ActionWithdrawAutomationFunds:
		amount1, err = k.getAmountWallet(ctx, address, constants.BaseCurrency, action.Amount)
		if err != nil {
			return
		}

		err = k.withdrawAutomationFunds(ctx, amount1, address.String())
		if err != nil {
			return
		}

		amount2 = k.GetAutomationFunds(ctx, address.String())

	default:
		err = fmt.Errorf("unknown action type: %v", action.ActionType)
	}

	return
}

func (k Keeper) getAmountBorrowable(ctx context.Context, address sdk.AccAddress, denom string, amount string) (math.Int, error) {
	if !k.DenomKeeper.IsBorrowableDenom(ctx, denom) {
		return math.Int{}, denomtypes.ErrInvalidBorrowableDenom
	}

	integer, ok := math.NewIntFromString(amount)
	if ok {
		return integer, nil
	}

	borrowable, err := k.MMKeeper.CalculateBorrowableAmount(ctx, address.String(), denom)
	if err != nil {
		return math.Int{}, err
	}

	return borrowable.Mul(percentageToFactor(amount)).TruncateInt(), nil
}

func (k Keeper) getAmountWithdrawableCollateral(ctx context.Context, address sdk.AccAddress, denom string, amount string) (math.Int, error) {
	if !k.DenomKeeper.IsCollateralDenom(ctx, denom) {
		return math.Int{}, denomtypes.ErrInvalidCollateralDenom
	}

	integer, ok := math.NewIntFromString(amount)
	if ok {
		return integer, nil
	}

	withdrawable, err := k.MMKeeper.CalcWithdrawableCollateralAmount(ctx, address.String(), denom)
	if err != nil {
		return math.Int{}, err
	}

	return withdrawable.Mul(percentageToFactor(amount)).TruncateInt(), nil
}

func (k Keeper) getAmountWallet(ctx context.Context, address sdk.AccAddress, denom, amountString string) (math.Int, error) {
	if !k.DenomKeeper.IsValidDenom(ctx, denom) {
		return math.Int{}, denomtypes.ErrInvalidDexAsset
	}

	spendable := k.BankKeeper.SpendableCoin(ctx, address, denom).Amount
	if spendable.LTE(math.ZeroInt()) {
		return math.Int{}, types.ErrNoFunds
	}

	var amount math.Int
	if types.RegexPercentage.Match([]byte(amountString)) {
		walletAmount := spendable.ToLegacyDec()
		walletAmount = walletAmount.Mul(percentageToFactor(amountString))
		amount = walletAmount.TruncateInt()
		if spendable.LTE(math.ZeroInt()) {
			return math.Int{}, types.ErrNoFunds
		}
	} else {
		var ok bool
		if amount, ok = math.NewIntFromString(amountString); ok {
			if spendable.LT(amount) {
				return math.Int{}, types.ErrNotEnoughFunds
			}
		}

		if amount.IsNegative() {
			amount = spendable.Add(amount)
			if spendable.LTE(math.ZeroInt()) {
				return math.Int{}, types.ErrNotEnoughFunds
			}

			return amount, nil
		}
	}

	return amount, nil
}

func (k Keeper) getAmountLiquidity(ctx context.Context, address sdk.AccAddress, denom, amount string) math.Int {
	integer, ok := math.NewIntFromString(amount)
	if ok {
		return integer
	}

	liquidityAmount := k.DexKeeper.GetLiquidityByAddress(ctx, denom, address.String()).ToLegacyDec()
	liquidityAmount = liquidityAmount.Mul(percentageToFactor(amount))
	return liquidityAmount.TruncateInt()
}

func (k Keeper) getActionsCost(ctx context.Context, actions []types.Action) (sum int64) {
	params := k.GetParams(ctx)

	for _, action := range actions {
		sum += int64(getActionCost(params, action.ActionType))
	}

	return
}

func getActionCost(params types.Params, actionType int64) uint64 {
	switch actionType {
	default:
		return params.AutomationFeeAction
	}
}

func percentageToFactor(percentage string) math.LegacyDec {
	percentage = strings.TrimSuffix(percentage, "%")
	value, _ := strconv.Atoi(percentage)
	return math.LegacyNewDecWithPrec(int64(value), 2)
}

func errorIsOf(err error, errs []error) bool {
	for _, e := range errs {
		if errors.IsOf(err, e) {
			return true
		}
	}

	return false
}

func stringToInt(intString string) (*math.Int, error) {
	if intString == "" {
		return nil, nil
	}

	intString = strings.ReplaceAll(intString, ",", "")
	integer, ok := math.NewIntFromString(intString)
	if !ok {
		return nil, types.ErrInvalidIntegerFormat
	}

	return &integer, nil
}
