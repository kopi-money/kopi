package keeper

import (
	"context"
	"cosmossdk.io/collections"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	"github.com/kopi-money/kopi/x/mm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetLoansByDenom(ctx context.Context, req *types.GetLoansByDenomQuery) (*types.GetLoansResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	cAsset, err := k.DenomKeeper.GetCAssetByBaseName(ctx, req.Denom)
	if err != nil {
		return nil, err
	}

	loanSum := k.GetLoanSumWithDefault(ctx, req.Denom)
	utilityRate := k.getUtilityRate(ctx, cAsset)
	interestRate := k.calculateInterestRate(ctx, utilityRate)

	var loans []*types.DenomLoan
	var amountBorrowedUSD math.LegacyDec

	iterator := k.LoanIterator(ctx, cAsset.BaseDenom)
	for iterator.Valid() {
		loan := iterator.GetNext()
		loanValue := k.getLoanValue(loanSum, loan)

		amountBorrowedUSD, err = k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, loanValue)
		if err != nil {
			return nil, err
		}

		loans = append(loans, &types.DenomLoan{
			LoanIndex:         loan.Index,
			Address:           loan.Address,
			AmountBorrowed:    loanValue.String(),
			AmountBorrowedUsd: amountBorrowedUSD.String(),
		})
	}

	return &types.GetLoansResponse{
		Loans:        loans,
		InterestRate: interestRate.String(),
	}, nil
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
		amountAvailableUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, amountAvailable.ToLegacyDec())
		if err != nil {
			return nil, err
		}

		loanSum := k.GetLoanSumWithDefault(ctx, cAsset.BaseDenom).LoanSum
		loanSumUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, loanSum)
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

	addr := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
	vault := k.BankKeeper.SpendableCoins(ctx, addr.GetAddress())

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		utilityRate := k.getUtilityRate(ctx, cAsset)
		interestRate := k.calculateInterestRate(ctx, utilityRate)

		loanSum := k.GetLoanSumWithDefault(ctx, cAsset.BaseDenom)

		loan, has := k.loans.Get(ctx, collections.Join(cAsset.BaseDenom, req.Address))
		loanValue := math.LegacyZeroDec()
		if has {
			loanValue = k.getLoanValue(loanSum, loan)
		}

		vaultAmount := vault.AmountOf(cAsset.BaseDenom)
		amountAvailable, err := k.calculateBorrowableAmount(ctx, req.Address, cAsset.BaseDenom)
		if err != nil {
			return nil, err
		}

		amountAvailable = math.LegacyMinDec(vaultAmount.ToLegacyDec(), amountAvailable)

		amountAvailableUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, amountAvailable)
		if err != nil {
			return nil, err
		}

		amountBorrowedUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, loanValue)
		if err != nil {
			return nil, err
		}

		userLoans = append(userLoans, &types.UserLoanStat{
			Denom:              cAsset.BaseDenom,
			AmountBorrowed:     loanValue.String(),
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

	loanValue := k.GetLoanValue(ctx, cAsset.BaseDenom, req.Address)
	amountUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, loanValue)
	if err != nil {
		return nil, err
	}

	return &types.GetUserDenomLoanResponse{Amount: loanValue.String(), AmountUsd: amountUSD.String()}, nil
}

func (k Keeper) GetNumLoans(ctx context.Context, req *types.GetNumLoansQuery) (*types.GetNumLoansResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &types.GetNumLoansResponse{Num: int64(k.GetLoansNum(ctx))}, nil
}

func (k Keeper) GetValueLoans(ctx context.Context, req *types.GetValueLoansQuery) (*types.GetValueLoansResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	valueUSD := math.LegacyZeroDec()

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		loanSum := k.GetLoanSumWithDefault(ctx, cAsset.BaseDenom)

		value, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, loanSum.LoanSum)
		if err != nil {
			return nil, errors.Wrap(err, "could not get value in usd")
		}

		valueUSD = valueUSD.Add(value)
	}

	return &types.GetValueLoansResponse{Value: valueUSD.String()}, nil
}

func (k Keeper) GetNumAddressLoans(ctx context.Context, req *types.GetNumAddressLoansQuery) (*types.GetNumAddressLoansResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &types.GetNumAddressLoansResponse{Amount: int64(k.GetLoansNumForAddress(ctx, req.Address))}, nil
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
