package types_test

import (
	"testing"

	"github.com/kopi-money/constants"
	"github.com/kopi-money/kopi/x/denominations/types"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

func TestExtractDenom1(t *testing.T) {
	f, d, err := types.ExtractNumberAndString("1")
	require.Equal(t, math.LegacyNewDec(1), f)
	require.Equal(t, "", d)
	require.NoError(t, err)

	f, d, err = types.ExtractNumberAndString("1ukusd")
	require.Equal(t, math.LegacyNewDec(1), f)
	require.Equal(t, constants.KUSD, d)
	require.NoError(t, err)

	f, d, err = types.ExtractNumberAndString("0.5ukusd")
	require.Equal(t, math.LegacyNewDecWithPrec(5, 1), f)
	require.Equal(t, constants.KUSD, d)
	require.NoError(t, err)

	f, d, err = types.ExtractNumberAndString(".5ukusd")
	require.Error(t, err)

	f, d, err = types.ExtractNumberAndString("1ibc/ABC")
	require.NoError(t, err)

	f, d, err = types.ExtractNumberAndString("1ibc/ABC/123")
	require.Error(t, err)
}
