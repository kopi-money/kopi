package mm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/cache"

	"github.com/kopi-money/kopi/x/mm/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(goCtx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	if err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		// this line is used by starport scaffolding # genesis/module/init
		if err := k.SetParams(ctx, genState.Params); err != nil {
			return err
		}

		k.SetNextLoanIndex(ctx, genState.NextLoanIndex)

		for _, loans := range genState.Loans {
			k.SetLoanSum(ctx, types.LoanSum{
				Denom:     loans.Denom,
				NumLoans:  uint64(len(loans.Loans)),
				LoanSum:   loans.LoanSum,
				WeightSum: loans.WeightSum,
			})

			for _, loan := range loans.Loans {
				k.SetLoan(ctx, loans.Denom, *loan)
			}
		}

		for _, collaterals := range genState.Collaterals {
			for _, collateral := range collaterals.Collaterals {
				k.SetCollateral(ctx, collaterals.Denom, collateral.Address, collateral.Amount)
			}
		}

		for _, denomRedemptions := range genState.DenomRedemptions {
			for _, denomRedemption := range denomRedemptions.Redemptions {
				if err := k.SetRedemption(ctx, denomRedemptions.Denom, types.Redemption{
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
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
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
