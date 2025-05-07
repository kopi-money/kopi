package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/dex module sentinel errors
var (
	ErrInvalidSigner               = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrNotEnoughFunds              = sdkerrors.Register(ModuleName, 1101, "not enough funds")
	ErrNegativePrice               = sdkerrors.Register(ModuleName, 1102, "negative price")
	ErrSameDenom                   = sdkerrors.Register(ModuleName, 1103, "cannot trade same denom")
	ErrInvalidAddress              = sdkerrors.Register(ModuleName, 1104, "invalid address")
	ErrBaseLiqEmpty                = sdkerrors.Register(ModuleName, 1105, "base liquidity is empty")
	ErrNegativeAmount              = sdkerrors.Register(ModuleName, 1106, "amount must not be negative")
	ErrMaxPriceNotSet              = sdkerrors.Register(ModuleName, 1107, "max_price not set")
	ErrInvalidDecimalFormat        = sdkerrors.Register(ModuleName, 1108, "invalid decimal format")
	ErrInvalidIntegerFormat        = sdkerrors.Register(ModuleName, 1109, "invalid integer format")
	ErrItemNotFound                = sdkerrors.Register(ModuleName, 1110, "item not found")
	ErrInvalidCreator              = sdkerrors.Register(ModuleName, 1111, "item does not belong to creator")
	ErrNegativeTradeAmount         = sdkerrors.Register(ModuleName, 1112, "set max_price is too low")
	ErrOrderNotFound               = sdkerrors.Register(ModuleName, 1113, "order not found for index")
	ErrZeroAmount                  = sdkerrors.Register(ModuleName, 1114, "zero amount given")
	ErrNoCoinSourceGiven           = sdkerrors.Register(ModuleName, 1115, "no coin source given")
	ErrNoCoinTargetGiven           = sdkerrors.Register(ModuleName, 1116, "no coin target given")
	ErrNilRatio                    = sdkerrors.Register(ModuleName, 1117, "ratio is nil")
	ErrZeroPrice                   = sdkerrors.Register(ModuleName, 1118, "zero price")
	ErrOrderSizeTooSmall           = sdkerrors.Register(ModuleName, 1119, "order size too small")
	ErrMinimumTradeAmountTooLarge  = sdkerrors.Register(ModuleName, 1120, "minimum trade amount must not be larger than the actual trade amount")
	ErrZeroTrade                   = sdkerrors.Register(ModuleName, 1121, "zero trade")
	ErrCannotRemoveCollateralDenom = sdkerrors.Register(ModuleName, 1122, "cannot remove collateral denom")
	ErrNoLiquidityGiving           = sdkerrors.Register(ModuleName, 1123, "no liquidity for giving denom")
	ErrNoLiquidityReceiving        = sdkerrors.Register(ModuleName, 1124, "no liquidity for receiving denom")
	ErrNotEnoughBuyableLiquidity   = sdkerrors.Register(ModuleName, 1125, "not enough buyable liquidity")
	ErrNotEnoughUsableLiquidity    = sdkerrors.Register(ModuleName, 1126, "not enough usage liquidity for address")
)
