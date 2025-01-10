package keeper

import (
	"context"
	"fmt"
	"sort"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

type transferKey struct {
	address string
	denom   string
}

type transferAmount struct {
	address string
	denom   string
	amount  math.Int
}

type transferAmounts []transferAmount

func (ta transferAmounts) remove(indexes []int) transferAmounts {
	for len(indexes) > 0 {
		index := indexes[len(indexes)-1]
		indexes = indexes[:len(indexes)-1]
		ta = append(ta[:index], ta[index+1:]...)
	}

	return ta
}

type TransferAmounts struct {
	transferAmounts map[transferKey]math.Int
}

func (ta *TransferAmounts) toSlice() (list transferAmounts) {
	for key, amount := range ta.transferAmounts {
		list = append(list, transferAmount{
			address: key.address,
			denom:   key.denom,
			amount:  amount,
		})
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].denom != list[j].denom {
			return list[i].denom < list[j].denom
		}

		return list[i].address < list[j].address
	})

	return list
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
	for index, transfer := range *t {
		if transfer.equals(newTransfer) {
			(*t)[index].add(newTransfer.Amount)
			return
		}
	}

	*t = append(*t, &newTransfer)
}

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

func (tb *TradeBalances) Merge(other *TradeBalances) {
	for sender, amount := range other.senders.transferAmounts {
		tb.senders.add(sender.address, sender.denom, amount)
	}

	for sender, amount := range other.receivers.transferAmounts {
		tb.receivers.add(sender.address, sender.denom, amount)
	}
}

func (tb *TradeBalances) AddTransfer(from, to, denom string, amount math.Int) {
	if amount.IsPositive() {
		tb.senders.add(from, denom, amount)
		tb.receivers.add(to, denom, amount)
	}
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

// Settle first merges all the receives and sends before then executing them. It can happen that two wallets get paired
// that otherwise wouldn't have interacted with each other, but that doesn't matter in this case because we only care
// about removing the correct amount of funds from the correct wallets and send it to wallets that are eligable.
func (tb *TradeBalances) Settle(ctx context.Context, bank types.Sender) error {
	transfers, err := tb.MergeTransfers()
	if err != nil {
		return fmt.Errorf("could not merge transfers: %w", err)
	}

	var accFrom, accTo sdk.AccAddress
	for _, transfer := range transfers {
		coins := sdk.NewCoins(sdk.NewCoin(transfer.Denom, transfer.Amount))
		accFrom, err = sdk.AccAddressFromBech32(transfer.From)
		if err != nil {
			return fmt.Errorf("invalid from address: %w", err)
		}

		accTo, err = sdk.AccAddressFromBech32(transfer.To)
		if err != nil {
			return fmt.Errorf("invalid to address: %w", err)
		}

		if err = bank.SendCoins(ctx, accFrom, accTo, coins); err != nil {
			return fmt.Errorf("could not send coins from %v: %w", accFrom.String(), err)
		}
	}

	return nil
}

// MergeTransfers combines the transfer details of different addresses. Send and receives requests are cancelling each
// other so as to minimize to total number of sends.
func (tb *TradeBalances) MergeTransfers() (Transfers, error) {
	var transfers Transfers

	// First, we check whether this receiver also has to send something in the same denom. If yes, both amounts
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

	receivers := tb.receivers.toSlice()
	senders := tb.senders.toSlice()

	// Second, we combine receives with sends to create transfers. If a receive request could not be combined with
	// enough sends, it means there was a problem. This should never happen and has been tested in unit tests.
	for _, receiver := range receivers {
		var deleteIndexes []int

		for senderIndex, sender := range senders {
			if receiver.amount.IsZero() {
				break
			}

			if sender.denom != receiver.denom {
				continue
			}

			amount := math.MinInt(sender.amount, receiver.amount)
			receiver.amount = receiver.amount.Sub(amount)

			if sender.amount.Equal(amount) {
				deleteIndexes = append(deleteIndexes, senderIndex)
			} else {
				senders[senderIndex] = transferAmount{
					address: sender.address,
					denom:   sender.denom,
					amount:  sender.amount.Sub(amount),
				}
			}

			transfers.add(Transfer{
				From:   sender.address,
				To:     receiver.address,
				Denom:  receiver.denom,
				Amount: amount,
			})
		}

		senders = senders.remove(deleteIndexes)
		if receiver.amount.GT(math.ZeroInt()) {
			return nil, fmt.Errorf("could not fullfill receiver request")
		}
	}

	// This should never happen, has been tested in unit tests.
	if len(senders) > 0 {
		return nil, fmt.Errorf("unused senders left")
	}

	return transfers, nil
}
