package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/dex module sentinel errors
var (
	ErrInvalidSigner               = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrDenomNotFound               = sdkerrors.Register(ModuleName, 1101, "denom not found")
	ErrNotEnoughFunds              = sdkerrors.Register(ModuleName, 1102, "not enough funds")
	ErrNegativePrice               = sdkerrors.Register(ModuleName, 1103, "negative price")
	ErrNotEnoughLiquidity          = sdkerrors.Register(ModuleName, 1104, "not enough liquidity")
	ErrSameDenom                   = sdkerrors.Register(ModuleName, 1105, "cannot trade same denom")
	ErrInvalidAddress              = sdkerrors.Register(ModuleName, 1106, "invalid address")
	ErrBaseLiqEmpty                = sdkerrors.Register(ModuleName, 1107, "base liquidity is empty")
	ErrNegativeAmount              = sdkerrors.Register(ModuleName, 1108, "amount must not be negative")
	ErrMaxPriceNotSet              = sdkerrors.Register(ModuleName, 1109, "max_price not set")
	ErrInvalidDecimalFormat        = sdkerrors.Register(ModuleName, 1110, "invalid decimal format")
	ErrInvalidIntegerFormat        = sdkerrors.Register(ModuleName, 1111, "invalid integer format")
	ErrItemNotFound                = sdkerrors.Register(ModuleName, 1112, "item not found")
	ErrInvalidCreator              = sdkerrors.Register(ModuleName, 1113, "item does not belong to creator")
	ErrPriceTooLow                 = sdkerrors.Register(ModuleName, 1114, "trade cannot be fully completed given the price")
	ErrNegativeTradeAmount         = sdkerrors.Register(ModuleName, 1115, "set max_price is too low")
	ErrOrderNotFound               = sdkerrors.Register(ModuleName, 1116, "order not found for index")
	ErrZeroAmount                  = sdkerrors.Register(ModuleName, 1117, "zero amount given")
	ErrNoLiquidity                 = sdkerrors.Register(ModuleName, 1118, "no liquidity")
	ErrNoCoinSourceGiven           = sdkerrors.Register(ModuleName, 1119, "no coin source given")
	ErrNoCoinTargetGiven           = sdkerrors.Register(ModuleName, 1120, "no coin target given")
	ErrTradeAmountTooSmall         = sdkerrors.Register(ModuleName, 1121, "trade amount too small")
	ErrNilRatio                    = sdkerrors.Register(ModuleName, 1122, "ratio is nil")
	ErrZeroPrice                   = sdkerrors.Register(ModuleName, 1123, "zero price")
	ErrOrderSizeTooSmall           = sdkerrors.Register(ModuleName, 1124, "order size too small")
	ErrMinimumTradeAmountTooLarge  = sdkerrors.Register(ModuleName, 1125, "minimum trade amount must not be larger than the actual trade amount")
	ErrZeroTrade                   = sdkerrors.Register(ModuleName, 1126, "zero trade")
	ErrCannotRemoveCollateralDenom = sdkerrors.Register(ModuleName, 1127, "cannot remove collateral denom")
)
