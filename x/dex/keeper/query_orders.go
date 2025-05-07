package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/kopi-money/kopi/x/dex/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Order(ctx context.Context, req *types.QueryOrderRequest) (*types.QueryOrderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	order, ok := k.GetOrder(ctx, req.Index)
	if !ok {
		return nil, types.ErrOrderNotFound
	}

	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("get reference denom: %w", err)
	}

	feeFac := k.GetJoinedFee(ctx)
	feeFac = feeFac.Add(math.LegacyOneDec())

	orderResponse, err := k.toOrderResponse(ctx, order, feeFac, referenceDenom)
	if err != nil {
		return nil, err
	}

	return &types.QueryOrderResponse{
		Order: orderResponse,
	}, nil
}

func (k Keeper) Orders(ctx context.Context, req *types.QueryOrdersRequest) (*types.QueryOrdersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("get reference denom: %w", err)
	}

	feeFac := k.GetJoinedFee(ctx)
	feeFac = feeFac.Add(math.LegacyOneDec())

	orders, pageRes, err := query.CollectionPaginate(
		ctx,
		k.orders,
		req.Pagination,
		func(_ uint64, order types.Order) (types.OrderResponse, error) {
			return k.toOrderResponse(ctx, order, feeFac, referenceDenom)
		},
	)

	if err != nil {
		return nil, fmt.Errorf("get orders from pagination: %w", err)
	}

	return &types.QueryOrdersResponse{
		Orders:     orders,
		Pagination: pageRes,
	}, nil
}

func (k Keeper) OrdersNum(_ context.Context, _ *types.QueryOrdersNumRequest) (*types.QueryOrdersNumResponse, error) {
	return &types.QueryOrdersNumResponse{Num: int64(k.GetAllOrdersNum())}, nil
}

func (k Keeper) OrdersAddress(ctx context.Context, req *types.QueryOrdersAddressRequest) (*types.QueryOrdersAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("get reference denom: %w", err)
	}

	feeFac := k.GetJoinedFee(ctx)
	feeFac = feeFac.Add(math.LegacyOneDec())

	orders, pageRes, err := query.CollectionFilteredPaginate(
		ctx,
		k.orders,
		req.Pagination,
		func(_ uint64, order types.Order) (include bool, err error) {
			if order.Creator != req.Address {
				return false, nil
			}

			if req.DenomGiving != "" && order.DenomGiving != req.DenomGiving {
				return false, nil
			}

			if req.DenomReceiving != "" && order.DenomReceiving != req.DenomReceiving {
				return false, nil
			}

			return true, nil
		},
		func(_ uint64, order types.Order) (types.OrderResponse, error) {
			return k.toOrderResponse(ctx, order, feeFac, referenceDenom)
		},
	)

	return &types.QueryOrdersAddressResponse{
		Orders:     orders,
		Pagination: pageRes,
	}, nil
}

func (k Keeper) toOrderResponse(ctx context.Context, order types.Order, feeFac math.LegacyDec, referenceDenom string) (types.OrderResponse, error) {
	amountLeftUSD, err := k.DenomKeeper.GetValueIn(ctx, order.DenomGiving, referenceDenom, order.AmountLeft.ToLegacyDec())
	if err != nil {
		return types.OrderResponse{}, fmt.Errorf("amount left in usd: %w", err)
	}

	amountReceivedUSD, err := k.DenomKeeper.GetValueIn(ctx, order.DenomReceiving, referenceDenom, order.AmountReceived.ToLegacyDec())
	if err != nil {
		return types.OrderResponse{}, fmt.Errorf("amount received in usd: %w", err)
	}

	maxPriceUSD, err := k.DenomKeeper.GetValueIn(ctx, order.DenomReceiving, referenceDenom, order.MaxPrice)
	if err != nil {
		return types.OrderResponse{}, fmt.Errorf("max price usd: %w", err)
	}

	currentPrice, err := k.DenomKeeper.CalculatePrice(ctx, order.DenomGiving, order.DenomReceiving)
	if err != nil {
		return types.OrderResponse{}, fmt.Errorf("calculate price: %w", err)
	}

	if order.IsBuyOrder {
		currentPrice = currentPrice.Mul(feeFac)
	} else {
		if !currentPrice.IsPositive() {
			return types.OrderResponse{}, fmt.Errorf("current price is not positive")
		}

		if !feeFac.IsPositive() {
			return types.OrderResponse{}, fmt.Errorf("feefac is not positive")
		}

		currentPrice = math.LegacyOneDec().Quo(currentPrice) // C
		currentPrice = currentPrice.Quo(feeFac)              // C
	}

	currentPriceUSD, err := k.DenomKeeper.GetValueIn(ctx, order.DenomReceiving, referenceDenom, currentPrice)
	if err != nil {
		return types.OrderResponse{}, fmt.Errorf("max price usd: %w", err)
	}

	return types.OrderResponse{
		Index:             order.Index,
		Address:           order.Creator,
		DenomGiving:       order.DenomGiving,
		DenomReceiving:    order.DenomReceiving,
		TradeAmount:       order.TradeAmount.String(),
		AmountLocked:      order.AmountLocked.String(),
		AmountLeft:        order.AmountLeft.String(),
		AmountLeftUsd:     amountLeftUSD.String(),
		AmountGiven:       order.AmountGiven.String(),
		AmountReceived:    order.AmountReceived.String(),
		AmountReceivedUsd: amountReceivedUSD.String(),
		CurrentPrice:      currentPrice.String(),
		CurrentPriceUsd:   currentPriceUSD.String(),
		MaxPrice:          order.MaxPrice.String(),
		MaxPriceUsd:       maxPriceUSD.String(),
		AddedAt:           uint64(order.AddedAt),
		NumBlocks:         order.NumBlocks,
		BlockEnd:          uint64(order.AddedAt) + order.NumBlocks,
		AllowIncomplete:   order.AllowIncomplete,
		IsBuyOrder:        order.IsBuyOrder,
	}, nil
}

func (k Keeper) OrdersByPair(ctx context.Context, req *types.OrdersByPairRequest) (*types.QueryOrdersByPairResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("get reference denom: %w", err)
	}

	feeFac := k.GetJoinedFee(ctx)
	feeFac = feeFac.Add(math.LegacyOneDec())

	var asks, bids []types.OrderResponse

	iterator := k.OrderIterator(ctx)
	for iterator.Valid() {
		order := iterator.GetNext()

		if order.DenomGiving == req.DenomGiving && order.DenomReceiving == req.DenomReceiving {
			var orderResponse types.OrderResponse
			orderResponse, err = k.toOrderResponse(ctx, order, feeFac, referenceDenom)
			if err != nil {
				return nil, err
			}

			bids = append(bids, orderResponse)
		}

		if order.DenomGiving == req.DenomReceiving && order.DenomReceiving == req.DenomGiving {
			var orderResponse types.OrderResponse
			orderResponse, err = k.toOrderResponse(ctx, order, feeFac, referenceDenom)
			if err != nil {
				return nil, err
			}

			asks = append(asks, orderResponse)
		}
	}

	return &types.QueryOrdersByPairResponse{
		Bids: bids,
		Asks: asks,
	}, nil
}
