package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) CreateTokenConversion(ctx context.Context, msg *types.MsgCreateTokenConversion) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExist
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	if _, has = k.tokenConversions.Get(ctx, factoryDenom.FullName); has {
		return nil, types.ErrTokenConversionAlreadyExists
	}

	if len(msg.Conversions) == 0 {
		return nil, types.ErrTokenConversionListEmpty
	}

	var (
		supply      = k.BankKeeper.GetSupply(ctx, factoryDenom.FullName)
		conversions []types.TokenConversionEntry
		coins       sdk.Coins
		rate        math.LegacyDec
	)

	for _, conversion := range msg.Conversions {
		if conversion.Denom == factoryDenom.FullName {
			return nil, types.ErrTokenConversionSameToken
		}

		if found, _ := coins.Find(conversion.Denom); found {
			return nil, types.ErrTokenConversionDuplicateToken
		}

		if convFactoryDenom, isFactoryDenom := k.GetDenomByFullName(ctx, conversion.Denom); isFactoryDenom {
			if convFactoryDenom.Mintable {
				return nil, types.ErrTokenConversionMintable
			}
		}

		rate, err = math.LegacyNewDecFromStr(conversion.ConversationRate)
		if err != nil {
			return nil, fmt.Errorf("invalid conversion rate: %w", err)
		}

		if !rate.IsPositive() {
			return nil, fmt.Errorf("invalid conversion rate: %v", rate)
		}

		requiredAmount := supply.Amount.ToLegacyDec().Mul(rate).Ceil().TruncateInt()
		spendable := k.BankKeeper.SpendableCoin(ctx, acc, conversion.Denom).Amount
		if spendable.LT(requiredAmount) {
			return nil, fmt.Errorf("not enough funds for %v: %v needed, %v spendable", conversion.Denom, requiredAmount, spendable)
		}

		coins = coins.Add(sdk.NewCoin(conversion.Denom, requiredAmount))
		conversions = append(conversions, types.TokenConversionEntry{
			NewToken:       conversion.Denom,
			ConversionRate: rate,
		})
	}

	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolConversions, coins); err != nil {
		return nil, fmt.Errorf("send to conversions pool: %w", err)
	}

	k.tokenConversions.Set(ctx, factoryDenom.FullName, types.TokenConversion{Entries: conversions})
	return &types.Void{}, nil
}

func (k msgServer) ConvertTokens(ctx context.Context, msg *types.MsgConvertTokens) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExist
	}

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	tokenConversion, has := k.tokenConversions.Get(ctx, factoryDenom.FullName)
	if !has {
		return nil, types.ErrTokenConversionDoesNotExist
	}

	amount, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %v", msg.Amount)
	}

	if !amount.IsPositive() {
		return nil, types.ErrNonPositiveAmount
	}

	if err = k.burnDenom(ctx, factoryDenom, amount, acc.String()); err != nil {
		return nil, err
	}

	coins := sdk.NewCoins()
	for _, conversion := range tokenConversion.Entries {
		newAmount := amount.ToLegacyDec().Mul(conversion.ConversionRate).TruncateInt()
		coins = coins.Add(sdk.NewCoin(conversion.NewToken, newAmount))
	}

	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolConversions, acc, coins); err != nil {
		return nil, fmt.Errorf("send from conversions pool: %w", err)
	}

	return &types.Void{}, nil
}
