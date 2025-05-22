package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/mm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetMarketStats(ctx context.Context, _ *types.GetMarketStatsQuery) (*types.GetMarketStatsResponse, error) {
	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("get reference denom: %w", err)
	}

	var (
		acc   = k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
		vault = k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

		totalBorrowed   = math.LegacyZeroDec()
		totalBorrowable = math.LegacyZeroDec()
		totalCollateral = math.LegacyZeroDec()
		totalRedeeming  = math.LegacyZeroDec()
		totalInterest   = math.LegacyZeroDec()

		availableUSD math.LegacyDec
		borrowedUSD  math.LegacyDec
		redeemingUSD math.LegacyDec
	)

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		available := vault.AmountOf(cAsset.BaseDexDenom)
		availableUSD, err = k.DenomKeeper.GetValueIn(ctx, cAsset.BaseDexDenom, referenceDenom, available.ToLegacyDec())
		if err != nil {
			return nil, err
		}

		borrowed := k.GetLoanSumWithDefault(ctx, cAsset.BaseDexDenom).LoanSum
		borrowedUSD, err = k.DenomKeeper.GetValueIn(ctx, cAsset.BaseDexDenom, referenceDenom, borrowed)
		if err != nil {
			return nil, err
		}

		redeeming := k.GetRedemptionSum(ctx, cAsset.DexDenom)
		redeemingUSD, err = k.DenomKeeper.GetValueIn(ctx, cAsset.DexDenom, referenceDenom, redeeming.ToLegacyDec())
		if err != nil {
			return nil, err
		}

		utilityRate := k.getUtilityRate(ctx, cAsset)
		interestRate := k.calculateInterestRate(ctx, utilityRate)

		totalBorrowed = totalBorrowed.Add(borrowedUSD)
		totalBorrowable = totalBorrowable.Add(availableUSD)
		totalRedeeming = totalRedeeming.Add(redeemingUSD)
		totalInterest = totalInterest.Add(borrowedUSD.Mul(interestRate))
	}

	var providedUSD math.LegacyDec
	for _, denom := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		provided := k.getCollateralSum(ctx, denom.DexDenom)
		providedUSD, err = k.DenomKeeper.GetValueIn(ctx, denom.DexDenom, referenceDenom, provided.ToLegacyDec())
		if err != nil {
			return nil, err
		}

		totalCollateral = totalCollateral.Add(providedUSD)
	}

	totalDeposited := totalBorrowed.Add(totalBorrowable)
	utilityRate := math.LegacyZeroDec()
	if totalDeposited.IsPositive() {
		utilityRate = totalBorrowed.Quo(totalDeposited) // C
	}

	weightedInterestRate := math.LegacyNewDecWithPrec(5, 2)
	if totalBorrowed.IsPositive() {
		weightedInterestRate = totalInterest.Quo(totalBorrowed) // C
	}

	return &types.GetMarketStatsResponse{
		TotalCollateral: totalCollateral.String(),
		TotalBorrowable: totalBorrowable.String(),
		TotalBorrowed:   totalBorrowed.String(),
		TotalDeposited:  totalDeposited.String(),
		TotalRedeeming:  totalRedeeming.String(),
		InterestRate:    weightedInterestRate.String(),
		UtilityRate:     utilityRate.String(),
	}, nil
}

func (k Keeper) GetUserStats(ctx context.Context, req *types.GetUserStatsQuery) (*types.GetUserStatsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	totalDeposited, totalRedeeming, err := k.getDepositUserStats(ctx, req.Address)
	if err != nil {
		return nil, err
	}

	totalCollateral, totalBorrowable, err := k.getCollateralUserSumUSD(ctx, req.Address)
	if err != nil {
		return nil, err
	}

	totalBorrowed, weightedInterestRateDec, err := k.getUserLoansSumUSD(ctx, req.Address)
	if err != nil {
		return nil, err
	}

	var utilityRate, weightedInterestRate string
	if totalBorrowable.IsPositive() {
		utilityRate = totalBorrowed.Quo(totalBorrowable).String() // C
		weightedInterestRate = weightedInterestRateDec.String()
	}

	return &types.GetUserStatsResponse{
		TotalDeposited:  totalDeposited.String(),
		TotalCollateral: totalCollateral.String(),
		TotalBorrowed:   totalBorrowed.String(),
		TotalRedeeming:  totalRedeeming.String(),
		TotalBorrowable: totalBorrowable.String(),
		UtilityRate:     utilityRate,
		InterestRate:    weightedInterestRate,
	}, nil
}

func (k Keeper) getDepositUserStats(ctx context.Context, address string) (math.LegacyDec, math.LegacyDec, error) {
	acc, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return math.LegacyDec{}, math.LegacyDec{}, types.ErrInvalidAddress
	}

	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return math.LegacyDec{}, math.LegacyDec{}, fmt.Errorf("get reference denom: %w", err)
	}

	var (
		totalDeposited = math.LegacyZeroDec()
		totalRedeeming = math.LegacyZeroDec()
		coins          = k.BankKeeper.SpendableCoins(ctx, acc)
		cAssetUSD      math.LegacyDec
		redeemingUSD   math.LegacyDec
	)

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		amountCAsset := coins.AmountOf(cAsset.DexDenom)

		redeeming, found := k.redemptions.Get(ctx, cAsset.BaseDexDenom, address)
		if !found {
			redeeming.Amount = math.ZeroInt()
		}

		amountBase := k.ConvertToBaseAmount(ctx, cAsset, amountCAsset.ToLegacyDec())
		cAssetUSD, err = k.DenomKeeper.GetValueIn(ctx, cAsset.BaseDexDenom, referenceDenom, amountBase)
		if err != nil {
			return totalDeposited, totalRedeeming, err
		}

		redeemingUSD, err = k.DenomKeeper.GetValueIn(ctx, cAsset.DexDenom, referenceDenom, redeeming.Amount.ToLegacyDec())
		if err != nil {
			return totalDeposited, totalRedeeming, err
		}

		totalDeposited = totalDeposited.Add(cAssetUSD)
		totalRedeeming = totalRedeeming.Add(redeemingUSD)
	}

	return totalDeposited, totalRedeeming, nil
}

func (k Keeper) getUserLoansSumBase(ctx context.Context, address string) (math.LegacyDec, error) {
	sum := math.LegacyZeroDec()

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		loanValue := k.GetLoanValue(ctx, cAsset.BaseDexDenom, address)
		valueBase, err := k.DenomKeeper.GetValueInBase(ctx, cAsset.BaseDexDenom, loanValue)
		if err != nil {
			return sum, err
		}

		sum = sum.Add(valueBase)
	}

	return sum, nil
}

func (k Keeper) getUserLoansSumUSD(ctx context.Context, address string) (math.LegacyDec, math.LegacyDec, error) {
	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return math.LegacyDec{}, math.LegacyDec{}, fmt.Errorf("get reference denom: %w", err)
	}

	var (
		sum             = math.LegacyZeroDec()
		interestRateSum = math.LegacyZeroDec()
		valueUSD        math.LegacyDec
	)

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		loanValue := k.GetLoanValue(ctx, cAsset.BaseDexDenom, address)
		valueUSD, err = k.DenomKeeper.GetValueIn(ctx, cAsset.BaseDexDenom, referenceDenom, loanValue)
		if err != nil {
			return sum, interestRateSum, err
		}

		sum = sum.Add(valueUSD)

		interestRate := k.CalculateInterestRate(ctx, cAsset)
		interestRateSum = interestRateSum.Add(interestRate.Mul(valueUSD))
	}

	interestRate := math.LegacyZeroDec()
	if sum.IsPositive() {
		interestRate = interestRateSum.Quo(sum) // C
	}

	return sum, interestRate, nil
}

func (k Keeper) getCollateralUserSumUSD(ctx context.Context, address string) (math.LegacyDec, math.LegacyDec, error) {
	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return math.LegacyDec{}, math.LegacyDec{}, fmt.Errorf("get reference denom: %w", err)
	}

	var (
		valueDepositUSD    math.LegacyDec
		valueBorrowableUSD math.LegacyDec
	)

	sumDeposit := math.LegacyZeroDec()
	sumBorrowable := math.LegacyZeroDec()

	for _, denom := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		amount, found := k.collateral.Get(ctx, denom.DexDenom, address)
		if !found {
			continue
		}

		valueDepositUSD, err = k.DenomKeeper.GetValueIn(ctx, denom.DexDenom, referenceDenom, amount.Amount.ToLegacyDec())
		if err != nil {
			return sumDeposit, sumBorrowable, err
		}

		collateralLTV := amount.Amount.ToLegacyDec().Mul(denom.Ltv)
		valueBorrowableUSD, err = k.DenomKeeper.GetValueIn(ctx, denom.DexDenom, referenceDenom, collateralLTV)
		if err != nil {
			return sumDeposit, sumBorrowable, err
		}

		sumDeposit = sumDeposit.Add(valueDepositUSD)
		sumBorrowable = sumBorrowable.Add(valueBorrowableUSD)
	}

	return sumDeposit, sumBorrowable, nil
}
