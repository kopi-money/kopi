package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/strategies module sentinel errors
var (
	ErrInvalidSigner                    = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidAddress                   = sdkerrors.Register(ModuleName, 1101, "invalid address")
	ErrInvalidAmountFormat              = sdkerrors.Register(ModuleName, 1102, "invalid amount format")
	ErrNegativeAmount                   = sdkerrors.Register(ModuleName, 1103, "amount must not be negative")
	ErrNotEnoughFunds                   = sdkerrors.Register(ModuleName, 1104, "not enough funds")
	ErrNoFunds                          = sdkerrors.Register(ModuleName, 1105, "no funds")
	ErrZeroAmount                       = sdkerrors.Register(ModuleName, 1106, "amount must not be zero")
	ErrZeroMint                         = sdkerrors.Register(ModuleName, 1107, "zero assets minted")
	ErrEmptyVault                       = sdkerrors.Register(ModuleName, 1108, "empty vault")
	ErrNotEnoughVault                   = sdkerrors.Register(ModuleName, 1109, "cannot fulfill redemption request")
	ErrEmptyConditions                  = sdkerrors.Register(ModuleName, 1110, "no conditions given")
	ErrAutomationNotFound               = sdkerrors.Register(ModuleName, 1111, "no automation found for given index")
	ErrAutomationInvalidCreator         = sdkerrors.Register(ModuleName, 1112, "automation belongs to different address")
	ErrAutomationTitleEmpty             = sdkerrors.Register(ModuleName, 1113, "automation title empty")
	ErrAutomationTitleTooLong           = sdkerrors.Register(ModuleName, 1114, "automation title too long")
	ErrAutomationFundsWithdrawlTooLarge = sdkerrors.Register(ModuleName, 1115, "cannot withdraw more than is in funds")
	ErrInvalidIntervalLength            = sdkerrors.Register(ModuleName, 1116, "interval length must be larger than 0")
	ErrEmptyAutomationFunds             = sdkerrors.Register(ModuleName, 1117, "empty automation funds")
	ErrNonExistingValidator             = sdkerrors.Register(ModuleName, 1118, "validator not in list")
	ErrInvalidIntegerFormat             = sdkerrors.Register(ModuleName, 1119, "invalid integer format")
)
