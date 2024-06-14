package keeper

import (
	"context"
	"cosmossdk.io/collections"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/mm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) FullBalance(ctx context.Context, req *types.QueryFullBalanceRequest) (*types.QueryFullBalanceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	orders := k.DexKeeper.GetAllOrdersByAddress(ctx, req.Address)
	addr, _ := sdk.AccAddressFromBech32(req.Address)
	coins := k.BankKeeper.SpendableCoins(ctx, addr)

	sumSumUSD := math.LegacyZeroDec()
	sumLiqUSD := math.LegacyZeroDec()
	sumOrdUSD := math.LegacyZeroDec()
	sumWalUSD := math.LegacyZeroDec()
	sumColUSD := math.LegacyZeroDec()

	entries := []*types.FullDenomBalance{}
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		liq := k.DexKeeper.GetLiquidityByAddress(ctx, denom, req.Address)
		ord := getOrderValueByDenom(orders, denom)
		wal := coins.AmountOf(denom)
		col := k.getProvidedCollateral(ctx, req.Address, denom)
		sum := liq.Add(ord).Add(wal).Add(col)

		liqUSD, _ := k.DexKeeper.GetValueInUSD(ctx, denom, liq.ToLegacyDec())
		ordUSD, _ := k.DexKeeper.GetValueInUSD(ctx, denom, ord.ToLegacyDec())
		walUSD, _ := k.DexKeeper.GetValueInUSD(ctx, denom, wal.ToLegacyDec())
		colUSD, _ := k.DexKeeper.GetValueInUSD(ctx, denom, col.ToLegacyDec())
		sumUSD, _ := k.DexKeeper.GetValueInUSD(ctx, denom, sum.ToLegacyDec())

		sumSumUSD = sumSumUSD.Add(sumUSD)
		sumLiqUSD = sumLiqUSD.Add(liqUSD)
		sumOrdUSD = sumOrdUSD.Add(ordUSD)
		sumWalUSD = sumWalUSD.Add(walUSD)
		sumColUSD = sumColUSD.Add(colUSD)

		entries = append(entries, &types.FullDenomBalance{
			Denom:         denom,
			Sum:           sum.String(),
			SumUsd:        sumUSD.String(),
			Wallet:        wal.String(),
			WalletUsd:     walUSD.String(),
			Liquidity:     liq.String(),
			LiquidityUsd:  liqUSD.String(),
			Orders:        ord.String(),
			OrdersUsd:     ordUSD.String(),
			Collateral:    col.String(),
			CollateralUsd: colUSD.String(),
		})
	}

	return &types.QueryFullBalanceResponse{
		Sum:           sumSumUSD.String(),
		SumWallet:     sumWalUSD.String(),
		SumLiquidity:  sumLiqUSD.String(),
		SumOrders:     sumOrdUSD.String(),
		SumCollateral: sumColUSD.String(),
		Denoms:        entries,
	}, nil
}

func (k Keeper) getProvidedCollateral(ctx context.Context, address, denom string) math.Int {
	collateral, found := k.collateral.Get(ctx, collections.Join(denom, address))
	if !found {
		return math.ZeroInt()
	}

	return collateral.Amount
}

func getOrderValueByDenom(orders []dextypes.Order, denom string) math.Int {
	sum := math.ZeroInt()

	for _, order := range orders {
		if order.DenomFrom == denom {
			sum = sum.Add(order.AmountLeft)
		}
	}

	return sum
}
