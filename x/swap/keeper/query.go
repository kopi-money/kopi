package keeper

import (
	"github.com/kopi-money/kopi/x/swap/types"
)

var _ types.QueryServer = Keeper{}
