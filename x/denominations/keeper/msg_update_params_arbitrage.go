package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) ArbitrageAddDenom(ctx context.Context, req *types.MsgAddArbitrageDenom) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
	strategyDenoms := params.StrategyDenoms

	buyThreshold, _ := math.LegacyNewDecFromStr(req.BuyThreshold)
	sellTreshold, _ := math.LegacyNewDecFromStr(req.SellThreshold)
	buyAmount, _ := math.NewIntFromString(req.BuyTradeAmount)
	sellAmount, _ := math.NewIntFromString(req.SellTradeAmount)
	redemptionFee, _ := math.LegacyNewDecFromStr(req.RedemptionFee)
	redemptionFeeReserveShare, _ := math.LegacyNewDecFromStr(req.RedemptionFeeReserveShare)

	cAsset, err := k.GetDexDenom(ctx, req.CAsset)
	if err != nil {
		return nil, err
	}

	dexDenom, ratio, err := k.CreateDexDenom(ctx, req.Name, req.Factor, req.MinLiquidity, req.MinOrderSize, req.MinVirtualLiquidity, cAsset.Exponent)
	if err != nil {
		return nil, err
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
	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	k.ratios.Set(ctx, req.Name, ratio)

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) ArbitrageUpdateBuyThreshold(ctx context.Context, req *types.MsgArbitrageUpdateBuyThreshold) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
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
		return nil, types.ErrInvalidArbitrageDenom
	}

	strategyDenoms.ArbitrageDenoms = arbitrageDenoms
	params.StrategyDenoms = strategyDenoms

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
func (k msgServer) ArbitrageUpdateSellThreshold(ctx context.Context, req *types.MsgArbitrageUpdateSellThreshold) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
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
		return nil, types.ErrInvalidArbitrageDenom
	}

	strategyDenoms.ArbitrageDenoms = arbitrageDenoms
	params.StrategyDenoms = strategyDenoms

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) ArbitrageUpdateBuyAmount(ctx context.Context, req *types.MsgArbitrageUpdateBuyAmount) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
	strategyDenoms := params.StrategyDenoms

	buyAmount, ok := math.NewIntFromString(req.BuyAmount)
	if !ok {
		return nil, fmt.Errorf("invalid buy amount: %v", req.BuyAmount)
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
		return nil, types.ErrInvalidArbitrageDenom
	}

	strategyDenoms.ArbitrageDenoms = arbitrageDenoms
	params.StrategyDenoms = strategyDenoms

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) ArbitrageUpdateSellAmount(ctx context.Context, req *types.MsgArbitrageUpdateSellAmount) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
	strategyDenoms := params.StrategyDenoms

	sellAmount, ok := math.NewIntFromString(req.SellAmount)
	if !ok {
		return nil, fmt.Errorf("invalid sell amount: %v", req.SellAmount)
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
		return nil, types.ErrInvalidArbitrageDenom
	}

	strategyDenoms.ArbitrageDenoms = arbitrageDenoms
	params.StrategyDenoms = strategyDenoms

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) ArbitrageUpdateRedemptionFee(ctx context.Context, req *types.MsgArbitrageUpdateRedemptionFee) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
	strategyDenoms := params.StrategyDenoms

	fee, err := math.LegacyNewDecFromStr(req.RedemptionFee)
	if err != nil {
		return nil, errorsmod.Wrap(err, fmt.Sprintf("invalid fee: %v", req.RedemptionFee))
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
		return nil, types.ErrInvalidArbitrageDenom
	}

	strategyDenoms.ArbitrageDenoms = arbitrageDenoms
	params.StrategyDenoms = strategyDenoms

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, err
}

func (k msgServer) ArbitrageUpdateRedemptionFeeReserveShare(ctx context.Context, req *types.MsgArbitrageUpdateRedemptionFeeReserveShare) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
	strategyDenoms := params.StrategyDenoms

	share, err := math.LegacyNewDecFromStr(req.RedemptionFeeReserveShare)
	if err != nil {
		return nil, errorsmod.Wrap(err, fmt.Sprintf("invalid fee: %v", req.RedemptionFeeReserveShare))
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
		return nil, types.ErrInvalidArbitrageDenom
	}

	strategyDenoms.ArbitrageDenoms = arbitrageDenoms
	params.StrategyDenoms = strategyDenoms

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, err
}
