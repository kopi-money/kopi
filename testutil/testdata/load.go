package testdata

import (
	"encoding/json"
	"github.com/kopi-money/kopi/data"
	"github.com/pkg/errors"
	"sort"
	"strconv"
)

type LiquidityEntry struct {
	Address string `json:"address"`
	Amount  string `json:"amount"`
	Denom   string `json:"denom"`
	Index   string `json:"index"`
}

func LoadLiquidity() ([]LiquidityEntry, error) {
	var payload []LiquidityEntry
	if err := json.Unmarshal([]byte(data.LiquidityEntries), &payload); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	sort.Slice(payload, func(i, j int) bool {
		index1, _ := strconv.Atoi(payload[i].Index)
		index2, _ := strconv.Atoi(payload[j].Index)

		return index1 < index2
	})

	return payload, nil
}

type Order struct {
	Index           string `json:"index"`
	Creator         string `json:"creator"`
	DenomFrom       string `json:"denom_from"`
	DenomTo         string `json:"denom_to"`
	AmountGiven     string `json:"amount_given"`
	AmountLeft      string `json:"amount_left"`
	TradeAmount     string `json:"trade_amount"`
	MaxPrice        string `json:"max_price"`
	NumBlocks       string `json:"num_blocks"`
	AllowIncomplete bool   `json:"allow_incomplete"`
}

func LoadOrders() ([]Order, error) {
	var payload []Order
	if err := json.Unmarshal([]byte(data.Orders), &payload); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	sort.Slice(payload, func(i, j int) bool {
		index1, _ := strconv.Atoi(payload[i].Index)
		index2, _ := strconv.Atoi(payload[j].Index)

		return index1 < index2
	})

	return payload, nil
}
