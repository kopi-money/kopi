package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	"github.com/kopi-money/kopi/x/blockspeed/types"
)

func (k Keeper) AdjustBlockspeed(ctx context.Context) {
	height := sdk.UnwrapSDKContext(ctx).BlockHeight()
	now := sdk.UnwrapSDKContext(ctx).BlockTime()

	timestamp := now.UnixMilli()
	blockspeed := k.GetBlockspeed(ctx)

	// There is some delay between the genesis creation and the first blocks. To ignore that delay, we use hardcoded
	// values for the first few blocks.
	if blockspeed.PreviousTimestamp == 0 || height <= 10 {
		blockspeed.AverageTime = math.LegacyNewDec(1000) // 1000ms = 1s
	} else {
		diff := timestamp - blockspeed.PreviousTimestamp
		blockspeed.AverageTime = k.calcAverageTime(ctx, blockspeed.AverageTime, diff)
	}

	blockspeed.PreviousTimestamp = timestamp
	k.blockspeed.Set(ctx, blockspeed)
}

func (k Keeper) GetBlockspeed(ctx context.Context) types.Blockspeed {
	blockspeed, _ := k.blockspeed.Get(ctx)
	return blockspeed
}

func (k Keeper) SetBlockspeed(ctx context.Context, blockspeed types.Blockspeed) {
	k.blockspeed.Set(ctx, blockspeed)
}

func (k Keeper) calcAverageTime(ctx context.Context, averageTime math.LegacyDec, timeDiff int64) math.LegacyDec {
	movingAverageFactor := k.movingAverageFactor(ctx)
	averageTime = averageTime.Mul(movingAverageFactor)
	toAdd := math.LegacyOneDec().Sub(movingAverageFactor).Mul(math.LegacyNewDec(timeDiff))
	return averageTime.Add(toAdd)
}

func (k Keeper) GetSecondsPerBlock(ctx context.Context) math.LegacyDec {
	blockspeed := k.GetBlockspeed(ctx)
	blocksPerSecond := blockspeed.AverageTime.Quo(math.LegacyNewDec(1000)) // C
	blocksPerSecond = math.LegacyMinDec(blocksPerSecond, math.LegacyNewDec(5))
	return blocksPerSecond
}

func (k Keeper) GetBlocksPerSecond(ctx context.Context) (math.LegacyDec, error) {
	secondPerBlock := k.GetSecondsPerBlock(ctx)
	if !secondPerBlock.IsPositive() {
		return math.LegacyDec{}, fmt.Errorf("seconds per block is not positive")
	}

	return math.LegacyOneDec().Quo(secondPerBlock), nil // C
}

func (k Keeper) BlocksPerYear(ctx context.Context) (math.LegacyDec, error) {
	secondsPerYear := math.LegacyNewDec(constants.SecondsPerYear)
	blockPerSecond, err := k.GetBlocksPerSecond(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	}

	if blockPerSecond.IsZero() {
		return math.LegacyDec{}, types.ErrDivisionByZero
	}

	return secondsPerYear.Quo(blockPerSecond), nil // C
}
