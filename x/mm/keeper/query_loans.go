package keeper

import (
	"context"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	"github.com/kopi-money/kopi/x/mm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetLoans(ctx context.Context, req *types.GetLoansQuery) (*types.GetLoansResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	loans := []*types.UserLoan{}

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		utilityRate := k.getUtilityRate(ctx, cAsset)
		interestRate := k.calculateInterestRate(ctx, utilityRate)

		for _, loan := range k.GetAllLoansByDenom(ctx, cAsset.BaseDenom) {
			amountBorrowedUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, loan.Amount.RoundInt())
			if err != nil {
				return nil, err
			}

			loans = append(loans, &types.UserLoan{
				Denom:             cAsset.BaseDenom,
				AmountBorrowed:    loan.Amount.String(),
				AmountBorrowedUsd: amountBorrowedUSD.String(),
				InterestRate:      interestRate.String(),
			})
		}
	}

	return &types.GetLoansResponse{Loans: loans}, nil
}

func (k Keeper) GetLoansStats(ctx context.Context, req *types.GetLoanStatsQuery) (*types.GetLoanStatsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
	vault := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

	loanStats := []*types.DenomLoanStat{}
	totalAvailableToBorrowUSD := math.LegacyZeroDec()
	totalLoanSumUSD := math.LegacyZeroDec()

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		utilityRate := k.getUtilityRate(ctx, cAsset)
		interestRate := k.calculateInterestRate(ctx, utilityRate)

		amountAvailable := vault.AmountOf(cAsset.BaseDenom)
		amountAvailableUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, amountAvailable)
		if err != nil {
			return nil, err
		}

		loanSum := k.GetLoansSum(ctx, cAsset.BaseDenom)
		loanSumUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, loanSum.RoundInt())
		if err != nil {
			return nil, err
		}

		totalAvailableToBorrowUSD = totalAvailableToBorrowUSD.Add(amountAvailableUSD)
		totalLoanSumUSD = totalAvailableToBorrowUSD.Add(loanSumUSD)

		loanStats = append(loanStats, &types.DenomLoanStat{
			Denom:                cAsset.BaseDenom,
			UtilityRate:          utilityRate.String(),
			InterestRate:         interestRate.String(),
			AvailableToBorrow:    amountAvailable.String(),
			AvailableToBorrowUsd: amountAvailableUSD.String(),
			LoanSum:              loanSum.String(),
			LoanSumUsd:           loanSumUSD.String(),
		})
	}

	return &types.GetLoanStatsResponse{
		LoanStats:                 loanStats,
		TotalAvailableToBorrowUsd: totalAvailableToBorrowUSD.String(),
		TotalLoanSumUsd:           totalLoanSumUSD.String(),
	}, nil
}

func (k Keeper) GetUserLoans(ctx context.Context, req *types.GetUserLoansQuery) (*types.GetUserLoansResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	userLoans := []*types.UserLoanStat{}

	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
	vault := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		utilityRate := k.getUtilityRate(ctx, cAsset)
		interestRate := k.calculateInterestRate(ctx, utilityRate)

		loan, found := k.GetLoan(ctx, cAsset.BaseDenom, req.Address)
		if !found {
			loan.Amount = math.LegacyZeroDec()
		}

		amountAvailable := vault.AmountOf(cAsset.BaseDenom)
		amountAvailableUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, amountAvailable)
		if err != nil {
			return nil, err
		}

		amountBorrowedUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, loan.Amount.RoundInt())
		if err != nil {
			return nil, err
		}

		userLoans = append(userLoans, &types.UserLoanStat{
			Denom:              cAsset.BaseDenom,
			AmountBorrowed:     loan.Amount.String(),
			AmountBorrowedUsd:  amountBorrowedUSD.String(),
			AmountAvailable:    amountAvailable.String(),
			AmountAvailableUsd: amountAvailableUSD.String(),
			InterestRate:       interestRate.String(),
		})
	}

	return &types.GetUserLoansResponse{UserLoans: userLoans}, nil
}

func (k Keeper) GetUserDenomLoan(ctx context.Context, req *types.GetUserDenomLoanQuery) (*types.GetUserDenomLoanResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	cAsset, err := k.DenomKeeper.GetCAssetByBaseName(ctx, req.Denom)
	if err != nil {
		return nil, err
	}

	loan, found := k.GetLoan(ctx, cAsset.BaseDenom, req.Address)
	if !found {
		loan.Amount = math.LegacyZeroDec()
	}

	amountUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, loan.Amount.RoundInt())
	if err != nil {
		return nil, err
	}

	return &types.GetUserDenomLoanResponse{Amount: loan.Amount.String(), AmountUsd: amountUSD.String()}, nil
}

func (k Keeper) GetNumLoans(ctx context.Context, req *types.GetNumLoansQuery) (*types.GetNumLoansResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	iterator := k.LoanIterator(ctx)
	defer iterator.Close()

	var counter int64 = 0
	for ; iterator.Valid(); iterator.Next() {
		counter++
	}

	return &types.GetNumLoansResponse{Num: counter}, nil
}

func (k Keeper) GetValueLoans(ctx context.Context, req *types.GetValueLoansQuery) (*types.GetValueLoansResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	valueUSD := math.LegacyZeroDec()

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		for _, loan := range k.GetAllLoansByDenom(ctx, cAsset.BaseDenom) {
			value, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, loan.Amount.RoundInt())
			if err != nil {
				return nil, errors.Wrap(err, "could not get value in usd")
			}

			valueUSD = valueUSD.Add(value)
		}
	}

	return &types.GetValueLoansResponse{Value: valueUSD.String()}, nil
}

func (k Keeper) GetNumAddressLoans(ctx context.Context, req *types.GetNumAddressLoansQuery) (*types.GetNumAddressLoansResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	iterator := k.LoanIterator(ctx)
	defer iterator.Close()

	var counter int64 = 0
	for ; iterator.Valid(); iterator.Next() {
		var val types.Loan
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		if val.Address == req.Address {
			counter++
		}
	}

	return &types.GetNumAddressLoansResponse{Amount: counter}, nil
}

func (k Keeper) GetAvailableToBorrow(ctx context.Context, req *types.GetAvailableToBorrowRequest) (*types.GetAvailableToBorrowResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	amount, err := k.CalcAvailableToBorrow(ctx, req.Address, req.Denom)
	if err != nil {
		return nil, errors.Wrap(err, "could not calculate available amount to borrow")
	}

	amountUSD, err := k.DexKeeper.GetValueInUSD(ctx, req.Denom, amount)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert amount to usd")
	}

	return &types.GetAvailableToBorrowResponse{
		Amount:    amount.String(),
		AmountUsd: amountUSD.String(),
	}, nil
}
