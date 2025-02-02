package keeper

import (
	"context"
	"fmt"

	"github.com/kopi-money/kopi/cache"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) ArbitrageAddDenom(ctx context.Context, req *types.MsgAddArbitrageDenom) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)
		strategyDenoms := params.StrategyDenoms
		buyThreshold, _ := math.LegacyNewDecFromStr(req.BuyThreshold)
		sellTreshold, _ := math.LegacyNewDecFromStr(req.SellThreshold)
		buyAmount, _ := math.NewIntFromString(req.BuyTradeAmount)
		sellAmount, _ := math.NewIntFromString(req.SellTradeAmount)
		redemptionFee, _ := math.LegacyNewDecFromStr(req.RedemptionFee)
		redemptionFeeReserveShare, _ := math.LegacyNewDecFromStr(req.RedemptionFeeReserveShare)

		cAsset, has := k.GetDexDenom(innerCtx, req.CAsset)
		if !has {
			return fmt.Errorf("no dex asset found for given c asset: %v", req.CAsset)
		}

		dexDenom, ratio, err := k.createDexDenom(ctx, req.Name, req.Factor, req.MinLiquidity, req.MinOrderSize, cAsset.Exponent)
		if err != nil {
			return err
		}

		params.DexDenoms = append(params.DexDenoms, dexDenom)

		strategyDenoms.ArbitrageDenoms = append(strategyDenoms.ArbitrageDenoms, types.ArbitrageDenom{
			DexDenom:                  req.Name,
			KCoin:                     req.Kcoin,
			CAsset:                    req.CAsset,
			BuyThreshold:              buyThreshold,
			SellThreshold:             sellTreshold,
			BuyTradeAmount:            buyAmount,
			SellTradeAmount:           sellAmount,
			RedemptionFee:             redemptionFee,
			RedemptionFeeReserveShare: redemptionFeeReserveShare,
		})

		params.StrategyDenoms = strategyDenoms
		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		k.ratios.Set(innerCtx, req.Name, ratio)

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) ArbitrageUpdateBuyThreshold(ctx context.Context, req *types.MsgArbitrageUpdateBuyThreshold) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)
		strategyDenoms := params.StrategyDenoms
		buyTreshold, _ := math.LegacyNewDecFromStr(req.BuyThreshold)

		arbitrageDenoms := []types.ArbitrageDenom{}
		found := false

		for _, arbitrageDenom := range strategyDenoms.ArbitrageDenoms {
			if arbitrageDenom.DexDenom == req.Name {
				arbitrageDenom.BuyThreshold = buyTreshold
				found = true
			}

			arbitrageDenoms = append(arbitrageDenoms, arbitrageDenom)
		}

		if !found {
			return types.ErrInvalidArbitrageDenom
		}

		strategyDenoms.ArbitrageDenoms = arbitrageDenoms
		params.StrategyDenoms = strategyDenoms

		if err := k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}
func (k msgServer) ArbitrageUpdateSellThreshold(ctx context.Context, req *types.MsgArbitrageUpdateSellThreshold) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)
		strategyDenoms := params.StrategyDenoms
		sellTreshold, _ := math.LegacyNewDecFromStr(req.SellThreshold)
		arbitrageDenoms := []types.ArbitrageDenom{}
		found := false

		for _, arbitrageDenom := range strategyDenoms.ArbitrageDenoms {
			if arbitrageDenom.DexDenom == req.Name {
				arbitrageDenom.SellThreshold = sellTreshold
				found = true
			}

			arbitrageDenoms = append(arbitrageDenoms, arbitrageDenom)
		}

		if !found {
			return types.ErrInvalidArbitrageDenom
		}

		strategyDenoms.ArbitrageDenoms = arbitrageDenoms
		params.StrategyDenoms = strategyDenoms

		if err := k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) ArbitrageUpdateBuyAmount(ctx context.Context, req *types.MsgArbitrageUpdateBuyAmount) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)
		strategyDenoms := params.StrategyDenoms

		buyAmount, ok := math.NewIntFromString(req.BuyAmount)
		if !ok {
			return fmt.Errorf("invalid buy amount: %v", req.BuyAmount)
		}

		arbitrageDenoms := []types.ArbitrageDenom{}
		found := false

		for _, arbitrageDenom := range strategyDenoms.ArbitrageDenoms {
			if arbitrageDenom.DexDenom == req.Name {
				arbitrageDenom.BuyTradeAmount = buyAmount
				found = true
			}

			arbitrageDenoms = append(arbitrageDenoms, arbitrageDenom)
		}

		if !found {
			return types.ErrInvalidArbitrageDenom
		}

		strategyDenoms.ArbitrageDenoms = arbitrageDenoms
		params.StrategyDenoms = strategyDenoms

		if err := k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) ArbitrageUpdateSellAmount(ctx context.Context, req *types.MsgArbitrageUpdateSellAmount) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)
		strategyDenoms := params.StrategyDenoms

		sellAmount, ok := math.NewIntFromString(req.SellAmount)
		if !ok {
			return fmt.Errorf("invalid sell amount: %v", req.SellAmount)
		}

		arbitrageDenoms := []types.ArbitrageDenom{}
		found := false

		for _, arbitrageDenom := range strategyDenoms.ArbitrageDenoms {
			if arbitrageDenom.DexDenom == req.Name {
				arbitrageDenom.SellTradeAmount = sellAmount
				found = true
			}

			arbitrageDenoms = append(arbitrageDenoms, arbitrageDenom)
		}

		if !found {
			return types.ErrInvalidArbitrageDenom
		}

		strategyDenoms.ArbitrageDenoms = arbitrageDenoms
		params.StrategyDenoms = strategyDenoms

		if err := k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) ArbitrageUpdateRedemptionFee(ctx context.Context, req *types.MsgArbitrageUpdateRedemptionFee) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)
		strategyDenoms := params.StrategyDenoms

		fee, err := math.LegacyNewDecFromStr(req.RedemptionFee)
		if err != nil {
			return errorsmod.Wrap(err, fmt.Sprintf("invalid fee: %v", req.RedemptionFee))
		}

		arbitrageDenoms := []types.ArbitrageDenom{}
		found := false

		for _, arbitrageDenom := range strategyDenoms.ArbitrageDenoms {
			if arbitrageDenom.DexDenom == req.Name {
				arbitrageDenom.RedemptionFee = fee
				found = true
			}

			arbitrageDenoms = append(arbitrageDenoms, arbitrageDenom)
		}

		if !found {
			return types.ErrInvalidArbitrageDenom
		}

		strategyDenoms.ArbitrageDenoms = arbitrageDenoms
		params.StrategyDenoms = strategyDenoms

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) ArbitrageUpdateRedemptionFeeReserveShare(ctx context.Context, req *types.MsgArbitrageUpdateRedemptionFeeReserveShare) (*types.MsgUpdateParamsResponse, error) {
	err := cache.Transact(ctx, func(innerCtx context.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		params := k.GetParams(innerCtx)
		strategyDenoms := params.StrategyDenoms

		share, err := math.LegacyNewDecFromStr(req.RedemptionFeeReserveShare)
		if err != nil {
			return errorsmod.Wrap(err, fmt.Sprintf("invalid fee: %v", req.RedemptionFeeReserveShare))
		}

		arbitrageDenoms := []types.ArbitrageDenom{}
		found := false

		for _, arbitrageDenom := range strategyDenoms.ArbitrageDenoms {
			if arbitrageDenom.DexDenom == req.Name {
				arbitrageDenom.RedemptionFeeReserveShare = share
				found = true
			}

			arbitrageDenoms = append(arbitrageDenoms, arbitrageDenom)
		}

		if !found {
			return types.ErrInvalidArbitrageDenom
		}

		strategyDenoms.ArbitrageDenoms = arbitrageDenoms
		params.StrategyDenoms = strategyDenoms

		if err = k.SetParams(innerCtx, params); err != nil {
			return err
		}

		return nil
	})

	return &types.MsgUpdateParamsResponse{}, err
}
