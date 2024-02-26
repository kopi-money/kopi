package keeper

import (
	"github.com/kopi-money/kopi/x/dex/types"
)

var _ types.QueryServer = Keeper{}
