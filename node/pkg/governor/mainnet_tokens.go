package governor

func TokenList() []TokenConfigEntry {
	return append(manualTokenList(), generatedMainnetTokenList()...)
}
