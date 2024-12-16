package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/tokenfactory module sentinel errors
var (
	ErrInvalidSigner       = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrDenomAlreadyExists  = sdkerrors.Register(ModuleName, 1101, "there already is a name with the given denom")
	ErrDenomDoesNotExists  = sdkerrors.Register(ModuleName, 1102, "there is no denom with the given name")
	ErrInvalidAddress      = sdkerrors.Register(ModuleName, 1103, "invalid address")
	ErrIncorrectAdmin      = sdkerrors.Register(ModuleName, 1104, "given address is not admin")
	ErrInvalidAmountFormat = sdkerrors.Register(ModuleName, 1105, "amount format invalid")
	ErrNonPositiveAmount   = sdkerrors.Register(ModuleName, 1106, "amount must be bigger than zero")
	ErrEmptyKCoin          = sdkerrors.Register(ModuleName, 1107, "given kcoin was empty")
	ErrNoKCoin             = sdkerrors.Register(ModuleName, 1108, "given denom is no kcoin")
	ErrPoolAlreadyExists   = sdkerrors.Register(ModuleName, 1109, "liquitidy pool for given pool already exists")
	ErrPoolDoesNotExist    = sdkerrors.Register(ModuleName, 1110, "factory denom has no liquidity pool")
	ErrNegativeLiquidity   = sdkerrors.Register(ModuleName, 1111, "negative liquidity amount")
	ErrSameDenom           = sdkerrors.Register(ModuleName, 1112, "same denom given twice")
	ErrInsufficientFunds   = sdkerrors.Register(ModuleName, 1113, "insufficient funds")
	ErrInvalidPriceFormat  = sdkerrors.Register(ModuleName, 1114, "invalid price format")
	ErrInvalidFeeFormat    = sdkerrors.Register(ModuleName, 1115, "invalid fee format")
	ErrInvalidNegativeFee  = sdkerrors.Register(ModuleName, 1116, "pool fee must not be negative")
	ErrPoolFeeToLarge      = sdkerrors.Register(ModuleName, 1117, "fee must not be larger than 1")
	ErrTradeAmountTooSmall = sdkerrors.Register(ModuleName, 1118, "trade amount too small")
	ErrMarketPriceTooHigh  = sdkerrors.Register(ModuleName, 1119, "market price too high")
	ErrEmptyTrade          = sdkerrors.Register(ModuleName, 1120, "given max price results in empty trade")
	ErrAmountBelowMinimum  = sdkerrors.Register(ModuleName, 1121, "amount below minimum size")
	ErrNotMintable         = sdkerrors.Register(ModuleName, 1122, "given denom is not mintable")
	ErrAmountTooLarge      = sdkerrors.Register(ModuleName, 1123, "given amount exceeds user's share of liquidity pool")
	ErrSymbolAlreadyExists = sdkerrors.Register(ModuleName, 1124, "there already is a denom with the given symbol")
	ErrUnlockTooShort      = sdkerrors.Register(ModuleName, 1125, "unlock period too short")
	ErrShorterUnlockPeriod = sdkerrors.Register(ModuleName, 1126, "new unlock period must not be shorter than old unlock period")
)
