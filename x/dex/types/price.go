package types

func (mp *MaxPrice) getMaxPrice() string {
	if mp == nil {
		return ""
	}

	return mp.MaxPrice
}
