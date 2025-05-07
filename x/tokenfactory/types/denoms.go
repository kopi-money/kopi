package types

func (fd FactoryDenom) FactoryTradeDenom() string {
	if fd.LocalName != "" {
		return fd.LocalName
	}

	return fd.FullName
}

func (fd FactoryDenom) ReplaceWithFactoryTradeDenom(denom string) string {
	if denom == fd.FullName {
		return fd.FactoryTradeDenom()
	}

	return denom
}
