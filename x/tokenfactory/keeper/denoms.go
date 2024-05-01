package keeper

import (
	"context"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"fmt"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func toFullName(denom string) string {
	return fmt.Sprintf("factory/%v", denom)
}

func (k Keeper) GetAllDenoms(ctx context.Context) (list []types.FactoryDenom) {
	iterator := k.DenomIterator(ctx)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.FactoryDenom
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) SetDenom(ctx context.Context, denom types.FactoryDenom) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixFactoryDenoms))

	b := k.cdc.MustMarshal(&denom)
	store.Set(types.KeyDenom(denom.Denom), b)
}

// GetDenom returns a deposits from its index
func (k Keeper) GetDenom(ctx context.Context, denom string) (types.FactoryDenom, bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixFactoryDenoms))

	b := store.Get(types.KeyDenom(denom))
	if b == nil {
		return types.FactoryDenom{}, false
	}

	var deposit types.FactoryDenom
	k.cdc.MustUnmarshal(b, &deposit)
	return deposit, true
}

func (k Keeper) DenomStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.Key(types.KeyPrefixFactoryDenoms))
}

func (k Keeper) DenomIterator(ctx context.Context) storetypes.Iterator {
	return storetypes.KVStorePrefixIterator(k.DenomStore(ctx), []byte{})
}

func (k Keeper) CreateDenom(ctx context.Context, denom, address string) error {
	_, exists := k.GetDenom(ctx, denom)
	if exists {
		return types.ErrDenomAlreadyExists
	}

	if err := k.processCreationFee(ctx, address); err != nil {
		return err
	}

	k.SetDenom(ctx, types.FactoryDenom{
		Denom: denom,
		Admin: address,
	})

	return nil
}

func (k Keeper) processCreationFee(ctx context.Context, address string) error {
	addr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return types.ErrInvalidAddress
	}

	feeAmount := k.GetParams(ctx).CreationFee
	if feeAmount.IsNil() {
		return fmt.Errorf("feeAmount is nil")
	}

	if feeAmount.Equal(math.ZeroInt()) {
		return nil
	}

	coins := sdk.NewCoins(sdk.NewCoin("ukopi", feeAmount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, coins); err != nil {
		return err
	}

	if err = k.BankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
		return err
	}

	return nil
}
