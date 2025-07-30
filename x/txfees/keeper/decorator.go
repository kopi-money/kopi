package keeper

import (
	"bytes"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	"github.com/kopi-money/constants"
	"github.com/kopi-money/kopi/x/txfees/types"
)

// MempoolFeeDecorator will check if the transaction's fee is at least as large
// as the local validator's minimum gasFee (defined in validator config).
// If fee is too low, decorator returns error and tx is rejected from mempool.
// Note this only applies when ctx.CheckTx = true
// If fee is high enough or not CheckTx, then call next AnteHandler
// CONTRACT: Tx must implement FeeTx to use MempoolFeeDecorator.
type MempoolFeeDecorator struct {
	TXFeesKeeper Keeper
	DenomKeeper  types.DenomKeeper
	Opts         types.MempoolFeeOptions
}

func NewMempoolFeeDecorator(txFeesKeeper Keeper, denomKeeper types.DenomKeeper, opts types.MempoolFeeOptions) MempoolFeeDecorator {
	return MempoolFeeDecorator{
		TXFeesKeeper: txFeesKeeper,
		DenomKeeper:  denomKeeper,
		Opts:         opts,
	}
}

func (mfd MempoolFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if ctx.IsCheckTx() && !simulate {
		if feeTx.GetGas() > mfd.Opts.MaxGasWantedPerTx {
			msg := "Too much gas wanted: %d, maximum is %d"
			return ctx, errorsmod.Wrapf(sdkerrors.ErrOutOfGas, msg, feeTx.GetGas(), mfd.Opts.MaxGasWantedPerTx)
		}
	}

	msgs := tx.GetMsgs()
	for _, msg := range msgs {
		// If one of the msgs is an IBC Transfer msg, limit it's size due to current spam potential.
		// 500KB for entire msg
		// 400KB for memo
		// 65KB for receiver
		if transferMsg, ok := msg.(*ibctransfertypes.MsgTransfer); ok {
			if transferMsg.Size() > 500000 { // 500KB
				return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "msg size is too large")
			}

			if len([]byte(transferMsg.Memo)) > 400000 { // 400KB
				return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "memo is too large")
			}

			if len(transferMsg.Receiver) > 65000 { // 65KB
				return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "receiver address is too large")
			}
		}

		// If one of the msgs is from ICA, limit it's size due to current spam potential.
		// 500KB for packet data
		// 65KB for sender
		if icaMsg, ok := msg.(*icacontrollertypes.MsgSendTx); ok {
			if icaMsg.PacketData.Size() > 500000 { // 500KB
				return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "packet data is too large")
			}

			if len([]byte(icaMsg.Owner)) > 65000 { // 65KB
				return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "owner address is too large")
			}
		}
	}

	// If this is genesis height, don't check the fee.
	// This is needed so that gentx's can be created without having to pay a fee (chicken/egg problem).
	if ctx.BlockHeight() == 0 {
		return next(ctx, tx, simulate)
	}

	feeCoins := feeTx.GetFee()

	if len(feeCoins) > 1 {
		return ctx, types.ErrTooManyFeeCoins
	}

	// If there is a fee attached to the tx, make sure the fee denom is a denom accepted by the chain
	if len(feeCoins) == 1 {
		feeDenom := feeCoins.GetDenomByIndex(0)
		if feeDenom != constants.BaseCurrency {
			if !mfd.DenomKeeper.IsFeeDenom(ctx, feeDenom) {
				return ctx, types.ErrInvalidFeeToken
			}
		}
	}

	// Determine if these fees are sufficient for the tx to pass.
	minBaseGasPrice := mfd.getMinBaseGasPrice(ctx, simulate)

	// If minBaseGasPrice is zero, then we don't need to check the fee. Continue
	if minBaseGasPrice.IsZero() {
		return next(ctx, tx, simulate)
	}
	// You should only be able to pay with one fee token in a single tx
	if len(feeCoins) != 1 {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrInsufficientFee,
			"Expected 1 fee denom attached, got %d", len(feeCoins))
	}
	// The minimum base gas price is in uosmo, convert the fee denom's worth to uosmo terms.
	// Then compare if its sufficient for paying the tx fee.
	err = mfd.TXFeesKeeper.IsSufficientFee(ctx, minBaseGasPrice, feeTx.GetGas(), feeCoins[0])
	if err != nil {
		return ctx, err
	}

	return next(ctx, tx, simulate)
}

func (mfd MempoolFeeDecorator) getMinBaseGasPrice(ctx sdk.Context, simulate bool) math.LegacyDec {
	minBaseGasPrice := types.ConsensusMinFee
	if ctx.BlockHeight() == 0 || simulate {
		minBaseGasPrice = math.LegacyZeroDec()
	}
	return minBaseGasPrice
}

// IsSufficientFee checks if the feeCoin provided (in any asset), is worth enough osmo at current spot prices
// to pay the gas cost of this tx.
func (k Keeper) IsSufficientFee(ctx sdk.Context, minBaseGasPrice math.LegacyDec, gasRequested uint64, feeCoin sdk.Coin) error {
	// Determine the required fees by multiplying the required minimum gas
	// price by the gas limit, where fee = ceil(minGasPrice * gasLimit).
	// note we mutate this one line below, to avoid extra heap allocations.
	glDec := math.LegacyNewDec(int64(gasRequested))
	baseFeeAmt := glDec.MulMut(minBaseGasPrice).Ceil().RoundInt()
	baseFeeCoin := sdk.NewCoin(constants.BaseCurrency, baseFeeAmt)

	convertedFee, err := k.denomKeeper.GetValueInBase(ctx, feeCoin.Denom, feeCoin.Amount.ToLegacyDec())
	if err != nil {
		return err
	}
	convertedFeeCoin := sdk.NewCoin(constants.BaseCurrency, convertedFee.RoundInt())

	// check to ensure that the convertedFee should always be greater than or equal to the requireBaseFee
	if !(convertedFeeCoin.IsGTE(baseFeeCoin)) {
		return errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %s which converts to %s. required: %s", feeCoin, convertedFeeCoin, baseFeeCoin)
	}

	return nil
}

// DeductFeeDecorator deducts fees from the first signer of the tx.
// If the first signer does not have the funds to pay for the fees, we return an InsufficientFunds error.
// We call next AnteHandler if fees successfully deducted.
//
// CONTRACT: Tx must implement FeeTx interface to use DeductFeeDecorator
type DeductFeeDecorator struct {
	ak             types.AccountKeeper
	bankKeeper     types.BankKeeper
	feegrantKeeper types.FeegrantKeeper
	txFeesKeeper   Keeper
}

func NewDeductFeeDecorator(tk Keeper, ak types.AccountKeeper, bk types.BankKeeper, fk types.FeegrantKeeper) DeductFeeDecorator {
	return DeductFeeDecorator{
		ak:             ak,
		bankKeeper:     bk,
		feegrantKeeper: fk,
		txFeesKeeper:   tk,
	}
}

func (dfd DeductFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	// checks to make sure the auth module account has been set to collect tx fees in base token, to be used for staking rewards
	if addr := dfd.ak.GetModuleAddress(authtypes.FeeCollectorName); addr == nil {
		return ctx, fmt.Errorf("fee collector module account (%s) has not been set", authtypes.FeeCollectorName)
	}

	// fee can be in any denom (checked for validity later)
	fee := feeTx.GetFee()
	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()

	// set the fee payer as the default address to deduct fees from
	deductFeesFrom := feePayer

	// If a fee granter was set, deduct fee from the fee granter's account.
	if feeGranter != nil {
		if dfd.feegrantKeeper == nil {
			return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "fee grants is not enabled")
		} else if !bytes.Equal(feeGranter, feePayer) {
			if err = dfd.feegrantKeeper.UseGrantedFees(ctx, feeGranter, feePayer, fee, tx.GetMsgs()); err != nil {
				return ctx, errorsmod.Wrapf(err, "%s not allowed to pay fees from %s", feeGranter, feePayer)
			}
		}

		// if no errors, change the account charged for fees to the fee granter
		deductFeesFrom = feeGranter
	}

	deductFeesFromAcc := dfd.ak.GetAccount(ctx, deductFeesFrom)
	if deductFeesFromAcc == nil {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrUnknownAddress, "fee payer address: %s does not exist", deductFeesFrom)
	}

	fees := feeTx.GetFee()

	// deducts the fees and transfer them to the module account
	if !fees.IsZero() {
		if err = DeductFees(dfd.txFeesKeeper, dfd.bankKeeper, ctx, deductFeesFromAcc, fees); err != nil {
			return ctx, err
		}
	}

	ctx.EventManager().EmitEvents(sdk.Events{sdk.NewEvent(sdk.EventTypeTx,
		sdk.NewAttribute(sdk.AttributeKeyFee, fees.String()),
	)})

	return next(ctx, tx, simulate)
}

// DeductFees deducts fees from the given account and transfers them to the set module account.
func DeductFees(denomKeeper Keeper, bankKeeper types.BankKeeper, ctx sdk.Context, acc sdk.AccountI, fees sdk.Coins) error {
	// Checks the validity of the fee tokens (sorted, have positive amount, valid and unique denomination)
	if !fees.IsValid() {
		return errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	// checks if input fee is ukopi (assumes only one fee token exists in the fees array (as per the check in mempoolFeeDecorator))
	if fees[0].Denom == constants.BaseCurrency {
		if err := denomKeeper.bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), authtypes.FeeCollectorName, fees); err != nil {
			return fmt.Errorf("insufficient funds: %w", err)
		}
	} else {
		if err := bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), types.ModuleName, fees); err != nil {
			return fmt.Errorf("insufficient funds: %w", err)
		}
	}

	return nil
}
