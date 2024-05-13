package mm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/x/mm/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	if genState.NextLoanIndex == nil {
		k.SetNextLoanIndex(ctx, types.NextLoanIndex{Index: 1})
	} else {
		k.SetNextLoanIndex(ctx, *genState.NextLoanIndex)
	}

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
			k.SetCollateral(ctx, collaterals.Denom, *collateral)
		}
	}

	for _, denomRedemptions := range genState.DenomRedemptions {
		for _, denomRedemption := range denomRedemptions.Redemptions {
			k.SetRedemption(ctx, denomRedemptions.Denom, *denomRedemption)
		}
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	nli := k.GetNextLoanIndex(ctx)

	genesis.NextLoanIndex = &nli
	genesis.Loans = k.GetGenesisLoans(ctx)
	genesis.Collaterals = k.GetAllDenomCollaterals(ctx)
	genesis.DenomRedemptions = k.GetDenomRedemptions(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
