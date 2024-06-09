package keeper

import (
	"github.com/kopi-money/kopi/x/mm/types"
)

var _ types.QueryServer = Keeper{}
