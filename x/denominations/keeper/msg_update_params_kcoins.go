package keeper

import (
	"context"

	"github.com/kopi-money/kopi/cache"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) KCoinAddDenom(ctx context.Context, req *types.MsgKCoinAddDenom) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)

		maxSupply, _ := math.NewIntFromString(req.MaxSupply)
		maxBurnAmount, _ := math.NewIntFromString(req.MaxBurnAmount)
		maxMintAmount, _ := math.NewIntFromString(req.MaxMintAmount)

		params.KCoins = append(params.KCoins, types.KCoin{
			DexDenom:      req.Name,
			References:    req.References,
			MaxSupply:     maxSupply,
			MaxMintAmount: maxMintAmount,
			MaxBurnAmount: maxBurnAmount,
		})

		dexDenom, ratio, err := k.createDexDenom(ctx, req.Name, req.Factor, req.MinLiquidity, req.MinOrderSize, req.Exponent)
		if err != nil {
			return err
		}

		params.DexDenoms = append(params.DexDenoms, dexDenom)

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		k.ratios.Set(innerCtx, req.Name, ratio)

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) KCoinUpdateSupplyLimit(ctx context.Context, req *types.MsgKCoinUpdateSupplyLimit) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)
		maxSupply, _ := math.NewIntFromString(req.MaxSupply)
		kCoins := []types.KCoin{}
		found := false

		for _, kCoin := range params.KCoins {
			if kCoin.DexDenom == req.Denom {
				kCoin.MaxSupply = maxSupply
				found = true
			}

			kCoins = append(kCoins, kCoin)
		}

		if !found {
			return types.ErrInvalidKCoin
		}

		params.KCoins = kCoins

		if err := k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) KCoinUpdateMintAmount(ctx context.Context, req *types.MsgKCoinUpdateMintAmount) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)
		maxMintAmount, _ := math.NewIntFromString(req.MaxMintAmount)
		kCoins := []types.KCoin{}
		found := false

		for _, kCoin := range params.KCoins {
			if kCoin.DexDenom == req.Denom {
				kCoin.MaxMintAmount = maxMintAmount
				found = true
			}

			kCoins = append(kCoins, kCoin)
		}

		if !found {
			return types.ErrInvalidKCoin
		}

		params.KCoins = kCoins

		if err := k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) KCoinUpdateBurnAmount(ctx context.Context, req *types.MsgKCoinUpdateBurnAmount) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)
		maxBurnAmount, _ := math.NewIntFromString(req.MaxBurnAmount)
		kCoins := []types.KCoin{}
		found := false

		for _, kCoin := range params.KCoins {
			if kCoin.DexDenom == req.Denom {
				kCoin.MaxBurnAmount = maxBurnAmount
				found = true
			}

			kCoins = append(kCoins, kCoin)
		}

		if !found {
			return types.ErrInvalidKCoin
		}

		params.KCoins = kCoins

		if err := k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) KCoinAddReferences(ctx context.Context, req *types.MsgKCoinAddReferences) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)

		kCoins := []types.KCoin{}
		found := false

		for _, kCoin := range params.KCoins {
			if kCoin.DexDenom == req.Denom {
				kCoin.References = append(kCoin.References, req.References...)
				found = true
			}

			kCoins = append(kCoins, kCoin)
		}

		if !found {
			return types.ErrInvalidKCoin
		}

		params.KCoins = kCoins

		if err := k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) KCoinRemoveReferences(ctx context.Context, req *types.MsgKCoinRemoveReferences) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)

		kCoins := []types.KCoin{}
		found := false

		for _, kCoin := range params.KCoins {
			if kCoin.DexDenom == req.Denom {
				kCoin.References = filterReferences(kCoin.References, req.References)
				found = true
			}

			kCoins = append(kCoins, kCoin)
		}

		if !found {
			return types.ErrInvalidKCoin
		}

		params.KCoins = kCoins

		if err := k.SetParams(innerCtx, params); err != nil {
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
