package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) UpdateFeeAmount(ctx context.Context, msg *types.MsgUpdateFeeAmount) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	feeAmount, ok := math.NewIntFromString(msg.FeeAmount)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %v", msg.FeeAmount)
	}

	params := k.GetParams(ctx)
	params.CreationFee = feeAmount

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateMinimumUnlock(ctx context.Context, msg *types.MsgUpdateMinimumUnlock) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)
	params.MinimumUnlockInSeconds = max(msg.MinimumUnlock, types.MinimumUnlockingInSeconds)

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateReserveFeeShare(ctx context.Context, msg *types.MsgUpdateReserveFeeShare) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	reserveFeeShare, err := math.LegacyNewDecFromStr(msg.ReserveFeeShare)
	if err != nil {
		return nil, fmt.Errorf("invalid reserve fee share: %w", err)
	}

	params := k.GetParams(ctx)
	params.ReserveFeeShare = reserveFeeShare

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, err
}

func (k msgServer) UpdateMinimumPoolSize(ctx context.Context, msg *types.MsgUpdateMinimumPoolSize) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	minimumPoolSize, ok := math.NewIntFromString(msg.MinimumPoolSize)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %v", msg.MinimumPoolSize)
	}

	params := k.GetParams(ctx)
	params.MinimumPoolSize = minimumPoolSize

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateTradeFeeLimits(ctx context.Context, msg *types.MsgUpdateTradeFeeLimits) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	minimumPoolFee, err := math.LegacyNewDecFromStr(msg.MinimumPoolFee)
	if err != nil {
		return nil, fmt.Errorf("invalid minimum pool fee: %v", msg.MinimumPoolFee)
	}

	maximumPoolFee, err := math.LegacyNewDecFromStr(msg.MaximumPoolFee)
	if err != nil {
		return nil, fmt.Errorf("invalid maximum pool fee: %v", msg.MaximumPoolFee)
	}

	params := k.GetParams(ctx)
	params.MinimumPoolFee = minimumPoolFee
	params.MaximumPoolFee = maximumPoolFee

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateMaximumVestingUnlockSteps(ctx context.Context, msg *types.MsgUpdateMaximumVestingUnlockSteps) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)
	params.MaximumVestingUnlockSteps = msg.MaximumVestingSteps

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateChangeSecondsDescription(ctx context.Context, msg *types.MsgUpdateChangeSecondsDescription) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)
	params.ChangeSecondsDescription = max(msg.ChangeSecondsDescription, 0)

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateChangeSecondsWebsite(ctx context.Context, msg *types.MsgUpdateChangeSecondsWebsite) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)
	params.ChangeSecondsWebsite = max(msg.ChangeSecondsWebsite, 0)

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateChangeSecondsImage(ctx context.Context, msg *types.MsgUpdateChangeSecondsImage) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)
	params.ChangeSecondsImage = max(msg.ChangeSecondsImage, 0)

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) AddTokenCategory(ctx context.Context, msg *types.MsgAddTokenCategory) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	cost, ok := math.NewIntFromString(msg.Cost)
	if !ok {
		return nil, fmt.Errorf("invalid cost: %v", msg.Cost)
	}

	params := k.GetParams(ctx)
	params.Categories.Categories = append(params.Categories.Categories, types.Category{
		Index:         msg.CategoryIndex,
		Name:          msg.Name,
		CreationPrice: cost,
		IsIbc:         msg.IsIbc,
	})

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateTokenCategory(ctx context.Context, msg *types.MsgUpdateTokenCategory) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	cost, ok := math.NewIntFromString(msg.Cost)
	if !ok {
		return nil, fmt.Errorf("invalid cost: %v", msg.Cost)
	}

	params := k.GetParams(ctx)

	var (
		newCategories []types.Category
		seen          bool
	)

	for _, category := range params.Categories.Categories {
		if category.Index == msg.CategoryIndex {
			category = types.Category{
				Index:         category.Index,
				Name:          msg.Name,
				CreationPrice: cost,
			}

			seen = true
		}

		newCategories = append(newCategories, category)
	}

	if !seen {
		return nil, fmt.Errorf("no category found for given index: %v", msg.CategoryIndex)
	}

	params.Categories.Categories = newCategories
	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) MoveLiquidityPool(ctx context.Context, msg *types.MsgMoveLiquidityPool) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExist
	}

	if err := k.MoveDenom(ctx, factoryDenom); err != nil {
		return nil, fmt.Errorf("move liquidity: %w", err)
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdatePoolThresholdSeconds(ctx context.Context, msg *types.MsgUpdatePoolThresholdSeconds) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)
	params.PoolTresholdSeconds = max(msg.PoolThresholdSeconds, 0)

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) UpdateOfferFee(ctx context.Context, msg *types.MsgUpdateOfferFee) (*types.Void, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	offerFee, err := math.LegacyNewDecFromStr(msg.OfferFee)
	if err != nil {
		return nil, fmt.Errorf("invalid offer fee: %w", err)
	}

	params := k.GetParams(ctx)
	params.OfferFee = offerFee

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}
