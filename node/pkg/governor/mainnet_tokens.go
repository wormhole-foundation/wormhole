package governor

func tokenList() []tokenConfigEntry {
	return append(manualTokenList(), generatedMainnetTokenList()...)
}
