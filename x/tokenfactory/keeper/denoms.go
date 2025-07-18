package keeper

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	reservetypes "github.com/kopi-money/kopi/x/reserve/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func ToFullName(creator, symbol string) string {
	return strings.ToLower(fmt.Sprintf("factory/%v/%v", creator, symbol))
}

func (k Keeper) GetAllDenoms(ctx context.Context) []types.FactoryDenom {
	iterator := k.factoryDenoms.Iterator(ctx, nil)
	return iterator.GetAll()
}

func (k Keeper) SetDenom(ctx context.Context, denom types.FactoryDenom) {
	k.factoryDenoms.Set(ctx, denom.FullName, denom)
}

func (k Keeper) GetDenom(ctx context.Context, address, symbol string) (types.FactoryDenom, bool) {
	return k.GetDenomByFullName(ctx, ToFullName(address, symbol))
}

func (k Keeper) GetDenomByFullName(ctx context.Context, fullName string) (types.FactoryDenom, bool) {
	return k.factoryDenoms.Get(ctx, fullName)
}

func (k Keeper) IsFactoryDenom(ctx context.Context, fullName string) bool {
	_, has := k.factoryDenoms.Get(ctx, fullName)
	return has
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

func (k Keeper) CreateDenom(ctx context.Context, address, displayName, symbol, feeDenom, description, website, iconHash, localName string, exponent, categoryIndex uint64, mintable bool) (types.FactoryDenom, error) {
	fullName := ToFullName(address, symbol)

	if _, exists := k.GetDenomByFullName(ctx, fullName); exists {
		return types.FactoryDenom{}, types.ErrDenomAlreadyExists
	}

	if exponent < 1 {
		return types.FactoryDenom{}, fmt.Errorf("exponent has to be at least 1")
	}

	if exponent > 18 {
		return types.FactoryDenom{}, fmt.Errorf("exponent must not be larger than 18")
	}

	category, has := k.getCategory(ctx, categoryIndex)
	if !has {
		return types.FactoryDenom{}, types.ErrCategoryDoesNotExist
	}

	if category.IsIbc {
		if err := k.checkLocalToken(ctx, localName); err != nil {
			return types.FactoryDenom{}, err
		}

		mintable = false
	} else {
		if localName != "" {
			return types.FactoryDenom{}, types.ErrInvalidLocalNameSet
		}
	}

	// website is allowed to be empty, so only check the uri when it's not empty
	if website != "" {
		u, err := url.ParseRequestURI(website)
		if err != nil {
			return types.FactoryDenom{}, types.ErrWebsiteURLInvalid
		}

		if u.Scheme != "https" && u.Scheme != "http" {
			return types.FactoryDenom{}, types.ErrWebsiteURLInvalid
		}
	}

	if err := k.processCreationFee(ctx, category, feeDenom, address); err != nil {
		return types.FactoryDenom{}, fmt.Errorf("processing fee: %w", err)
	}

	blocktime := sdk.UnwrapSDKContext(ctx).BlockTime()
	factoryDenom := types.FactoryDenom{
		Admin:                 address,
		DisplayName:           displayName,
		FullName:              fullName,
		Description:           description,
		Website:               website,
		IconHash:              strings.ToUpper(iconHash),
		Symbol:                symbol,
		Exponent:              exponent,
		CategoryIndex:         categoryIndex,
		LastImageChange:       blocktime,
		LastWebsiteChange:     blocktime,
		LastDescriptionChange: blocktime,
		Mintable:              mintable,
		LocalName:             localName,
	}

	k.SetDenom(ctx, factoryDenom)

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"factory_denom_created",
			sdk.NewAttribute("factory_denom_full_name", factoryDenom.FullName),
			sdk.NewAttribute("creator", factoryDenom.Admin),
			sdk.NewAttribute("description", factoryDenom.Description),
			sdk.NewAttribute("website", factoryDenom.Website),
			sdk.NewAttribute("icon_hash", factoryDenom.IconHash),
		),
	})

	return factoryDenom, nil
}

func (k Keeper) processCreationFee(ctx context.Context, category types.Category, feeDenom, address string) error {
	addr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return types.ErrInvalidAddress
	}

	if feeDenom == "" {
		feeDenom = constants.KUSD
	} else if !k.DenomKeeper.IsFactoryPoolDenom(ctx, feeDenom) {
		return types.ErrNoValidPoolDenom
	}

	coins := sdk.NewCoins(sdk.NewCoin(feeDenom, category.CreationPrice))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, addr, reservetypes.BuyingKCoins, coins); err != nil {
		return fmt.Errorf("send coins from account to module: %w", err)
	}

	return nil
}

func (k Keeper) checkLocalToken(ctx context.Context, localName string) error {
	if !strings.HasPrefix(localName, "ibc/") {
		return types.ErrInvalidLocalToken
	}

	if k.BankKeeper.GetSupply(ctx, localName).IsZero() {
		return types.ErrLocalTokenDoesNotExist
	}

	return nil
}
