package keeper

import (
	"context"
	"fmt"

	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) QueryEpochCountdown(ctx context.Context, _ *types.QueryEpochCountdownRequest) (*types.QueryEpochCountdownResponse, error) {
	return &types.QueryEpochCountdownResponse{
		Seconds: fmt.Sprintf("%.4f", k.getEpochSecondsLeft(ctx)),
	}, nil
}

func (k Keeper) QueryEpochPositions(ctx context.Context, _ *types.QueryEpochPositionsRequest) (*types.QueryEpochPositionsResponse, error) {
	epochAddresses := []types.AddressEpochPositions{}
	addresses := k.epochShares.OuterKeys()

	for _, address := range addresses {
		epochAddress := types.AddressEpochPositions{}
		epochAddress.Address = address

		iterator := k.epochShares.Iterator(ctx, nil, address)
		for iterator.Valid() {
			keyValue := iterator.GetNextKeyValue()

			epochAddress.Positions = append(epochAddress.Positions, types.EpochShares{
				Shares: keyValue.Value().Value().Shares,
			})
		}

		epochAddresses = append(epochAddresses, epochAddress)
	}

	return &types.QueryEpochPositionsResponse{
		AddressEpochShares: epochAddresses,
	}, nil
}
