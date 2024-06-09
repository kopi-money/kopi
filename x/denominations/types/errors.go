package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/denominations module sentinel errors
var (
	ErrInvalidSigner          = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidDexAsset        = sdkerrors.Register(ModuleName, 1101, "given denom is no dex asset")
	ErrInvalidCAsset          = sdkerrors.Register(ModuleName, 1102, "given denom is no c asset")
	ErrInvalidKCoin           = sdkerrors.Register(ModuleName, 1103, "given denom is no kcoin")
	ErrInvalidCollateralDenom = sdkerrors.Register(ModuleName, 1104, "given collateral denom is no collateral denom")
	ErrInvalidAmount          = sdkerrors.Register(ModuleName, 1105, "invalid amount")
)
