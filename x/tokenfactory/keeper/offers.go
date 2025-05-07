package keeper

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"time"

	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k Keeper) SetOffer(ctx context.Context, offer types.Offer) {
	nextIndex, _ := k.offersNextIndex.Get(ctx)
	nextIndex += 1
	k.offersNextIndex.Set(ctx, nextIndex)
	offer.Index = nextIndex

	k.offers.Set(ctx, offer.Index, offer)
}

func (k Keeper) GetOffer(ctx context.Context, index uint64) (types.Offer, bool) {
	return k.offers.Get(ctx, index)
}

func (k Keeper) RemoveOffer(ctx context.Context, index uint64) {
	k.offers.Remove(ctx, index)
}

func (k Keeper) HandleOffers(ctx context.Context, blockTime time.Time) error {
	iterator := k.offers.Iterator(ctx, nil)
	for iterator.Valid() {
		keyValue := iterator.GetNextKeyValue()

		offer := keyValue.Value().Value()
		if blockTime.After(offer.ValidUntil) {
			if err := k.cancelOffer(ctx, offer.Index); err != nil {
				return fmt.Errorf("cancelOffer: %w", err)
			}
		}
	}

	return nil
}

func (k Keeper) cancelOffer(ctx context.Context, index uint64) error {
	offer, has := k.GetOffer(ctx, index)
	if !has {
		return types.ErrOfferNotFound
	}

	factoryDenom, has := k.GetDenomByFullName(ctx, offer.FactoryDenom)
	if !has {
		return types.ErrDenomDoesNotExists
	}

	acc, _ := sdk.AccAddressFromBech32(factoryDenom.Admin)
	coins := sdk.NewCoins(sdk.NewCoin(factoryDenom.FullName, offer.FactoryDenomAmount))
	if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolOffers, acc, coins); err != nil {
		return err
	}

	k.RemoveOffer(ctx, offer.Index)

	return nil
}
