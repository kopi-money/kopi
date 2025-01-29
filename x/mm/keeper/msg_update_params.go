package keeper

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/cache"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k msgServer) UpdateProtocolShare(ctx context.Context, req *types.MsgUpdateProtocolShare) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		protocolShare, err := math.LegacyNewDecFromStr(req.ProtocolShare)
		if err != nil {
			return err
		}

		params := k.GetParams(innerCtx)
		params.ProtocolShare = protocolShare

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateRedemptionFees(ctx context.Context, req *types.MsgUpdateRedemptionFees) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		minRedemptionFee, err := math.LegacyNewDecFromStr(req.MinRedemptionFee)
		if err != nil {
			return err
		}

		maxRedemptionFee, err := math.LegacyNewDecFromStr(req.MinRedemptionFee)
		if err != nil {
			return err
		}

		params := k.GetParams(innerCtx)
		params.MinRedemptionFee = minRedemptionFee
		params.MaxRedemptionFee = maxRedemptionFee

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateCollateralDiscount(ctx context.Context, req *types.MsgUpdateCollateralDiscount) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		collateralDiscount, err := math.LegacyNewDecFromStr(req.CollateralDiscount)
		if err != nil {
			return err
		}

		params := k.GetParams(innerCtx)
		params.CollateralDiscount = collateralDiscount

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) UpdateInterestRateParameters(ctx context.Context, req *types.MsgUpdateInterestRateParameters) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		minInterestRate, err := math.LegacyNewDecFromStr(req.MinInterestRate)
		if err != nil {
			return err
		}

		a, err := math.LegacyNewDecFromStr(req.A)
		if err != nil {
			return err
		}

		b, err := math.LegacyNewDecFromStr(req.B)
		if err != nil {
			return err
		}

		params := k.GetParams(innerCtx)
		params.MinInterestRate = minInterestRate
		params.A = a
		params.B = b

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.Void{}, err
}

func (k msgServer) DelistCollateralDenom(ctx context.Context, req *types.MsgDelistCollateralDenom) (*types.Void, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		return k.Keeper.DelistCollateralDenom(ctx, req.Denom)
	})

	return &types.Void{}, err
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
