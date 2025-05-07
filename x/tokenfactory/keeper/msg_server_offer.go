package keeper

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) CreateOffers(ctx context.Context, msg *types.MsgCreateOffers) (*types.Void, error) {
	acc, _ := sdk.AccAddressFromBech32(msg.Creator)
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExists
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	amountFactory, ok := math.NewIntFromString(msg.FactoryDenomAmount)
	if !ok {
		return nil, types.ErrInvalidAmountFormat
	}

	if !k.DenomKeeper.IsValidDenom(ctx, msg.AskDenom) {
		return nil, types.ErrInvalidAskDenom
	}

	askAmount, ok := math.NewIntFromString(msg.AskAmount)
	if !ok {
		return nil, types.ErrInvalidAmountFormat
	}

	validUntil := time.UnixMilli(msg.ValidUntil)

	var vestedUntil *time.Time
	if msg.VestedUntil > 0 {
		vestedUntil_ := time.UnixMilli(msg.VestedUntil)
		if vestedUntil_.Before(validUntil) {
			return nil, types.ErrOfferInvalidVestingEnd
		}

		vestedUntil = &vestedUntil_

		if msg.NumUnlockSteps < 1 {
			return nil, types.ErrVestingNegativeSteps
		}

		if msg.NumUnlockSteps > k.getMaximumVestingUnlockSteps(ctx) {
			return nil, types.ErrVestingTooManySteps
		}
	}

	for _, receiver := range msg.Receivers {
		if _, err := sdk.AccAddressFromBech32(receiver); err != nil {
			return nil, types.ErrInvalidAddress
		}

		coins := sdk.NewCoins(sdk.NewCoin(factoryDenom.FullName, amountFactory))
		if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolOffers, coins); err != nil {
			return nil, err
		}

		k.SetOffer(ctx, types.Offer{
			FactoryDenom:       factoryDenom.FullName,
			AddressReceiver:    receiver,
			FactoryDenomAmount: amountFactory,
			AskDenom:           msg.AskDenom,
			AskAmount:          askAmount,
			VestedUntil:        vestedUntil,
			NumUnlocksSteps:    msg.NumUnlockSteps,
			CreatedAt:          sdk.UnwrapSDKContext(ctx).BlockTime(),
		})
	}

	return &types.Void{}, nil
}

func (k msgServer) CancelOffers(ctx context.Context, msg *types.MsgCancelOffers) (*types.Void, error) {
	for _, index := range msg.OfferIndexes {
		offer, has := k.GetOffer(ctx, index)
		if !has {
			return nil, types.ErrOfferNotFound
		}

		factoryDenom, has := k.GetDenomByFullName(ctx, offer.FactoryDenom)
		if !has {
			return nil, types.ErrDenomDoesNotExists
		}

		if msg.Creator != factoryDenom.Admin {
			return nil, types.ErrIncorrectAdmin
		}

		if err := k.cancelOffer(ctx, offer.Index); err != nil {
			return nil, err
		}
	}

	return &types.Void{}, nil
}

func (k msgServer) TakeOffer(ctx context.Context, msg *types.MsgTakeOffer) (*types.Void, error) {
	offer, has := k.GetOffer(ctx, msg.OfferIndex)
	if !has {
		return nil, types.ErrOfferNotFound
	}

	if msg.Creator != offer.AddressReceiver {
		return nil, types.ErrOfferWrongTaker
	}

	factoryDenom, has := k.GetDenomByFullName(ctx, offer.FactoryDenom)
	if !has {
		return nil, types.ErrDenomDoesNotExists
	}

	accUser, _ := sdk.AccAddressFromBech32(msg.Creator)
	accAdmin, _ := sdk.AccAddressFromBech32(factoryDenom.Admin)

	coins := sdk.NewCoins(sdk.NewCoin(offer.AskDenom, offer.AskAmount))
	if err := k.BankKeeper.SendCoins(ctx, accUser, accAdmin, coins); err != nil {
		return nil, err
	}

	k.RemoveOffer(ctx, offer.Index)

	if offer.VestedUntil != nil {
		offerPoolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolOffers)
		offerPoolAddress := offerPoolAcc.GetAddress().String()

		if err := k.createVesting(ctx, offerPoolAddress, offer.AddressReceiver, factoryDenom.FullName, offer.FactoryDenomAmount, offer.CreatedAt, *offer.VestedUntil, offer.NumUnlocksSteps); err != nil {
			return nil, fmt.Errorf("create vesting: %w", err)
		}
	} else {
		coins = sdk.NewCoins(sdk.NewCoin(factoryDenom.FullName, offer.FactoryDenomAmount))
		if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolOffers, accUser, coins); err != nil {
			return nil, err
		}
	}

	return &types.Void{}, nil
}

func (k msgServer) DeclineOffer(ctx context.Context, msg *types.MsgDeclineOffer) (*types.Void, error) {
	offer, has := k.GetOffer(ctx, msg.OfferIndex)
	if !has {
		return nil, types.ErrOfferNotFound
	}

	if msg.Creator != offer.AddressReceiver {
		return nil, types.ErrOfferWrongTaker
	}

	factoryDenom, has := k.GetDenomByFullName(ctx, offer.FactoryDenom)
	if !has {
		return nil, types.ErrDenomDoesNotExists
	}

	accAdmin, _ := sdk.AccAddressFromBech32(factoryDenom.Admin)
	coins := sdk.NewCoins(sdk.NewCoin(offer.FactoryDenom, offer.FactoryDenomAmount))
	if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolOffers, accAdmin, coins); err != nil {
		return nil, err
	}

	k.RemoveOffer(ctx, offer.Index)

	return &types.Void{}, nil
}
