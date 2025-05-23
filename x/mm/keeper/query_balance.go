package keeper

import (
	"context"
	"fmt"

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

	address, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("get reference denom: %w", err)
	}

	var (
		orders  = k.DexKeeper.GetAllOrdersByAddress(ctx, req.Address)
		coins   = k.BankKeeper.SpendableCoins(ctx, address)
		entries = []types.FullDenomBalance{}
	)

	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		liq := k.DexKeeper.GetLiquidityByAddress(ctx, denom, req.Address)
		ord := getOrderValueByDenom(orders, denom)
		wal := coins.AmountOf(denom)
		col := k.getProvidedCollateral(ctx, req.Address, denom)
		sum := liq.Add(ord).Add(wal).Add(col)

		liqUSD, _ := k.DenomKeeper.GetValueIn(ctx, denom, referenceDenom, liq.ToLegacyDec())
		ordUSD, _ := k.DenomKeeper.GetValueIn(ctx, denom, referenceDenom, ord.ToLegacyDec())
		walUSD, _ := k.DenomKeeper.GetValueIn(ctx, denom, referenceDenom, wal.ToLegacyDec())
		colUSD, _ := k.DenomKeeper.GetValueIn(ctx, denom, referenceDenom, col.ToLegacyDec())
		sumUSD, _ := k.DenomKeeper.GetValueIn(ctx, denom, referenceDenom, sum.ToLegacyDec())

		entries = append(entries, types.FullDenomBalance{
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
		Denoms: entries,
	}, nil
}

func (k Keeper) getProvidedCollateral(ctx context.Context, address, denom string) math.Int {
	collateral, found := k.collateral.Get(ctx, denom, address)
	if !found {
		return math.ZeroInt()
	}

	return collateral.Amount
}

func getOrderValueByDenom(orders []dextypes.Order, denom string) math.Int {
	sum := math.ZeroInt()

	for _, order := range orders {
		if order.DenomGiving == denom {
			sum = sum.Add(order.AmountLocked)
		}
	}

	return sum
}
