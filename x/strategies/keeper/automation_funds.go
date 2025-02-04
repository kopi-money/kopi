package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/cache"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/strategies/types"
)

// SetAutomationFunds sets a specific liquidity in the store from its index. When the index is zero, i.e. it's a new entry,
// the NextIndex is increased and updated as well.
func (k Keeper) SetAutomationFunds(ctx context.Context, address string, funds math.Int) {
	if funds.GT(math.ZeroInt()) {
		k.automationFunds.Set(ctx, address, types.AutomationFunds{Funds: funds})
	} else {
		k.automationFunds.Remove(ctx, address)
	}
}

func (k Keeper) SetGenesisAutomationFunds(ctx context.Context, genesisAutomationFunds []*types.GenesisAutomationFunds) error {
	for _, automationFunds := range genesisAutomationFunds {
		if automationFunds == nil {
			return fmt.Errorf("automationFunds is nil")
		}

		k.automationFunds.Set(ctx, automationFunds.Address, types.AutomationFunds{Funds: automationFunds.Funds})
	}

	return nil
}

func (k Keeper) consumeAutomationFunds(ctx context.Context, accAddr sdk.AccAddress, amount uint64, totalAmount *uint64) error {
	return cache.Transact(ctx, func(innerCtx context.Context) error {
		funds := k.GetAutomationFunds(innerCtx, accAddr.String())
		amountInt := math.NewInt(int64(amount))

		if amountInt.GT(funds) {
			return types.ErrNotEnoughFunds
		}

		*totalAmount += amount
		funds = funds.Sub(amountInt)
		k.SetAutomationFunds(innerCtx, accAddr.String(), funds)
		return nil
	})
}

func (k Keeper) GetAutomationFunds(ctx context.Context, address string) math.Int {
	funds, has := k.automationFunds.Get(ctx, address)
	if !has {
		return math.ZeroInt()
	}

	return funds.Funds
}

func (k Keeper) GetAllAutomationFunds(ctx context.Context) (list []*types.GenesisAutomationFunds) {
	iterator := k.automationFunds.Iterator(ctx, nil)
	for iterator.Valid() {
		keyValue := iterator.GetNextKeyValue()
		list = append(list, &types.GenesisAutomationFunds{
			Address: keyValue.Key(),
			Funds:   keyValue.Value().Value().Funds,
		})
	}

	return
}

func (k Keeper) AutomationFundsIterator(ctx context.Context) cache.Iterator[string, types.AutomationFunds] {
	return k.automationFunds.Iterator(ctx, nil)
}
