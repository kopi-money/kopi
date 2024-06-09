package keeper

import (
	"context"

	"cosmossdk.io/math"
)

type ConstantProduct struct{}

func (cp ConstantProduct) Forward(poolFrom, poolTo, offer math.LegacyDec) math.Int {
	constantProduct := poolFrom.Mul(poolTo)
	amount := poolTo.Sub(constantProduct.Quo(poolFrom.Add(offer)))
	return amount.TruncateInt()
}

func (cp ConstantProduct) Backward(poolFrom, poolTo, result math.LegacyDec) math.Int {
	constantProduct := poolFrom.Mul(poolTo)
	return constantProduct.Quo(poolTo.Sub(result)).Sub(poolFrom).TruncateInt()
}

type FlatPrice struct{}

func (fp FlatPrice) Forward(_, _, offer math.LegacyDec) math.Int {
	return offer.RoundInt()
}

func (fp FlatPrice) Backward(_, _, result math.LegacyDec) math.Int {
	return result.RoundInt()
}

func (k Keeper) ConstantProductTrade(ctx context.Context, denomFrom, denomTo string, offer math.LegacyDec) math.LegacyDec {
	poolFrom, poolTo := k.GetFullLiquidityBaseOther(ctx, denomFrom, denomTo)
	return ConstantProductTrade(poolFrom, poolTo, offer)
}

func ConstantProductTrade(poolFrom, poolTo, offer math.LegacyDec) math.LegacyDec {
	constantProduct := poolFrom.Mul(poolTo)
	return poolTo.Sub(constantProduct.Quo(poolFrom.Add(offer)))
}
