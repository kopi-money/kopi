package keeper

import (
	"context"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/mm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetUserStats(ctx context.Context, req *types.GetUserStatsQuery) (*types.GetUserStatsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	totalDeposited, totalWithdrawing, err := k.getDepositUserStats(ctx, req.Address)
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
	if totalBorrowable.GT(math.LegacyZeroDec()) {
		utilityRate = totalBorrowed.Quo(totalBorrowable).String()
		weightedInterestRate = weightedInterestRateDec.String()
	}

	return &types.GetUserStatsResponse{
		TotalDeposited:   totalDeposited.String(),
		TotalCollateral:  totalCollateral.String(),
		TotalBorrowed:    totalBorrowed.String(),
		TotalWithdrawing: totalWithdrawing.String(),
		TotalBorrowable:  totalBorrowable.String(),
		UtilityRate:      utilityRate,
		InterestRate:     weightedInterestRate,
	}, nil
}

func (k Keeper) getBorrowableAmountUSD(ctx context.Context, address string) (math.LegacyDec, error) {
	sum := math.LegacyZeroDec()
	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		borrowableAmount, err := k.calculateBorrowableAmount(ctx, address, cAsset.BaseDenom)
		if err != nil {
			return sum, err
		}

		borrowableAmountUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, borrowableAmount.TruncateInt())
		if err != nil {
			return sum, err
		}

		sum = sum.Add(borrowableAmountUSD)
	}

	return sum, nil
}

func (k Keeper) getDepositUserStats(ctx context.Context, address string) (math.LegacyDec, math.LegacyDec, error) {
	totalDeposited := math.LegacyZeroDec()
	totalWithdrawing := math.LegacyZeroDec()

	acc, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return totalDeposited, totalWithdrawing, types.ErrInvalidAddress
	}

	coins := k.BankKeeper.SpendableCoins(ctx, acc)

	var cAssetUSD, withdrawalUSD math.LegacyDec
	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		amountCAsset := coins.AmountOf(cAsset.Name)

		redeeming, found := k.GetRedemption(ctx, cAsset.BaseDenom, address)
		if !found {
			redeeming.Amount = math.ZeroInt()
		}

		amountBase := k.ConvertToBaseAmount(ctx, cAsset, amountCAsset)
		cAssetUSD, err = k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, amountBase.RoundInt())
		if err != nil {
			return totalDeposited, totalWithdrawing, err
		}

		withdrawalUSD, err = k.DexKeeper.GetValueInUSD(ctx, cAsset.Name, redeeming.Amount)
		if err != nil {
			return totalDeposited, totalWithdrawing, err
		}

		totalDeposited = totalDeposited.Add(cAssetUSD)
		totalWithdrawing = totalWithdrawing.Add(withdrawalUSD)
	}

	return totalDeposited, totalWithdrawing, nil
}

func (k Keeper) getUserLoansSumBase(ctx context.Context, address string) (math.LegacyDec, error) {
	sum := math.LegacyZeroDec()

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		loan, found := k.GetLoan(ctx, cAsset.BaseDenom, address)
		if !found {
			continue
		}

		valueBase, err := k.DexKeeper.GetValueInBase(ctx, cAsset.BaseDenom, loan.Amount.RoundInt())
		if err != nil {
			return sum, err
		}

		sum = sum.Add(valueBase)
	}

	return sum, nil
}

func (k Keeper) getUserLoansSumUSD(ctx context.Context, address string) (math.LegacyDec, math.LegacyDec, error) {
	sum := math.LegacyZeroDec()
	interestRateSum := math.LegacyZeroDec()

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		loan, found := k.GetLoan(ctx, cAsset.BaseDenom, address)
		if !found {
			continue
		}

		valueUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, loan.Amount.RoundInt())
		if err != nil {
			return sum, interestRateSum, err
		}

		sum = sum.Add(valueUSD)

		interestRate := k.CalculateInterestRate(ctx, cAsset)
		interestRateSum = interestRateSum.Add(interestRate.Mul(valueUSD))
	}

	interestRate := math.LegacyZeroDec()
	if sum.GT(math.LegacyZeroDec()) {
		interestRate = interestRateSum.Quo(sum)
	}

	return sum, interestRate, nil
}

func (k Keeper) getCollateralUserSumUSD(ctx context.Context, address string) (math.LegacyDec, math.LegacyDec, error) {
	sumDeposit := math.LegacyZeroDec()
	sumBorrowable := math.LegacyZeroDec()

	for _, denom := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		amount, found := k.GetCollateral(ctx, denom.Denom, address)
		if !found {
			continue
		}

		valueDepositUSD, err := k.DexKeeper.GetValueInUSD(ctx, denom.Denom, amount.Amount)
		if err != nil {
			return sumDeposit, sumBorrowable, err
		}

		collateralLTV := amount.Amount.ToLegacyDec().Mul(denom.Ltv).RoundInt()
		valueBorrowableUSD, err := k.DexKeeper.GetValueInUSD(ctx, denom.Denom, collateralLTV)
		if err != nil {
			return sumDeposit, sumBorrowable, err
		}

		sumDeposit = sumDeposit.Add(valueDepositUSD)
		sumBorrowable = sumBorrowable.Add(valueBorrowableUSD)
	}

	return sumDeposit, sumBorrowable, nil
}
