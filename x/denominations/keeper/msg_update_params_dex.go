package keeper

import (
	"context"
	"fmt"
	"github.com/kopi-money/kopi/cache"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) AddDEXDenom(goCtx context.Context, req *types.MsgAddDEXDenom) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)

		factor, err := math.LegacyNewDecFromStr(req.Factor)
		if err != nil {
			return err
		}

		minLiquidity, ok := math.NewIntFromString(req.MinLiquidity)
		if !ok {
			return fmt.Errorf("invalid min liquidity value: %v", req.MinLiquidity)
		}

		minOrderSize, ok := math.NewIntFromString(req.MinOrderSize)
		if !ok {
			return fmt.Errorf("invalid min order size: %v", req.MinOrderSize)
		}

		dexDenom := &types.DexDenom{
			Name:         req.Name,
			Factor:       &factor,
			MinLiquidity: minLiquidity,
			MinOrderSize: minOrderSize,
		}

		params.DexDenoms = append(params.DexDenoms, dexDenom)

		if err = k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) UpdateDEXDenomMinimumLiquidity(goCtx context.Context, req *types.MsgUpdateDEXDenomMinimumLiquidity) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)

		minLiquidity, ok := math.NewIntFromString(req.MinLiquidity)
		if !ok {
			return fmt.Errorf("invalid min liquidity value: %v", req.MinLiquidity)
		}

		dexDenoms := []*types.DexDenom{}
		found := false

		for _, dexDenom := range params.DexDenoms {
			if dexDenom.Name == req.Name {
				dexDenom.MinLiquidity = minLiquidity
				found = true
			}

			dexDenoms = append(dexDenoms, dexDenom)
		}

		if !found {
			return types.ErrInvalidDexAsset
		}

		params.DexDenoms = dexDenoms

		if err := k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) UpdateDEXDenomMinimumOrderSize(goCtx context.Context, req *types.MsgUpdateDEXDenomMinimumOrderSize) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)

		minOrderSize, ok := math.NewIntFromString(req.MinOrderSize)
		if !ok {
			return fmt.Errorf("invalid min liquidity value: %v", req.MinOrderSize)
		}

		dexDenoms := []*types.DexDenom{}
		found := false

		for _, dexDenom := range params.DexDenoms {
			if dexDenom.Name == req.Name {
				dexDenom.MinOrderSize = minOrderSize
				found = true
			}

			dexDenoms = append(dexDenoms, dexDenom)
		}

		if !found {
			return types.ErrInvalidDexAsset
		}

		params.DexDenoms = dexDenoms

		if err := k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}
