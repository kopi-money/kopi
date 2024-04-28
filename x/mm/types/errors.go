package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/mm module sentinel errors
var (
	ErrInvalidSigner                  = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidDepositDenom            = sdkerrors.Register(ModuleName, 1101, "given denom is no valid deposit denom")
	ErrInvalidAmountFormat            = sdkerrors.Register(ModuleName, 1102, "invalid amount format")
	ErrNegativeAmount                 = sdkerrors.Register(ModuleName, 1103, "amount must not be negative")
	ErrInvalidAddress                 = sdkerrors.Register(ModuleName, 1104, "invalid address")
	ErrNotEnoughFunds                 = sdkerrors.Register(ModuleName, 1105, "not enough funds")
	ErrRedemptionRequestAlreadyExists = sdkerrors.Register(ModuleName, 1106, "redemption request for address and denom already exists")
	ErrRedemptionRequestNotFound      = sdkerrors.Register(ModuleName, 1107, "no redemption request exists for address and denom")
	ErrZeroAmount                     = sdkerrors.Register(ModuleName, 1108, "amount must not be zero")
	ErrInvalidCollateralDenom         = sdkerrors.Register(ModuleName, 1109, "given denom is no valid collateral denom")
	ErrNoCollateralFound              = sdkerrors.Register(ModuleName, 1110, "no collateral exists for address and denom")
	ErrNoLoanFound                    = sdkerrors.Register(ModuleName, 1111, "no loan found for given address and denom")
	ErrDepositLimitExceeded           = sdkerrors.Register(ModuleName, 1112, "deposit limit for collateral denom has been exceeded")
	ErrRedemptionFeeTooLow            = sdkerrors.Register(ModuleName, 1113, "redemption fee must not be lower than minimum fee")
	ErrRedemptionFeeTooHigh           = sdkerrors.Register(ModuleName, 1114, "priority must not be larger than 1")
	ErrZeroCAssets                    = sdkerrors.Register(ModuleName, 1115, "zero c assets minted")
	ErrNegativeCollateral             = sdkerrors.Register(ModuleName, 1116, "collateral amount is negative")
	ErrNotEnoughFundsInVault          = sdkerrors.Register(ModuleName, 1117, "not enough funds in vault")
	ErrBorrowLimitExceeded            = sdkerrors.Register(ModuleName, 1118, "denom borrow limit exceeded")
)
