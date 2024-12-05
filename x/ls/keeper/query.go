package keeper

import (
	"github.com/kopi-money/kopi/x/ls/types"
)

var _ types.QueryServer = Keeper{}
