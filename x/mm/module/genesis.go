package mm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/x/mm/keeper"
	"github.com/kopi-money/kopi/x/mm/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)

	if genState.NextLoanIndex == nil {
		k.SetNextLoanIndex(ctx, types.NextLoanIndex{Index: 0})
	} else {
		k.SetNextLoanIndex(ctx, *genState.NextLoanIndex)
	}

	for _, loans := range genState.Loans {
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
	genesis.Loans = k.GetDenomLoans(ctx)
	genesis.Collaterals = k.GetAllDenomCollaterals(ctx)
	genesis.DenomRedemptions = k.GetDenomRedemptions(ctx)

	//repeated Loans loans = 2 [(gogoproto.nullable) = false];
	//repeated Collaterals collaterals = 3 [(gogoproto.nullable) = false];
	//repeated DenomRedemption denom_redemptions = 4 [(gogoproto.nullable) = false];
	//NextLoanIndex next_loan_index = 5;

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
