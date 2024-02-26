package keeper

import (
	"github.com/kopi-money/kopi/x/denominations/types"
)

var _ types.QueryServer = Keeper{}
