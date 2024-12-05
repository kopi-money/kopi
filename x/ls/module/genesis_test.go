package ls_test

import (
	"testing"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/testutil/nullify"
	ls "github.com/kopi-money/kopi/x/ls/module"
	"github.com/kopi-money/kopi/x/ls/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.LsKeeper(t)
	ls.InitGenesis(ctx, k, genesisState)
	got := ls.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
