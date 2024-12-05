package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/kopi-money/kopi/testutil/keeper"
	"github.com/kopi-money/kopi/x/ls/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.LsKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}
