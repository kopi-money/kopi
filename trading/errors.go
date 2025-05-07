package trading

import (
	"errors"
)

var (
	ErrRequestedAmountTooLarge = errors.New("requested amount too large")
	ErrEmptyTrade              = errors.New("given max price results in empty trade")
	ErrMarketPriceTooHigh      = errors.New("market price too high")
	ErrPriceTooLow             = errors.New("trade cannot be fully completed given the price")
	ErrTradeAmountTooSmall     = errors.New("trade amount too small")
	ErrInvalidMaxPriceFormat   = errors.New("invalid max price format")
	ErrMaxPriceNotPositive     = errors.New("max price is not positive")
	ErrTradeAmountNotPositive  = errors.New("trade amount is not positive")
)
