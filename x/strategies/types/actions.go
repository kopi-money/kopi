package types

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/trading"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	mmtypes "github.com/kopi-money/kopi/x/mm/types"
)

const (
	ActionSell = 1 + iota
	ActionBuy
	ActionDeposit
	ActionRedeem
	ActionCollateralAdd
	ActionCollateralWithdraw
	ActionLiquidityAdd
	ActionLiquidityWithdraw
	ActionLoanBorrow
	ActionLoanRepay
	ActionSendCoins
	ActionStake
	ActionDepositAutomationFunds
	ActionWithdrawAutomationFunds
	ActionWithdrawRewardsAndStake
	ActionWithdrawRewards

	// Factory
	ActionFactorySell
	ActionFactoryBuy
	ActionFactoryLiquidityAddBoth
	ActionFactoryLiquidityAddDexDenom
	ActionFactoryLiquidityAddFactoryDenom
	ActionFactoryLiquidityWithdraw
)

var NoAmountActions = []int64{
	ActionWithdrawRewards,
	ActionWithdrawRewardsAndStake,
}

// ValidErrors is a list of errors which will not be logged because they are of no further interest. Covers issues like
// trades that cannot be executed because there is not enough liquidity.
var ValidErrors = []error{
	dextypes.ErrBaseLiqEmpty,
	dextypes.ErrNotEnoughFunds,
	dextypes.ErrNotEnoughUsableLiquidity,
	dextypes.ErrNoLiquidityGiving,
	dextypes.ErrNoLiquidityReceiving,

	trading.ErrRequestedAmountTooLarge,
	trading.ErrEmptyTrade,
	trading.ErrMarketPriceTooHigh,
	trading.ErrPriceTooLow,
	trading.ErrTradeAmountTooSmall,
	trading.ErrInvalidMaxPriceFormat,
	trading.ErrMaxPriceNotPositive,
	trading.ErrTradeAmountNotPositive,

	mmtypes.ErrBorrowLimitExceeded,
	mmtypes.ErrCannotWithdrawCollateral,
	mmtypes.ErrCollateralBorrowLimitExceeded,
	mmtypes.ErrCollateralDepositLimitExceeded,
	mmtypes.ErrInvalidCollateralDenom,
	mmtypes.ErrLoanSizeTooSmall,
	mmtypes.ErrNegativeAmount,
	mmtypes.ErrNotEnoughFunds,
	mmtypes.ErrNotEnoughFundsInVault,
	mmtypes.ErrRedemptionRequestAlreadyExists,
	mmtypes.ErrZeroAmount,

	ErrNotEnoughFunds,
	ErrNonExistingValidator,
}

func checkActions(actions []*Action) error {
	for actionIndex, action := range actions {
		if err := checkAction(action); err != nil {
			return fmt.Errorf("invalid action[%d]: %w", actionIndex, err)
		}
	}

	return nil
}

func checkAction(action *Action) error {
	if err := checkAutomationString(action.String1); err != nil {
		return fmt.Errorf("invalid string1: %w", err)
	}

	if err := checkAutomationString(action.String2); err != nil {
		return fmt.Errorf("invalid string2: %w", err)
	}

	if err := checkAmountString(action.Amount); err != nil {
		return err
	}

	if action.MinimumTradeAmount != "" {
		if _, ok := math.NewIntFromString(action.MinimumTradeAmount); !ok {
			return fmt.Errorf("invalid minimum trade amount")
		}
	}

	return nil
}
