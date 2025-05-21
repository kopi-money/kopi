package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

var (
	_ sdk.Msg = &MsgCreatePool{}
	_ sdk.Msg = &MsgAddLiquidity{}
	_ sdk.Msg = &MsgUnlockLiquidity{}
	_ sdk.Msg = &MsgUpdateLiquidityPoolSettings{}
)

func (msg *MsgCreatePool) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if err := ValidateDenomName(msg.FullFactoryDenomName); err != nil {
		return fmt.Errorf("full_factory_denom_name: %w", err)
	}

	if err := denomtypes.IsInt(msg.FactoryDenomAmount, math.ZeroInt()); err != nil {
		return fmt.Errorf("factory_denom_amount: %w", err)
	}

	if err := denomtypes.IsInt(msg.KCoinAmount, math.ZeroInt()); err != nil {
		return fmt.Errorf("k_coin_amount: %w", err)
	}

	if err := denomtypes.IsDec(msg.PoolFee, math.LegacyZeroDec()); err != nil {
		return fmt.Errorf("pool_fee: %w", err)
	}

	return nil
}

func (msg *MsgAddLiquidity) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if err := ValidateDenomName(msg.FullFactoryDenomName); err != nil {
		return fmt.Errorf("full_factory_denom_name: %w", err)
	}

	if err := denomtypes.IsInt(msg.FactoryDenomAmount, math.ZeroInt()); err != nil {
		return fmt.Errorf("factory_denom_amount: %w", err)
	}

	return nil
}

func (msg *MsgUnlockLiquidity) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if err := ValidateDenomName(msg.FullFactoryDenomName); err != nil {
		return fmt.Errorf("full_factory_denom_name: %w", err)
	}

	if err := denomtypes.IsInt(msg.FactoryDenomAmount, math.ZeroInt()); err != nil {
		return fmt.Errorf("factory_denom_amount: %w", err)
	}

	return nil
}

func (msg *MsgUpdateLiquidityPoolSettings) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if err := ValidateDenomName(msg.FullFactoryDenomName); err != nil {
		return fmt.Errorf("full_factory_denom_name: %w", err)
	}

	if err := denomtypes.IsDec(msg.PoolFee, math.LegacyZeroDec()); err != nil {
		return fmt.Errorf("pool_fee: %w", err)
	}

	return nil
}
