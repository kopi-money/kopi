package types

func (rsa ReserveSellAmount) Equal(other *ReserveSellAmount) bool {
	if other == nil {
		return false
	}

	if rsa.Denom != other.Denom {
		return false
	}

	if !rsa.SellAmount.Equal(other.SellAmount) {
		return false
	}

	return true
}
