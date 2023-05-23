package governor

// manualTokenList() returns a list of mainnet tokens that are added manually because they cannot be auto generated.
func manualTokenList() []tokenConfigEntry {
	return []tokenConfigEntry{
		{chain: 2, addr: "000000000000000000000000ef19f4e48830093ce5bc8b3ff7f903a0ae3e9fa1", symbol: "BOTX", coinGeckoId: "botxcoin", decimals: 8, price: 0.02053366},
		{chain: 2, addr: "00000000000000000000000085f17cf997934a597031b2e18a9ab6ebd4b9f6a4", symbol: "NEAR", coinGeckoId: "near", decimals: 8, price: 3.85}, // Near on ethereum
		{chain: 9, addr: "000000000000000000000000e4b9e004389d91e4134a28f19bd833cba1d994b6", symbol: "FRAX", coinGeckoId: "frax", decimals: 8, price: 1.00},
		{chain: 9, addr: "000000000000000000000000c42c30ac6cc15fac9bd938618bcaa1a1fae8501d", symbol: "NEAR", coinGeckoId: "near", decimals: 8, price: 3.85}, // Near on aurora. 24 decimals
		{chain: 12, addr: "0000000000000000000000000000000000000000000100000000000000000002", symbol: "DOT", coinGeckoId: "polkadot", decimals: 8, price: 6.48},
		{chain: 13, addr: "0000000000000000000000005fff3a6c16c2208103f318f4713d4d90601a7313", symbol: "KLEVA", coinGeckoId: "kleva", decimals: 8, price: 0.086661},
		{chain: 13, addr: "0000000000000000000000005096db80b21ef45230c9e423c373f1fc9c0198dd", symbol: "WEMIX", coinGeckoId: "wemix-token", decimals: 8, price: 1.74},
		{chain: 13, addr: "0000000000000000000000005c74070fdea071359b86082bd9f9b3deaafbe32b", symbol: "KDAI", coinGeckoId: "dai", decimals: 8, price: 1.00},
		{chain: 15, addr: "0000000000000000000000000000000000000000000000000000000000000000", symbol: "NEAR", coinGeckoId: "near", decimals: 8, price: 3.85},
		{chain: 19, addr: "017038850bf3af746c36803cce35009268f00d22ae2b55ffb59ac5f2a6add40b", symbol: "INJ", coinGeckoId: "injective-protocol", decimals: 8, price: 1.64},
		{chain: 21, addr: "9258181f5ceac8dbffb7030890243caed69a9599d2886d957a9cb7656af3bdb3", symbol: "Sui", coinGeckoId: "sui", decimals: 9, price: 0.1},
		{chain: 2, addr: "00000000000000000000000018084fba666a33d37592fa2633fd49a74dd93a88", symbol: "tBTC", coinGeckoId: "tbtc", decimals: 18, price: 26553},
		{chain: 32, addr: "00881043998ff2b738519d444d2dd0da3da4545de08290c1076746538d5333df", symbol: "Sei", coinGeckoId: "sei-network", decimals: 6, price: 0.0},
	}
}
