package swap_test

import (
	"testing"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/testutil/nullify"
	swap "github.com/kopi-money/kopi/x/swap/module"
	"github.com/kopi-money/kopi/x/swap/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, _, ctx := keepertest.SwapKeeper(t)
	swap.InitGenesis(ctx, k, genesisState)
	got := swap.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
