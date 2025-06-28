package types

func (c *Categories) Equal(c2 *Categories) bool {
	if len(c.Categories) != len(c2.Categories) {
		return false
	}

	for i := range c.Categories {
		if !c.Categories[i].Equal(c2.Categories[i]) {
			return false
		}
	}

	return true
}

func (c *Category) Equal(c2 Category) bool {
	return c.Index == c2.Index &&
		c.Name == c2.Name &&
		c.IsIbc == c2.IsIbc &&
		c.CreationPrice.Equal(c2.CreationPrice)
}
