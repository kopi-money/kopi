package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/tokenfactory module sentinel errors
var (
	ErrInvalidSigner      = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrDenomAlreadyExists = sdkerrors.Register(ModuleName, 1101, "there already is a name with the given denom")
	ErrDenomDoesntExists  = sdkerrors.Register(ModuleName, 1102, "there is no denom with the given name")
	ErrInvalidAddress     = sdkerrors.Register(ModuleName, 1103, "invalid address")
	ErrIncorrectAdmin     = sdkerrors.Register(ModuleName, 1104, "given address is not admin")
	ErrInvalidAmount      = sdkerrors.Register(ModuleName, 1105, "amount format invalid")
	ErrNonPositiveAmount  = sdkerrors.Register(ModuleName, 1106, "amount must be bigger than zero")
)
