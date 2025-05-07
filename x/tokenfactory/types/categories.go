package types

func (c *Categories) Equal(c2 *Categories) bool {
	if len(c.Categories) != len(c2.Categories) {
		return false
	}

	for i := range c.Categories {
		if c.Categories[i].Equal(c2.Categories[i]) {
			return false
		}
	}

	return true
}

func (c *Category) Equal(c2 Category) bool {
	if c.Index != c2.Index {
		return false
	}

	return true
}
