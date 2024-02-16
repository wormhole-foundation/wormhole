package governor

func tokenList() []tokenConfigEntry {
	allTokens := append(manualTokenList(), generatedMainnetTokenList()...)
	flowCancelTokens := flowCancelTokenList()

	// Modify tokens to be flow cancelling by rewriting the allTokens slice.
	// Existing entries that should be flow-cancelled are overwritten by entries
	// from flowCancelTokens.
	// Equivalent to setting the tokenConfigEntry field `flowCancels` to `true`.
	finalTokens := allTokens[:0] // reuse already allocated storage. https://go.dev/wiki/SliceTricks#filtering-without-allocating
	outer:
	for _, token := range allTokens {
		// Lookup each token against the allowlist of flow cancelling tokens
		for _, flowCancelToken := range flowCancelTokens {
			newToken := token
			newToken.flowCancels = false // ensure that the field is explicitly set to false

			// Check if the tokens are equal (excluding price)
			if token.symbol == flowCancelToken.symbol &&
			token.coinGeckoId == flowCancelToken.coinGeckoId &&
			token.addr == flowCancelToken.addr &&
			token.chain == flowCancelToken.chain &&
			token.decimals == flowCancelToken.decimals {
				newToken.flowCancels = true
				finalTokens = append(finalTokens, newToken)
				continue outer // skip last line of outer loop
			}
		}
		// Simply re-add the token if it should not cancel flows
		finalTokens = append(finalTokens, token)
	}
	
	return finalTokens
}
