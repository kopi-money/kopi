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

func (k msgServer) AddKCoin(goCtx context.Context, req *types.MsgAddKCoin) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)

		maxSupply, ok := math.NewIntFromString(req.MaxSupply)
		if !ok {
			return fmt.Errorf("invalid max supply value: %v", req.MaxSupply)
		}

		maxBurnAmount, ok := math.NewIntFromString(req.MaxBurnAmount)
		if !ok {
			return fmt.Errorf("invalid max burn amount value: %v", req.MaxBurnAmount)
		}

		maxMintAmount, ok := math.NewIntFromString(req.MaxMintAmount)
		if !ok {
			return fmt.Errorf("invalid max mint amount value: %v", req.MaxMintAmount)
		}

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

		kCoin := types.KCoin{
			Denom:         req.Denom,
			References:    req.References,
			MaxSupply:     maxSupply,
			MaxMintAmount: maxMintAmount,
			MaxBurnAmount: maxBurnAmount,
		}

		dexDenom := types.DexDenom{
			Name:         req.Denom,
			Factor:       &factor,
			MinLiquidity: minLiquidity,
			MinOrderSize: minOrderSize,
		}

		params.KCoins = append(params.KCoins, &kCoin)
		params.DexDenoms = append(params.DexDenoms, &dexDenom)

		if err = k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) UpdateKCoinSupply(goCtx context.Context, req *types.MsgUpdateKCoinSupply) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)

		maxSupply, ok := math.NewIntFromString(req.MaxSupply)
		if !ok {
			return fmt.Errorf("invalid max supply value: %v", req.MaxSupply)
		}

		kCoins := []*types.KCoin{}
		found := false

		for _, kCoin := range params.KCoins {
			if kCoin.Denom == req.Denom {
				kCoin.MaxSupply = maxSupply
				found = true
			}

			kCoins = append(kCoins, kCoin)
		}

		if !found {
			return types.ErrInvalidKCoin
		}

		params.KCoins = kCoins

		if err := k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) UpdateKCoinMintAmount(goCtx context.Context, req *types.MsgUpdateKCoinMintAmount) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)

		maxMintAmount, ok := math.NewIntFromString(req.MaxMintAmount)
		if !ok {
			return fmt.Errorf("invalid max mint amount value: %v", req.MaxMintAmount)
		}

		kCoins := []*types.KCoin{}
		found := false

		for _, kCoin := range params.KCoins {
			if kCoin.Denom == req.Denom {
				kCoin.MaxMintAmount = maxMintAmount
				found = true
			}

			kCoins = append(kCoins, kCoin)
		}

		if !found {
			return types.ErrInvalidKCoin
		}

		params.KCoins = kCoins

		if err := k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) UpdateKCoinBurnAmount(goCtx context.Context, req *types.MsgUpdateKCoinBurnAmount) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)

		maxBurnAmount, ok := math.NewIntFromString(req.MaxBurnAmount)
		if !ok {
			return fmt.Errorf("invalid max burn amount value: %v", req.MaxBurnAmount)
		}

		kCoins := []*types.KCoin{}
		found := false

		for _, kCoin := range params.KCoins {
			if kCoin.Denom == req.Denom {
				kCoin.MaxBurnAmount = maxBurnAmount
				found = true
			}

			kCoins = append(kCoins, kCoin)
		}

		if !found {
			return types.ErrInvalidKCoin
		}

		params.KCoins = kCoins

		if err := k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) AddKCoinReferences(goCtx context.Context, req *types.MsgAddKCoinReferences) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)

		kCoins := []*types.KCoin{}
		found := false

		for _, kCoin := range params.KCoins {
			if kCoin.Denom == req.Denom {
				kCoin.References = append(kCoin.References, req.References...)
				found = true
			}

			kCoins = append(kCoins, kCoin)
		}

		if !found {
			return types.ErrInvalidKCoin
		}

		params.KCoins = kCoins

		if err := k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) RemoveKCoinReferences(goCtx context.Context, req *types.MsgRemoveKCoinReferences) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(ctx)

		kCoins := []*types.KCoin{}
		found := false

		for _, kCoin := range params.KCoins {
			if kCoin.Denom == req.Denom {
				kCoin.References = filterReferences(kCoin.References, req.References)
				found = true
			}

			kCoins = append(kCoins, kCoin)
		}

		if !found {
			return types.ErrInvalidKCoin
		}

		params.KCoins = kCoins

		if err := k.SetParams(ctx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func filterReferences(existingReferences, toRemove []string) (filtered []string) {
	for _, existingReference := range existingReferences {
		if !contains(toRemove, existingReference) {
			filtered = append(filtered, existingReference)
		}
	}

	return
}
