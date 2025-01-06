package keeper

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	dextypes "github.com/kopi-money/kopi/x/dex/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

var factoryDenomReg = regexp.MustCompile(`^factory/[A-F0-9]{64}$`)

func ToFullName(denom string) string {
	if factoryDenomReg.Match([]byte(denom)) {
		return denom
	}

	denom = toHash(denom)
	denom = fmt.Sprintf("factory/%v", denom)
	return denom
}

func toHash(text string) string {
	h := sha256.New()
	h.Write([]byte(text))
	bs := h.Sum(nil)
	text = fmt.Sprintf("%x", bs)
	text = strings.ToUpper(text)
	return text
}

func (k Keeper) GetAllDenoms(ctx context.Context) []types.FactoryDenom {
	iterator := k.factoryDenoms.Iterator(ctx, nil)
	return iterator.GetAll()
}

func (k Keeper) SetDenom(ctx context.Context, denom types.FactoryDenom) {
	k.factoryDenoms.Set(ctx, denom.FullName, denom)
}

func (k Keeper) GetDenomByDisplayName(ctx context.Context, displayName string) (types.FactoryDenom, bool) {
	return k.GetDenomByFullName(ctx, ToFullName(displayName))
}

func (k Keeper) GetDenomByFullName(ctx context.Context, fullName string) (types.FactoryDenom, bool) {
	return k.factoryDenoms.Get(ctx, fullName)
}

func (k Keeper) GetDenomBySymbol(ctx context.Context, symbol string) (types.FactoryDenom, bool) {
	iterator := k.factoryDenoms.Iterator(ctx, nil)
	for iterator.Valid() {
		value := iterator.GetNext()
		if value.Symbol == symbol {
			return value, true
		}
	}

	return types.FactoryDenom{}, false
}

func (k Keeper) CreateDenom(ctx context.Context, address, displayName, symbol, description, iconHash string, exponent uint64) (types.FactoryDenom, error) {
	fullName := ToFullName(displayName)

	if _, exists := k.GetDenomByFullName(ctx, fullName); exists {
		return types.FactoryDenom{}, types.ErrDenomAlreadyExists
	}

	if _, exists := k.GetDenomBySymbol(ctx, symbol); exists {
		return types.FactoryDenom{}, types.ErrSymbolAlreadyExists
	}

	if exponent < 1 {
		return types.FactoryDenom{}, fmt.Errorf("exponent has to be at least 1")
	}

	if exponent > 18 {
		return types.FactoryDenom{}, fmt.Errorf("exponent must not be larger than 18")
	}

	if err := k.processCreationFee(ctx, address); err != nil {
		return types.FactoryDenom{}, fmt.Errorf("processing fee: %v", err)
	}

	factoryDenom := types.FactoryDenom{
		Admin:       address,
		DisplayName: displayName,
		FullName:    fullName,
		Description: description,
		IconHash:    strings.ToUpper(iconHash),
		Symbol:      symbol,
		Exponent:    exponent,
		Mintable:    true,
	}

	k.SetDenom(ctx, factoryDenom)
	return factoryDenom, nil
}

func (k Keeper) processCreationFee(ctx context.Context, address string) error {
	feeAmount := k.getCreationFee(ctx)
	if feeAmount.IsNil() {
		return fmt.Errorf("feeAmount is nil")
	}

	if feeAmount.IsZero() {
		return nil
	}

	addr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return types.ErrInvalidAddress
	}

	coins := sdk.NewCoins(sdk.NewCoin(constants.KUSD, feeAmount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, addr, dextypes.PoolReserve, coins); err != nil {
		return fmt.Errorf("could not send coins from account to module: %w", err)
	}

	return nil
}
