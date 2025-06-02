package keeper

import (
	"github.com/kopi-money/kopi/x/txfees/types"
)

var _ types.QueryServer = Keeper{}
