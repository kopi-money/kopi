package types

func (d *DiscountLevel) Equal(other *DiscountLevel) bool {
	if d == nil {
		return other == nil
	} else if other == nil {
		return false
	}

	if !d.TradeAmount.Equal(other.TradeAmount) {
		return false
	}

	if !d.Discount.Equal(other.Discount) {
		return false
	}

	return true
}
