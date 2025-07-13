package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	reservetypes "github.com/kopi-money/kopi/x/reserve/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) CreateOffers(ctx context.Context, msg *types.MsgCreateOffers) (*types.Void, error) {
	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExist
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	amountFactory, ok := math.NewIntFromString(msg.FactoryDenomAmount)
	if !ok {
		return nil, fmt.Errorf("invalid factory denom amount format: %s", msg.FactoryDenomAmount)
	}
	
	if !amountFactory.IsPositive() {
		return nil, fmt.Errorf("factory denom amount must be positive, got: %s", msg.FactoryDenomAmount)
	}

	if !k.DenomKeeper.IsValidDenom(ctx, msg.AskDenom) {
		return nil, types.ErrInvalidAskDenom
	}

	askAmount, ok := math.NewIntFromString(msg.AskAmount)
	if !ok {
		return nil, fmt.Errorf("invalid ask amount format: %s", msg.AskAmount)
	}

	if !askAmount.IsPositive() {
		return nil, fmt.Errorf("ask amount must be positive, got: %s", msg.AskAmount)
	}

	if !k.DenomKeeper.IsFactoryPoolDenom(ctx, msg.AskDenom) {
		return nil, types.ErrNoValidPoolDenom
	}

	if msg.VestedUntil != nil {
		if msg.VestedUntil.Before(msg.ValidUntil) {
			return nil, types.ErrOfferInvalidVestingEnd
		}

		if msg.NumUnlockSteps < 1 {
			return nil, types.ErrVestingNegativeSteps
		}

		if msg.NumUnlockSteps > k.getMaximumVestingUnlockSteps(ctx) {
			return nil, types.ErrVestingTooManySteps
		}
	}

	if len(msg.Receivers) == 0 {
		return nil, types.ErrEmptyOfferReceiversList
	}

	for _, receiver := range msg.Receivers {
		if _, err = sdk.AccAddressFromBech32(receiver); err != nil {
			return nil, fmt.Errorf("invalid user address: %w", err)
		}

		coins := sdk.NewCoins(sdk.NewCoin(factoryDenom.FullName, amountFactory))
		if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolOffers, coins); err != nil {
			return nil, err
		}

		k.SetOffer(ctx, types.Offer{
			FactoryDenom:       factoryDenom.FullName,
			AddressReceiver:    receiver,
			FactoryDenomAmount: amountFactory,
			AskDenom:           msg.AskDenom,
			AskAmount:          askAmount,
			VestedUntil:        msg.VestedUntil,
			ValidUntil:         msg.ValidUntil,
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
			return nil, types.ErrDenomDoesNotExist
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
		return nil, types.ErrDenomDoesNotExist
	}

	accUser, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	accAdmin, err := sdk.AccAddressFromBech32(factoryDenom.Admin)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	askAmount, err := k.handleOfferFee(ctx, offer.AskDenom, offer.AskAmount)
	if err != nil {
		return nil, err
	}

	coins := sdk.NewCoins(sdk.NewCoin(offer.AskDenom, askAmount))
	if err = k.BankKeeper.SendCoins(ctx, accUser, accAdmin, coins); err != nil {
		return nil, err
	}

	k.RemoveOffer(ctx, offer.Index)

	if offer.VestedUntil != nil {
		offerPoolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolOffers)
		offerPoolAddress := offerPoolAcc.GetAddress().String()

		if err = k.createVesting(ctx, offerPoolAddress, offer.AddressReceiver, factoryDenom.FullName, offer.FactoryDenomAmount, offer.CreatedAt, *offer.VestedUntil, offer.NumUnlocksSteps); err != nil {
			return nil, fmt.Errorf("create vesting: %w", err)
		}
	} else {
		coins = sdk.NewCoins(sdk.NewCoin(factoryDenom.FullName, offer.FactoryDenomAmount))
		if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolOffers, accUser, coins); err != nil {
			return nil, err
		}
	}

	return &types.Void{}, nil
}

func (k Keeper) handleOfferFee(ctx context.Context, askDenom string, askAmount math.Int) (math.Int, error) {
	offerFee := k.getOfferFee(ctx)
	if offerFee.IsZero() {
		return askAmount, nil
	}

	feeAmount := askAmount.ToLegacyDec().Mul(offerFee).TruncateInt()

	coins := sdk.NewCoins(sdk.NewCoin(askDenom, feeAmount))
	if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolFactoryLiquidity, reservetypes.BuyingKCoins, coins); err != nil {
		return math.Int{}, fmt.Errorf("send offer fee to module: %w", err)
	}

	askAmount = askAmount.Sub(feeAmount)
	return askAmount, nil
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
		return nil, types.ErrDenomDoesNotExist
	}

	accAdmin, err := sdk.AccAddressFromBech32(factoryDenom.Admin)
	if err != nil {
		return nil, fmt.Errorf("invalid user address: %w", err)
	}

	coins := sdk.NewCoins(sdk.NewCoin(offer.FactoryDenom, offer.FactoryDenomAmount))
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolOffers, accAdmin, coins); err != nil {
		return nil, err
	}

	k.RemoveOffer(ctx, offer.Index)

	return &types.Void{}, nil
}
