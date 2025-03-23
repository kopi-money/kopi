package keeper

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/cache"
	"sort"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k Keeper) HandleLiquidations(ctx context.Context) error {
	collateralDenomValues, err := k.getCollateralDenomsByValue(ctx)
	if err != nil {
		return fmt.Errorf("could not get collateral denoms by value: %w", err)
	}

	if len(collateralDenomValues) == 0 {
		return nil
	}

	for _, borrower := range k.getBorrowers(ctx) {
		if err = k.handleBorrowerLiquidation(ctx, collateralDenomValues, borrower); err != nil {
			return fmt.Errorf("could not handle liquidations for %v: %w", borrower, err)
		}
	}

	return nil
}

// getCollateralDenomsByValue returns a list of all whitelisted collateral tokens sorted DESC by their value on the dex.
func (k Keeper) getCollateralDenomsByValue(ctx context.Context) ([]string, error) {
	type DenomValue struct {
		denom string
		value math.LegacyDec
	}

	var denomValues []DenomValue
	for _, collateralDenom := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		value, err := k.DexKeeper.GetDenomValue(ctx, collateralDenom.DexDenom)
		if err != nil {
			k.Logger().Error(fmt.Sprintf("could not get denom value for %v", collateralDenom.DexDenom))
			continue
		}

		denomValues = append(denomValues, DenomValue{collateralDenom.DexDenom, value})
	}

	sort.SliceStable(denomValues, func(i, j int) bool {
		return denomValues[i].value.GT(denomValues[j].value)
	})

	var denoms []string
	for _, denomValue := range denomValues {
		denoms = append(denoms, denomValue.denom)
	}

	return denoms, nil
}

// handleBorrowerLiquidation compares with loan amount with the maximum allowed amount given the deposited collateral. A
// loan is only liquidated when the excess borrowed amount is bigger than a predetermined amount such as to prevent
// micro trades.
func (k Keeper) handleBorrowerLiquidation(ctx context.Context, collateralDenoms []string, borrower string) error {
	collateralBaseValue, err := k.calculateCollateralBaseValue(ctx, borrower)
	if err != nil {
		return fmt.Errorf("could not calculate collateral base value: %w", err)
	}

	loanBaseValue, loanValues, err := k.calculateLoanBaseValue(ctx, borrower)
	if err != nil {
		return fmt.Errorf("could not calculate loan base value: %w", err)
	}

	if loanBaseValue.LTE(collateralBaseValue) {
		return nil
	}

	discountFactor := math.LegacyOneDec().Sub(k.GetParams(ctx).CollateralDiscount)
	discountedCollateralValue := collateralBaseValue.Mul(discountFactor)
	excessAmountBase := loanBaseValue.Sub(discountedCollateralValue)
	loans := k.getUserLoans(ctx, borrower)

	sort.SliceStable(loans, func(i, j int) bool {
		return loans[i].Index < loans[j].Index
	})

	for _, loan := range loans {
		if loanUnderMinimumThreshold(loan.cAsset, loan.value) {
			_ = cache.Transact(ctx, func(innerCtx context.Context) error {
				k.updateLoan(innerCtx, loan.cAsset.BaseDexDenom, borrower, loan.value.Neg())
				return nil
			})

			sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
				sdk.NewEvent("loan_repaid",
					sdk.Attribute{Key: "address", Value: borrower},
					sdk.Attribute{Key: "denom", Value: loan.cAsset.BaseDexDenom},
					sdk.Attribute{Key: "index", Value: strconv.Itoa(int(loan.Index))},
					sdk.Attribute{Key: "amount", Value: loan.value.String()},
				),
			)

			sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
				sdk.NewEvent("loan_removed",
					sdk.Attribute{Key: "address", Value: borrower},
					sdk.Attribute{Key: "denom", Value: loan.cAsset.BaseDexDenom},
					sdk.Attribute{Key: "index", Value: strconv.Itoa(int(loan.Index))},
				),
			)

			continue
		}

		if err = cache.TransactWithNewMultiStore(ctx, func(innerCtx context.Context) error {
			tradeBalances := dexkeeper.NewTradeBalances()
			ordersCaches := k.DexKeeper.NewOrdersCaches(innerCtx)
			if err = k.liquidateCollateral(innerCtx, tradeBalances, ordersCaches, collateralDenoms, loan.cAsset, loan.Loan, borrower, loanValues[loan.cAsset.BaseDexDenom], &excessAmountBase); err != nil {
				return fmt.Errorf("liquidate collateral: %w", err)
			}

			if err = tradeBalances.Settle(ctx, k.BankKeeper); err != nil {
				return fmt.Errorf("settle trade balances: %w", err)
			}

			return nil
		}); err != nil {
			k.Logger().Error(fmt.Sprintf("could not liquidate collateral (%v): %v", loan.Index, err))
		}
	}

	return nil
}

func loanUnderMinimumThreshold(cAsset denomtypes.CAsset, loanValue math.LegacyDec) bool {
	if cAsset.MinimumLoanSize.IsNil() {
		return false
	}

	return cAsset.MinimumLoanSize.IsPositive() && loanValue.LT(cAsset.MinimumLoanSize.ToLegacyDec())
}

// liquidateCollateral calculates for each collateral denom how much collateral to sell such as to repay the loan and lower
// excess borrow amount. Sold collateral is sent to the vault.
func (k Keeper) liquidateCollateral(ctx context.Context, tradeBalances dextypes.TradeBalances, ordersCaches *dextypes.OrdersCaches, collateralDenoms []string, cAsset denomtypes.CAsset, loan types.Loan, borrower string, loanValue math.LegacyDec, excessAmountBase *math.LegacyDec) error {
	addr, _ := sdk.AccAddressFromBech32(borrower)
	repayAmount := math.LegacyZeroDec()

	excessAmount, err := k.DenomKeeper.GetValueIn(ctx, constants.BaseCurrency, cAsset.BaseDexDenom, *excessAmountBase)
	if err != nil {
		return err
	}

	// There might be loans in multiple denoms, but the excess amount for this loan must not be larger than the loan
	// itself. If the excessAmount is larger than this loan, it means the next loan will be repaid as well.
	excessAmount = math.LegacyMinDec(excessAmount, loanValue)

	var amountReceived math.Int
	for _, collateralDenom := range collateralDenoms {
		amountReceived, err = k.processLiquidation(ctx, tradeBalances, ordersCaches, cAsset, excessAmount, collateralDenom, addr.String())
		if err != nil {
			continue
		}

		excessAmount = excessAmount.Sub(amountReceived.ToLegacyDec())
		repayAmount = repayAmount.Add(amountReceived.ToLegacyDec())
	}

	if repayAmount.IsZero() {
		return nil
	}

	// In case we liquidated more than the loan was worth, the excesses funds will be sent to the user.
	excessRepayAmount := repayAmount.Sub(loanValue)
	if excessRepayAmount.IsPositive() {
		repayAmount = repayAmount.Sub(excessRepayAmount)

		poolVaultAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault).GetAddress().String()
		tradeBalances.AddTransfer(poolVaultAcc, addr.String(), cAsset.BaseDexDenom, excessRepayAmount.TruncateInt())
	}

	k.updateLoan(ctx, cAsset.BaseDexDenom, borrower, repayAmount.Neg())

	repayAmountBase, err := k.DenomKeeper.GetValueIn(ctx, cAsset.BaseDexDenom, constants.BaseCurrency, repayAmount)
	if err != nil {
		return fmt.Errorf("could not convert repay amount to base currency: %w", err)
	}

	*excessAmountBase = (*excessAmountBase).Sub(repayAmountBase)

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("loan_liquidation",
			sdk.Attribute{Key: "index", Value: strconv.Itoa(int(loan.Index))},
			sdk.Attribute{Key: "address", Value: borrower},
			sdk.Attribute{Key: "denom", Value: cAsset.BaseDexDenom},
			sdk.Attribute{Key: "repaid", Value: repayAmount.TruncateInt().String()},
		),
	)

	return nil
}

func (k Keeper) processLiquidation(ctx context.Context, tradeBalances dextypes.TradeBalances, ordersCaches *dextypes.OrdersCaches, cAsset denomtypes.CAsset, excessAmount math.LegacyDec, collateralDenom, address string) (math.Int, error) {
	collateral, found := k.collateral.Get(ctx, collateralDenom, address)
	if !found {
		return math.ZeroInt(), nil
	}

	accPoolCollateral := k.AccountKeeper.GetModuleAccount(ctx, types.PoolCollateral).GetAddress().String()
	accPoolVault := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault).GetAddress().String()

	var amountRepaid, usedAmount math.Int
	if collateralDenom == cAsset.BaseDexDenom {
		amountRepaid = math.MinInt(collateral.Amount, excessAmount.TruncateInt())
		usedAmount = amountRepaid

		tradeBalances.AddTransfer(accPoolCollateral, accPoolVault, collateralDenom, usedAmount)
	} else {
		tradeCtx := dextypes.TradeContext{
			Context:                ctx,
			CoinSource:             accPoolCollateral,
			CoinTarget:             accPoolVault,
			DiscountAddress:        address,
			TradeAmount:            excessAmount.TruncateInt(),
			MaximumAvailableAmount: collateral.Amount,
			TradeDenomGiving:       collateralDenom,
			TradeDenomReceiving:    cAsset.BaseDexDenom,
			OrdersCaches:           ordersCaches,
			TradeBalances:          tradeBalances,
			ProtocolTrade:          true,
			ExcludeFromDiscount:    true,
		}

		tradeResult, err := k.DexKeeper.ExecuteBuy(tradeCtx)
		if err != nil {
			k.Logger().Info(fmt.Sprintf("could not execute collateral sell: %v", err.Error()))
			return math.Int{}, err
		}

		amountRepaid = tradeResult.AmountReceived
		usedAmount = tradeResult.AmountGiven
	}

	newAmount := collateral.Amount.Sub(usedAmount)
	k.SetCollateral(ctx, collateralDenom, address, newAmount)

	return amountRepaid, nil
}
