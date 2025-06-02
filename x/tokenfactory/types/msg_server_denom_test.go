package types_test

import (
	"testing"

	"github.com/kopi-money/kopi/x/tokenfactory/types"
	"github.com/stretchr/testify/require"
)

func TestValidName(t *testing.T) {
	require.NoError(t, types.IsValidDisplayName("test"))
	require.Error(t, types.IsValidDisplayName("te"))
}
