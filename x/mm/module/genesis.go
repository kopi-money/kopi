package mm

import (
	"context"

	"github.com/cosmos/cosmos-sdk/cache"

	"github.com/kopi-money/kopi/x/mm/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState types.GenesisState) {
	if err := cache.Transact(ctx, func(innerCtx context.Context) error {
		// this line is used by starport scaffolding # genesis/module/init
		if err := k.SetParams(innerCtx, genState.Params); err != nil {
			return err
		}

		k.SetNextLoanIndex(innerCtx, genState.NextLoanIndex)

		for _, loans := range genState.Loans {
			k.SetLoanSum(innerCtx, types.LoanSum{
				Denom:     loans.Denom,
				NumLoans:  uint64(len(loans.Loans)),
				LoanSum:   loans.LoanSum,
				WeightSum: loans.WeightSum,
			})

			for _, loan := range loans.Loans {
				k.SetLoan(innerCtx, loans.Denom, loan.Address, types.Loan{
					Index:  loan.Index,
					Weight: loan.Weight,
				})
			}
		}

		for _, collaterals := range genState.Collaterals {
			for _, collateral := range collaterals.Collaterals {
				k.SetCollateral(innerCtx, collaterals.Denom, collateral.Address, collateral.Amount)
			}
		}

		for _, denomRedemptions := range genState.DenomRedemptions {
			for _, denomRedemption := range denomRedemptions.Redemptions {
				if err := k.SetRedemption(innerCtx, denomRedemptions.Denom, types.Redemption{
					Address: denomRedemption.Address,
					AddedAt: denomRedemption.AddedAt,
					Amount:  denomRedemption.Amount,
					Fee:     denomRedemption.Fee,
				}); err != nil {
					panic(err)
				}
			}
		}

		return nil
	}); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx context.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	nli, _ := k.GetNextLoanIndex(ctx)

	genesis.NextLoanIndex = nli
	genesis.Loans = k.GetGenesisLoans(ctx)
	genesis.Collaterals = k.GetAllDenomCollaterals(ctx)
	genesis.DenomRedemptions = k.GetDenomRedemptions(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
