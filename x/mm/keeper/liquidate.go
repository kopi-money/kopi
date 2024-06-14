package keeper

import (
	"context"
	"cosmossdk.io/collections"
	"fmt"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/pkg/errors"
	"sort"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/utils"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k Keeper) HandleLiquidations(ctx context.Context, eventManager sdk.EventManagerI) error {
	tradeBalances := dexkeeper.NewTradeBalances()
	ordersCaches := k.DexKeeper.NewOrdersCaches(ctx)

	collateralDenomValues, err := k.getCollateralDenomsByValue(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get collateral denoms by value")
	}

	if len(collateralDenomValues) == 0 {
		return nil
	}

	for _, borrower := range k.getBorrowers(ctx) {
		if err = k.handleBorrowerLiquidation(ctx, eventManager, tradeBalances, ordersCaches, collateralDenomValues, borrower); err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not handle liquidations for %v", borrower))
		}
	}

	if err = tradeBalances.Settle(ctx, k.BankKeeper); err != nil {
		return errors.Wrap(err, "could not settle trade balances")
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
		value, err := k.DexKeeper.GetDenomValue(ctx, collateralDenom.Denom)
		if err != nil {
			k.logger.Error(fmt.Sprintf("could not get denom value for %v", collateralDenom.Denom))
			continue
		}

		denomValues = append(denomValues, DenomValue{collateralDenom.Denom, value})
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
func (k Keeper) handleBorrowerLiquidation(ctx context.Context, eventManager sdk.EventManagerI, tradeBalances dextypes.TradeBalances, ordersCaches *dextypes.OrdersCaches, collateralDenoms []string, borrower string) error {
	collateralBaseValue, err := k.calculateCollateralBaseValue(ctx, borrower)
	if err != nil {
		return errors.Wrap(err, "could not calculate collateral base value")
	}

	loanBaseValue, err := k.calculateLoanBaseValue(ctx, borrower)
	if err != nil {
		return errors.Wrap(err, "could not calculate loan base value")
	}

	if loanBaseValue.LTE(collateralBaseValue) {
		return nil
	}

	discountedCollateralValue := collateralBaseValue.Mul(k.GetParams(ctx).CollateralDiscount)
	excessAmountBase := loanBaseValue.Sub(discountedCollateralValue)
	loans := k.getUserLoans(ctx, borrower)

	sort.SliceStable(loans, func(i, j int) bool {
		return loans[i].Index < loans[j].Index
	})

	for _, loan := range loans {
		if loanUnderMinimumThreshold(loan.cAsset, loan.value) {
			k.updateLoan(ctx, loan.cAsset.BaseDenom, loan.Address, loan.value.Neg())

			eventManager.EmitEvent(
				sdk.NewEvent("loan_repaid",
					sdk.Attribute{Key: "address", Value: loan.Address},
					sdk.Attribute{Key: "denom", Value: loan.cAsset.BaseDenom},
					sdk.Attribute{Key: "index", Value: strconv.Itoa(int(loan.Index))},
					sdk.Attribute{Key: "amount", Value: loan.value.String()},
				),
			)

			eventManager.EmitEvent(
				sdk.NewEvent("loan_removed",
					sdk.Attribute{Key: "address", Value: loan.Address},
					sdk.Attribute{Key: "denom", Value: loan.cAsset.BaseDenom},
					sdk.Attribute{Key: "index", Value: strconv.Itoa(int(loan.Index))},
				),
			)

			continue
		}

		if err = k.liquidateCollateral(ctx, eventManager, tradeBalances, ordersCaches, collateralDenoms, loan.cAsset, loan.Loan, &excessAmountBase); err != nil {
			return errors.Wrap(err, "could not liquidate collateral")
		}
	}

	return nil
}

func loanUnderMinimumThreshold(cAsset *denomtypes.CAsset, loanValue math.LegacyDec) bool {
	if cAsset.MinimumLoanSize.IsNil() {
		return false
	}

	return cAsset.MinimumLoanSize.GT(math.ZeroInt()) && loanValue.LT(cAsset.MinimumLoanSize.ToLegacyDec())
}

// liquidateCollateral calculates for each collateral denom how much collateral to sell such as to repay the loan and lower
// excess borrow amount. Sold collateral is sent to the vault.
func (k Keeper) liquidateCollateral(ctx context.Context, eventManager sdk.EventManagerI, tradeBalances dextypes.TradeBalances, ordersCaches *dextypes.OrdersCaches, collateralDenoms []string, cAsset *denomtypes.CAsset, loan types.Loan, excessAmountBase *math.LegacyDec) error {
	addr, _ := sdk.AccAddressFromBech32(loan.Address)
	repayAmount := math.LegacyZeroDec()

	excessAmount, err := k.DexKeeper.GetValueIn(ctx, utils.BaseCurrency, cAsset.BaseDenom, *excessAmountBase)
	if err != nil {
		return err
	}

	loanValue := k.GetLoanValue(ctx, cAsset.BaseDenom, loan.Address)
	// There might be loans in multiple denoms, but the excess amount for this loan must not be larger than the loan
	// itself. If the excessAmount is larger than this loan, it means the next loan will be repaid as well.
	excessAmount = math.LegacyMinDec(excessAmount, loanValue)

	var amountReceived math.Int
	for _, collateralDenom := range collateralDenoms {
		amountReceived, err = k.processLiquidation(ctx, tradeBalances, ordersCaches, cAsset, excessAmount, collateralDenom, addr.String())
		if err != nil {
			//if errors.Is(err, dextypes.ErrTradeAmountTooSmall) || errors.Is(err, dextypes.ErrZeroPrice) {
			//	k.logger.Debug(errors.Wrap(err, "could not execute trade").Error())
			//}

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
	if excessRepayAmount.GT(math.LegacyZeroDec()) {
		poolVaultAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault).GetAddress().String()
		tradeBalances.AddTransfer(poolVaultAcc, addr.String(), cAsset.BaseDenom, excessRepayAmount.TruncateInt())
	}

	k.updateLoan(ctx, cAsset.BaseDenom, loan.Address, repayAmount.Neg())

	repayAmountBase, err := k.DexKeeper.GetValueIn(ctx, cAsset.BaseDenom, utils.BaseCurrency, repayAmount)
	if err != nil {
		return errors.Wrap(err, "could not convert repay amount to base currency")
	}

	*excessAmountBase = (*excessAmountBase).Sub(repayAmountBase)

	eventManager.EmitEvent(
		sdk.NewEvent("loan_liquidation",
			sdk.Attribute{Key: "index", Value: strconv.Itoa(int(loan.Index))},
			sdk.Attribute{Key: "address", Value: loan.Address},
			sdk.Attribute{Key: "denom", Value: cAsset.BaseDenom},
			sdk.Attribute{Key: "repaid", Value: repayAmount.String()},
		),
	)

	return nil
}

func (k Keeper) processLiquidation(ctx context.Context, tradeBalances dextypes.TradeBalances, ordersCaches *dextypes.OrdersCaches, cAsset *denomtypes.CAsset, excessAmount math.LegacyDec, collateralDenom, address string) (math.Int, error) {
	key := collections.Join(collateralDenom, address)
	collateral, found := k.collateral.Get(ctx, key)
	if !found {
		return math.ZeroInt(), nil
	}

	accPoolCollateral := k.AccountKeeper.GetModuleAccount(ctx, types.PoolCollateral).GetAddress().String()
	accPoolVault := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault).GetAddress().String()

	var amountRepaid, usedAmount math.Int
	if collateralDenom == cAsset.BaseDenom {
		amountRepaid = math.MinInt(collateral.Amount, excessAmount.TruncateInt())
		usedAmount = amountRepaid

		tradeBalances.AddTransfer(accPoolCollateral, accPoolVault, collateralDenom, usedAmount)
	} else {
		tradeCtx := dextypes.TradeContext{
			Context:             ctx,
			CoinSource:          accPoolCollateral,
			CoinTarget:          accPoolVault,
			DiscountAddress:     address,
			GivenAmount:         excessAmount.TruncateInt(),
			TradeDenomStart:     cAsset.BaseDenom,
			TradeDenomEnd:       collateralDenom,
			AllowIncomplete:     true,
			ProtocolTrade:       true,
			OrdersCaches:        ordersCaches,
			ExcludeFromDiscount: true,
			TradeBalances:       tradeBalances,
		}

		// In order to calculate how much we need to give to receive the excess amount, we simulate an inverse trade,
		// ie in the opposite direction. The resulting amount is used for the actual trade in the actual correct
		// direction, meaning from the collateral denom to the loan denom.
		amountToGive, _, _, _ := k.DexKeeper.TradeSimulation(tradeCtx)
		tradeCtx.GivenAmount = math.MinInt(collateral.Amount, amountToGive)
		tradeCtx.TradeDenomStart = collateralDenom
		tradeCtx.TradeDenomEnd = cAsset.BaseDenom

		var err error
		usedAmount, amountRepaid, _, _, _, err = k.DexKeeper.ExecuteTrade(tradeCtx)
		if err != nil {
			return math.Int{}, err
		}
	}

	newAmount := collateral.Amount.Sub(usedAmount)
	k.SetCollateral(ctx, collateralDenom, address, newAmount)

	return amountRepaid, nil
}
