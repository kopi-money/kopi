package keeper

import (
	"context"

	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k Keeper) SetGenesisTokenConversion(ctx context.Context, conversion types.GenesisTokenConversion) {
	var conversions []types.TokenConversionEntry
	for _, c := range conversion.Entries {
		conversions = append(conversions, types.TokenConversionEntry{
			NewToken:       c.NewToken,
			ConversionRate: c.ConversionRate,
		})
	}

	k.tokenConversions.Set(ctx, conversion.FactoryDenomFullName, types.TokenConversion{
		Entries: conversions,
	})
}

func (k Keeper) GetGenesisTokenConversions(ctx context.Context) (list []types.GenesisTokenConversion) {
	iterator := k.tokenConversions.Iterator(ctx, nil)
	for iterator.Valid() {
		keyValue := iterator.GetNextKeyValue()
		conversion := keyValue.Value().Value()

		list = append(list, types.GenesisTokenConversion{
			FactoryDenomFullName: keyValue.Key(),
			Entries:              conversion.Entries,
		})
	}

	return
}

func (k Keeper) GetTokenConversion(ctx context.Context, denom string) (types.TokenConversion, bool) {
	return k.tokenConversions.Get(ctx, denom)
}
