package keeper

import (
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

var _ types.QueryServer = Keeper{}
