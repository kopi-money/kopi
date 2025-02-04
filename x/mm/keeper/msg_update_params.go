package keeper

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k msgServer) UpdateProtocolShare(ctx context.Context, req *types.MsgUpdateProtocolShare) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	protocolShare, err := math.LegacyNewDecFromStr(req.ProtocolShare)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.ProtocolShare = protocolShare

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, err
}

func (k msgServer) UpdateRedemptionFees(ctx context.Context, req *types.MsgUpdateRedemptionFees) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	minRedemptionFee, err := math.LegacyNewDecFromStr(req.MinRedemptionFee)
	if err != nil {
		return nil, err
	}

	maxRedemptionFee, err := math.LegacyNewDecFromStr(req.MinRedemptionFee)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.MinRedemptionFee = minRedemptionFee
	params.MaxRedemptionFee = maxRedemptionFee

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, err
}

func (k msgServer) UpdateCollateralDiscount(ctx context.Context, req *types.MsgUpdateCollateralDiscount) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	collateralDiscount, err := math.LegacyNewDecFromStr(req.CollateralDiscount)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.CollateralDiscount = collateralDiscount

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, err
}

func (k msgServer) UpdateInterestRateParameters(ctx context.Context, req *types.MsgUpdateInterestRateParameters) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	minInterestRate, err := math.LegacyNewDecFromStr(req.MinInterestRate)
	if err != nil {
		return nil, err
	}

	a, err := math.LegacyNewDecFromStr(req.A)
	if err != nil {
		return nil, err
	}

	b, err := math.LegacyNewDecFromStr(req.B)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.MinInterestRate = minInterestRate
	params.A = a
	params.B = b

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, err
}

func (k msgServer) DelistCollateralDenom(ctx context.Context, req *types.MsgDelistCollateralDenom) (*types.Void, error) {
	if err := k.Keeper.DelistCollateralDenom(ctx, req.Denom); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k Keeper) DelistCollateralDenom(ctx context.Context, denom string) error {
	if !k.DenomKeeper.IsValidCollateralDenom(ctx, denom) {
		return types.ErrInvalidCollateralDenom
	}

	ltv, err := k.DenomKeeper.GetLTV(ctx, denom)
	if err != nil {
		return err
	}

	if !ltv.IsZero() {
		return fmt.Errorf("can only delist collateral when ltv is zero")
	}

	iterator := k.collateral.Iterator(ctx, nil, denom)
	for iterator.Valid() {
		collateral := iterator.GetNext()

		address, _ := sdk.AccAddressFromBech32(collateral.Address)
		if _, err = k.WithdrawCollateral(ctx, address, denom, collateral.Amount); err != nil {
			return fmt.Errorf("withdraw collateral: %w", err)
		}
	}

	return k.DenomKeeper.RemoveCollateralDenom(ctx, denom)
}
