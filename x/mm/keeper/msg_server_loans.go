package keeper

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/mm/types"
)

func (k msgServer) Borrow(goCtx context.Context, msg *types.MsgBorrow) (*types.Void, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	cAsset, err := k.DenomKeeper.GetCAssetByBaseName(ctx, msg.Denom)
	if err != nil {
		return nil, types.ErrInvalidDepositDenom
	}

	amountStr := strings.ReplaceAll(msg.Amount, ",", "")
	amount, err := math.LegacyNewDecFromStr(amountStr)
	if err != nil {
		return nil, types.ErrInvalidAmountFormat
	}

	if amount.LT(math.LegacyZeroDec()) {
		return nil, types.ErrNegativeAmount
	}

	if amount.Equal(math.LegacyZeroDec()) {
		return nil, types.ErrZeroAmount
	}

	address, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
	vault := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())
	available := vault.AmountOf(msg.Denom)

	if available.LT(amount.TruncateInt()) {
		return nil, types.ErrNotEnoughFundsInVault
	}

	borrowableAmount, err := k.calculateBorrowableAmount(ctx, msg.Creator, msg.Denom)
	if err != nil {
		return nil, err
	}

	if borrowableAmount.LT(amount) {
		errMsg := fmt.Errorf("borrow threshold passed, borrowable: %v, requested: %v", borrowableAmount.String(), amount.String())
		return nil, errMsg
	}

	if cAsset.MinimumLoanSize.GT(math.ZeroInt()) && amount.LT(cAsset.MinimumLoanSize.ToLegacyDec()) {
		return nil, types.ErrLoanSizeTooSmall
	}

	if k.checkBorrowLimitExceeded(ctx, cAsset, amount) {
		return nil, types.ErrBorrowLimitExceeded
	}

	loanIndex, _ := k.updateLoan(ctx, msg.Denom, msg.Creator, amount)

	coins := sdk.NewCoins(sdk.NewCoin(msg.Denom, amount.Ceil().TruncateInt()))
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolVault, address, coins); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("funds_borrowed",
			sdk.Attribute{Key: "address", Value: msg.Creator},
			sdk.Attribute{Key: "denom", Value: msg.Denom},
			sdk.Attribute{Key: "amount", Value: msg.Amount},
			sdk.Attribute{Key: "index", Value: strconv.Itoa(int(loanIndex))},
		),
	)

	return &types.Void{}, nil
}

func (k msgServer) RepayLoan(goCtx context.Context, msg *types.MsgRepayLoan) (*types.Void, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	loanValue := k.GetLoanValue(ctx, msg.Denom, msg.Creator)
	if loanValue.Equal(math.LegacyZeroDec()) {
		return nil, types.ErrNoLoanFound
	}

	if err := k.repay(ctx, ctx.EventManager(), msg.Denom, msg.Creator, loanValue); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k msgServer) PartiallyRepayLoan(goCtx context.Context, msg *types.MsgPartiallyRepayLoan) (*types.Void, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, err := k.DenomKeeper.GetCAssetByBaseName(ctx, msg.Denom); err != nil {
		return nil, types.ErrInvalidDepositDenom
	}

	_, found := k.GetLoan(ctx, msg.Denom, msg.Creator)
	if !found {
		return nil, types.ErrNoLoanFound
	}

	amountStr := strings.ReplaceAll(msg.Amount, ",", "")
	repayAmount, err := math.LegacyNewDecFromStr(amountStr)
	if err != nil {
		return nil, types.ErrInvalidAmountFormat
	}

	if repayAmount.LT(math.LegacyZeroDec()) {
		return nil, types.ErrNegativeAmount
	}

	if repayAmount.Equal(math.LegacyZeroDec()) {
		return nil, types.ErrZeroAmount
	}

	loanValue := k.GetLoanValue(ctx, msg.Denom, msg.Creator)
	repayAmount = math.LegacyMinDec(loanValue, repayAmount)

	if err = k.repay(ctx, ctx.EventManager(), msg.Denom, msg.Creator, repayAmount); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k Keeper) repay(ctx context.Context, eventManager sdk.EventManagerI, denom, address string, repayAmount math.LegacyDec) error {
	acc, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return types.ErrInvalidAddress
	}

	loanIndex, removed := k.updateLoan(ctx, denom, address, repayAmount.Neg())

	coins := sdk.NewCoins(sdk.NewCoin(denom, repayAmount.TruncateInt()))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolVault, coins); err != nil {
		return err
	}

	eventManager.EmitEvent(
		sdk.NewEvent("loan_repaid",
			sdk.Attribute{Key: "address", Value: address},
			sdk.Attribute{Key: "denom", Value: denom},
			sdk.Attribute{Key: "index", Value: strconv.Itoa(int(loanIndex))},
			sdk.Attribute{Key: "amount", Value: repayAmount.String()},
		),
	)

	if removed {
		eventManager.EmitEvent(
			sdk.NewEvent("loan_removed",
				sdk.Attribute{Key: "address", Value: address},
				sdk.Attribute{Key: "denom", Value: denom},
				sdk.Attribute{Key: "index", Value: strconv.Itoa(int(loanIndex))},
			),
		)
	}

	return nil
}
