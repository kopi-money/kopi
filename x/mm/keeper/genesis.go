package keeper

import (
	"context"

	"github.com/kopi-money/kopi/x/mm/types"
)

func (k Keeper) InitGenesis(ctx context.Context, gs types.GenesisState) error {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		return err
	}

	for _, collateral := range gs.Collaterals {
		k.collateral.Set(ctx, collateral.Denom, collateral.Address, types.Collateral{
			Address: collateral.Address,
			Amount:  collateral.Amount,
		})
	}

	for _, loans := range gs.Loans {
		k.loansSum.Set(ctx, loans.Denom, types.LoanSum{
			Denom:     loans.Denom,
			NumLoans:  uint64(len(loans.Loans)),
			LoanSum:   loans.LoanSum,
			WeightSum: loans.WeightSum,
		})

		for _, loan := range loans.Loans {
			k.loans.Set(ctx, loans.Denom, loan.Address, types.Loan{
				Index:  loan.Index,
				Weight: loan.Weight,
			})
		}
	}

	for _, redemption := range gs.Redemptions {
		k.redemptions.Set(ctx, redemption.Denom, redemption.Address, types.Redemption{
			Address: redemption.Address,
			AddedAt: redemption.AddedAt,
			Amount:  redemption.Amount,
			Fee:     redemption.Fee,
		})
	}

	k.loanNextIndex.Set(ctx, gs.NextLoanIndex)

	return nil
}

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	gs := types.DefaultGenesis()

	gs.Params = k.GetParams(ctx)
	gs.Collaterals = k.exportCollateral(ctx)
	gs.Loans = k.exportLoans(ctx)
	gs.Redemptions = k.exportRedemptions(ctx)
	gs.NextLoanIndex, _ = k.loanNextIndex.Get(ctx)

	return gs
}
