package keeper

import (
	"context"
	"cosmossdk.io/collections"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/mm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetMarketStats(ctx context.Context, req *types.GetMarketStatsQuery) (*types.GetMarketStatsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
	vault := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

	totalBorrowed := math.LegacyZeroDec()
	totalBorrowable := math.LegacyZeroDec()
	totalCollateral := math.LegacyZeroDec()
	totalRedeeming := math.LegacyZeroDec()
	totalInterest := math.LegacyZeroDec()

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		available := vault.AmountOf(cAsset.BaseDenom)
		availableUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, available)
		if err != nil {
			return nil, err
		}

		borrowed := k.GetLoanSumWithDefault(ctx, cAsset.BaseDenom).LoanSum
		borrowedUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, borrowed.RoundInt())
		if err != nil {
			return nil, err
		}

		redeeming := k.GetRedemptionSum(ctx, cAsset.BaseDenom)
		redeemingUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, redeeming)
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

	for _, denom := range k.DenomKeeper.GetCollateralDenoms(ctx) {
		provided := k.getCollateralSum(ctx, denom.Denom)
		providedUSD, err := k.DexKeeper.GetValueInUSD(ctx, denom.Denom, provided)
		if err != nil {
			return nil, err
		}

		totalCollateral = totalCollateral.Add(providedUSD)
	}

	totalDeposited := totalBorrowed.Add(totalBorrowable)
	utilityRate := math.LegacyZeroDec()
	if totalDeposited.GT(math.LegacyZeroDec()) {
		utilityRate = totalBorrowed.Quo(totalDeposited)
	}

	weightedInterestRate := math.LegacyNewDecWithPrec(5, 2)
	if totalBorrowed.GT(math.LegacyZeroDec()) {
		weightedInterestRate = totalInterest.Quo(totalBorrowed)
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
	if totalBorrowable.GT(math.LegacyZeroDec()) {
		utilityRate = totalBorrowed.Quo(totalBorrowable).String()
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
	totalRedeeming := math.LegacyZeroDec()

	acc, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return totalDeposited, totalRedeeming, types.ErrInvalidAddress
	}

	coins := k.BankKeeper.SpendableCoins(ctx, acc)

	var cAssetUSD, redeemingUSD math.LegacyDec
	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		amountCAsset := coins.AmountOf(cAsset.Name)

		key := collections.Join(cAsset.BaseDenom, address)
		redeeming, found := k.redemptions.Get(ctx, key)
		if !found {
			redeeming.Amount = math.ZeroInt()
		}

		amountBase := k.ConvertToBaseAmount(ctx, cAsset, amountCAsset)
		cAssetUSD, err = k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, amountBase.RoundInt())
		if err != nil {
			return totalDeposited, totalRedeeming, err
		}

		redeemingUSD, err = k.DexKeeper.GetValueInUSD(ctx, cAsset.Name, redeeming.Amount)
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
		loanValue := k.GetLoanValue(ctx, cAsset.BaseDenom, address)
		valueBase, err := k.DexKeeper.GetValueInBase(ctx, cAsset.BaseDenom, loanValue.RoundInt())
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
		loanValue := k.GetLoanValue(ctx, cAsset.BaseDenom, address)
		valueUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, loanValue.RoundInt())
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
		key := collections.Join(denom.Denom, address)
		amount, found := k.collateral.Get(ctx, key)
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
