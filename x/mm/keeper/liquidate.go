package keeper

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"sort"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/utils"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k Keeper) HandleLiquidations(ctx context.Context, eventManager sdk.EventManagerI) error {
	collateralDenomValues, err := k.getCollateralDenomsByValue(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get collateral denoms by value")
	}

	for _, borrower := range k.getBorrowers(ctx) {
		if err = k.handleBorrowerLiquidation(ctx, eventManager, collateralDenomValues, borrower); err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not handle liquidations for %v", borrower))
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
		value, err := k.DexKeeper.GetDenomValue(ctx, collateralDenom.Denom)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("could not get denom value for %v", collateralDenom.Denom))
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
func (k Keeper) handleBorrowerLiquidation(ctx context.Context, eventManager sdk.EventManagerI, collateralDenoms []string, borrower string) error {
	collateralBaseValue, err := k.calculateCollateralBaseValue(ctx, borrower)
	if err != nil {
		return errors.Wrap(err, "could not calculate collateral base value")
	}

	loanBaseValue, err := k.calculateLoanBaseValue(ctx, borrower)
	if err != nil {
		return errors.Wrap(err, "could not calculate loan base value")
	}

	if loanBaseValue.GT(collateralBaseValue) {
		discountedCollateralValue := collateralBaseValue.Mul(k.GetParams(ctx).CollateralDiscount)
		excessAmountBase := loanBaseValue.Sub(discountedCollateralValue)
		loans := k.getUserLoans(ctx, borrower)

		sort.SliceStable(loans, func(i, j int) bool {
			return loans[i].Index < loans[j].Index
		})

		for _, loan := range loans {
			if err = k.liquidateLoan(ctx, eventManager, collateralDenoms, loan.cAsset, loan.Loan, &excessAmountBase); err != nil {
				return errors.Wrap(err, "could not liquidate loan")
			}
		}
	}

	return nil
}

// liquidateLoan calculates for each collateral denom how much collateral to sell such as to repay the loan and lower
// excess borrow amount. Sold collateral is sent to the vault.
func (k Keeper) liquidateLoan(ctx context.Context, eventManager sdk.EventManagerI, collateralDenoms []string, cAsset *denomtypes.CAsset, loan types.Loan, excessAmountBase *math.LegacyDec) error {
	addr, _ := sdk.AccAddressFromBech32(loan.Address)
	repayAmount := math.LegacyZeroDec()

	excessAmount, err := k.DexKeeper.GetValueIn(ctx, utils.BaseCurrency, cAsset.BaseDenom, excessAmountBase.RoundInt())
	if err != nil {
		return err
	}

	excessAmount = math.LegacyMinDec(excessAmount, loan.Amount)

	var amountReceived math.Int
	for _, collateralDenom := range collateralDenoms {
		amountReceived, err = k.processLiquidation(ctx, eventManager, cAsset, excessAmount, collateralDenom, addr.String())
		if err != nil {
			if errors.Is(err, dextypes.ErrTradeAmountTooSmall) || errors.Is(err, dextypes.ErrZeroPrice) {
				k.logger.Error(errors.Wrap(err, "could not execute trade").Error())
			}

			continue
		}

		excessAmount = excessAmount.Sub(amountReceived.ToLegacyDec())
		repayAmount = repayAmount.Add(amountReceived.ToLegacyDec())
	}

	if repayAmount.Equal(math.LegacyZeroDec()) {
		return nil
	}

	loan.Amount = loan.Amount.Sub(repayAmount)
	if loan.Amount.LT(math.LegacyZeroDec()) {
		amount := loan.Amount.Neg().RoundInt()
		coins := sdk.NewCoins(sdk.NewCoin(cAsset.BaseDenom, amount))
		if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolVault, addr, coins); err != nil {
			return errors.Wrap(err, "could not send excess funds back to user")
		}
	}

	k.SetLoan(ctx, cAsset.BaseDenom, loan)

	repayAmountBase, err := k.DexKeeper.GetValueIn(ctx, cAsset.BaseDenom, utils.BaseCurrency, repayAmount.RoundInt())
	if err != nil {
		return errors.Wrap(err, "could not convert repay amount to base currency")
	}

	*excessAmountBase = (*excessAmountBase).Sub(repayAmountBase)

	eventManager.EmitEvent(
		sdk.NewEvent("loan_liquidation",
			sdk.Attribute{Key: "address", Value: loan.Address},
			sdk.Attribute{Key: "denom", Value: cAsset.BaseDenom},
			sdk.Attribute{Key: "repaid", Value: repayAmount.String()},
		),
	)

	return nil
}

func (k Keeper) processLiquidation(ctx context.Context, eventManager sdk.EventManagerI, cAsset *denomtypes.CAsset, excessAmount math.LegacyDec, collateralDenom, address string) (math.Int, error) {
	collateral, found := k.GetCollateral(ctx, collateralDenom, address)
	if !found {
		return math.ZeroInt(), nil
	}

	var amountRepaid, usedAmount math.Int
	if collateralDenom == cAsset.BaseDenom {
		amountRepaid = math.MinInt(collateral.Amount, excessAmount.TruncateInt())
		usedAmount = amountRepaid
	} else {
		amountToGive, _, _, _ := k.DexKeeper.TradeSimulation(ctx, cAsset.BaseDenom, collateralDenom, address, excessAmount.RoundInt(), false)
		tradeAmount := math.MinInt(collateral.Amount, amountToGive)

		coinSource := k.AccountKeeper.GetModuleAccount(ctx, types.PoolCollateral)
		vaultAddress := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
		options := dextypes.TradeOptions{
			CoinSource:      coinSource.GetAddress(),
			CoinTarget:      vaultAddress.GetAddress(),
			DiscountAddress: sdk.AccAddress(address),
			GivenAmount:     tradeAmount,
			MaxPrice:        nil,
			TradeDenomStart: collateralDenom,
			TradeDenomEnd:   cAsset.BaseDenom,
			AllowIncomplete: true,
			ProtocolTrade:   true,
		}

		var err error
		usedAmount, amountRepaid, _, _, err = k.DexKeeper.ExecuteTrade(ctx, eventManager, options)
		if err != nil {
			return math.Int{}, err
		}
	}

	collateral.Amount = collateral.Amount.Sub(usedAmount)
	k.SetCollateral(ctx, collateralDenom, collateral)

	return amountRepaid, nil
}
