package keeper

import (
	"context"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

type TradeBalances struct {
	senders   TransferAmounts
	receivers TransferAmounts
}

func NewTradeBalances() *TradeBalances {
	return &TradeBalances{
		senders:   TransferAmounts{transferAmounts: make(map[transferKey]math.Int)},
		receivers: TransferAmounts{transferAmounts: make(map[transferKey]math.Int)},
	}
}

type transferKey struct {
	address string
	denom   string
}

type TransferAmounts struct {
	transferAmounts map[transferKey]math.Int
}

func (td *TransferAmounts) add(address, denom string, amount math.Int) {
	key := transferKey{address, denom}
	detail, has := td.transferAmounts[key]
	if !has {
		detail = math.ZeroInt()
	}

	td.transferAmounts[key] = detail.Add(amount)
}

func (td *TransferAmounts) sub(key transferKey, amount math.Int) {
	detail, has := td.transferAmounts[key]
	if has {
		detail = detail.Sub(amount)
		if detail.IsZero() {
			delete(td.transferAmounts, key)
		} else {
			td.transferAmounts[key] = detail
		}
	}
}

func (td *TransferAmounts) next(denom string) (transferKey, math.Int, bool) {
	for key, amount := range td.transferAmounts {
		if key.denom == denom {
			return key, amount, true
		}
	}

	return transferKey{}, math.Int{}, false
}

type Transfer struct {
	From   string
	To     string
	Denom  string
	Amount math.Int
}

func (t *Transfer) add(amount math.Int) {
	t.Amount = t.Amount.Add(amount)
}

func (t *Transfer) equals(other Transfer) bool {
	return t.To == other.To && t.From == other.From && t.Denom == other.Denom
}

type Transfers []*Transfer

func (t *Transfers) add(newTransfer Transfer) {
	seen := false
	for index, transfer := range *t {
		if transfer.equals(newTransfer) {
			(*t)[index].add(newTransfer.Amount)
			seen = true
			break
		}
	}

	if !seen {
		*t = append(*t, &newTransfer)
	}
}

func (tb *TradeBalances) AddTransfer(from, to, denom string, amount math.Int) {
	if amount.IsNil() {
		fmt.Println()
	}

	tb.senders.add(from, denom, amount)
	tb.receivers.add(to, denom, amount)
}

func (tb *TradeBalances) printBalance(denom string) {
	fmt.Println(fmt.Sprintf("--- %v", denom))

	sendSum := math.ZeroInt()
	for key, sendAmount := range tb.senders.transferAmounts {
		if key.denom == denom {
			sendSum = sendSum.Add(sendAmount)
		}
	}
	fmt.Println(fmt.Sprintf("send:\t%v", sendSum.String()))

	receiveSum := math.ZeroInt()
	for key, receiveAmount := range tb.receivers.transferAmounts {
		if key.denom == denom {
			receiveSum = receiveSum.Add(receiveAmount)
		}
	}
	fmt.Println(fmt.Sprintf("receive:\t%v", receiveSum.String()))
}

func (tb *TradeBalances) NetBalance(acc, denom string) math.Int {
	sum := math.ZeroInt()
	key := transferKey{acc, denom}

	receive, has := tb.receivers.transferAmounts[key]
	if has {
		sum = sum.Add(receive)
	}

	send, has := tb.senders.transferAmounts[key]
	if has {
		sum = sum.Sub(send)
	}

	return sum
}

func (tb *TradeBalances) Settle(ctx context.Context, bank types.BankKeeper) error {
	transfers, err := tb.MergeTransfers()
	if err != nil {
		return errors.Wrap(err, "could not merge transfers")
	}

	var accFrom, accTo sdk.AccAddress
	for _, transfer := range transfers {
		coins := sdk.NewCoins(sdk.NewCoin(transfer.Denom, transfer.Amount))
		accFrom, err = sdk.AccAddressFromBech32(transfer.From)
		if err != nil {
			return errors.Wrap(err, "invalid from address")
		}

		accTo, err = sdk.AccAddressFromBech32(transfer.To)
		if err != nil {
			return errors.Wrap(err, "invalid to address")
		}

		if err = bank.SendCoins(ctx, accFrom, accTo, coins); err != nil {
			return errors.Wrap(err, "could not send coins")
		}
	}

	return nil
}

func (tb *TradeBalances) MergeTransfers() (Transfers, error) {
	var transfers Transfers

	// First we check whether this receiver also has to send something in the same denom. If yes, both amounts
	// entries cancel each other. One of each is removed, in the case where both entries are of the same amount both
	// are removed.
	for receiverKey, receiveAmount := range tb.receivers.transferAmounts {
		sendAmount, has := tb.senders.transferAmounts[receiverKey]
		if has {
			amount := math.MinInt(sendAmount, receiveAmount)
			tb.senders.sub(receiverKey, amount)
			tb.receivers.sub(receiverKey, amount)
		}
	}

	for receiverKey, receiveAmountLeft := range tb.receivers.transferAmounts {
		for senderKey, sendAmount := range tb.senders.transferAmounts {
			if receiveAmountLeft.IsZero() {
				break
			}

			if senderKey.denom != receiverKey.denom {
				continue
			}

			amount := math.MinInt(sendAmount, receiveAmountLeft)
			receiveAmountLeft = receiveAmountLeft.Sub(amount)
			tb.receivers.sub(receiverKey, amount)
			tb.senders.sub(senderKey, amount)

			transfers.add(Transfer{
				From:   senderKey.address,
				To:     receiverKey.address,
				Denom:  receiverKey.denom,
				Amount: amount,
			})
		}

		if receiveAmountLeft.GT(math.ZeroInt()) {
			return nil, fmt.Errorf("could not fullfill receiver request")
		}
	}

	if len(tb.senders.transferAmounts) > 0 {
		return nil, fmt.Errorf("unused senders left")
	}

	return transfers, nil
}
