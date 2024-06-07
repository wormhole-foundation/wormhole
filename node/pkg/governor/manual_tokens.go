package governor

// manualTokenList() returns a list of mainnet tokens that are added manually because they cannot be auto generated.
func manualTokenList() []tokenConfigEntry {
	return []tokenConfigEntry{
		{chain: 2, addr: "000000000000000000000000ef19f4e48830093ce5bc8b3ff7f903a0ae3e9fa1", symbol: "BOTX", coinGeckoId: "botxcoin", decimals: 8, price: 0.02053366},
		{chain: 2, addr: "00000000000000000000000085f17cf997934a597031b2e18a9ab6ebd4b9f6a4", symbol: "NEAR", coinGeckoId: "near", decimals: 8, price: 3.85},    // Near on ethereum
		{chain: 8, addr: "000000000000000000000000000000000000000000000000000000000004c5c1", symbol: "USDt", coinGeckoId: "tether", decimals: 6, price: 1.002}, // Addr: 312769, Notional: 22.31747085
		{chain: 9, addr: "000000000000000000000000e4b9e004389d91e4134a28f19bd833cba1d994b6", symbol: "FRAX", coinGeckoId: "frax", decimals: 8, price: 1.00},
		{chain: 9, addr: "000000000000000000000000c42c30ac6cc15fac9bd938618bcaa1a1fae8501d", symbol: "NEAR", coinGeckoId: "near", decimals: 8, price: 3.85}, // Near on aurora. 24 decimals
		{chain: 12, addr: "0000000000000000000000000000000000000000000100000000000000000002", symbol: "DOT", coinGeckoId: "polkadot", decimals: 8, price: 6.48},
		{chain: 13, addr: "0000000000000000000000005fff3a6c16c2208103f318f4713d4d90601a7313", symbol: "KLEVA", coinGeckoId: "kleva", decimals: 8, price: 0.086661},
		{chain: 13, addr: "0000000000000000000000005096db80b21ef45230c9e423c373f1fc9c0198dd", symbol: "WEMIX", coinGeckoId: "wemix-token", decimals: 8, price: 1.74},
		{chain: 15, addr: "0000000000000000000000000000000000000000000000000000000000000000", symbol: "NEAR", coinGeckoId: "near", decimals: 8, price: 3.85},
		{chain: 32, addr: "00881043998ff2b738519d444d2dd0da3da4545de08290c1076746538d5333df", symbol: "Sei", coinGeckoId: "sei-network", decimals: 6, price: 0.0},
		// BLAST (tokens over $50,000 24h volume)
		{chain: 36, addr: "0000000000000000000000004300000000000000000000000000000000000003", symbol: "USDB", coinGeckoId: "usdb", decimals: 18, price: 1.00},
		{chain: 36, addr: "0000000000000000000000004300000000000000000000000000000000000004", symbol: "WETH", coinGeckoId: "weth", decimals: 18, price: 3157.42},
		{chain: 36, addr: "0000000000000000000000002416092f143378750bb29b79ed961ab195cceea5", symbol: "EZETH", coinGeckoId: "renzo-restaked-eth", decimals: 18, price: 3092.32},
		{chain: 36, addr: "0000000000000000000000004fee793d435c6d2c10c135983bb9d6d4fc7b9bbd", symbol: "USD+", coinGeckoId: "usd", decimals: 18, price: 1.00},
		{chain: 36, addr: "000000000000000000000000818a92bc81aad0053d72ba753fb5bc3d0c5c0923", symbol: "JUICE", coinGeckoId: "juice-finance", decimals: 18, price: 0.1051},
		{chain: 36, addr: "0000000000000000000000009e20461bc2c4c980f62f1b279d71734207a6a356", symbol: "OMNI", coinGeckoId: "omnicat", decimals: 18, price: 0.0004575},
		{chain: 36, addr: "000000000000000000000000764933fbad8f5d04ccd088602096655c2ed9879f", symbol: "AI", coinGeckoId: "any-inu", decimals: 18, price: 0.00002742},
		{chain: 36, addr: "0000000000000000000000005ffd9ebd27f2fcab044c0f0a26a45cb62fa29c06", symbol: "PAC", coinGeckoId: "pacmoon", decimals: 18, price: 0.05459},
		{chain: 36, addr: "00000000000000000000000020fe91f17ec9080e3cac2d688b4ecb48c5ac3a9c", symbol: "YES", coinGeckoId: "yes-money", decimals: 18, price: 3.96},
		{chain: 36, addr: "00000000000000000000000076da31d7c9cbeae102aff34d3398bc450c8374c1", symbol: "MIM", coinGeckoId: "magic-internet-money", decimals: 18, price: 0.9935},
		{chain: 36, addr: "00000000000000000000000015d24de366f69b835be19f7cf9447e770315dd80", symbol: "KAP", coinGeckoId: "kapital-dao", decimals: 18, price: 0.1143},
		{chain: 36, addr: "000000000000000000000000b9dfcd4cf589bb8090569cb52fac1b88dbe4981f", symbol: "BAG", coinGeckoId: "bag", decimals: 18, price: 0.002972},
		{chain: 36, addr: "00000000000000000000000068449870eea84453044bd430822827e21fd8f101", symbol: "ZAI", coinGeckoId: "zaibot", decimals: 18, price: 0.2348},
		{chain: 36, addr: "00000000000000000000000047c337bd5b9344a6f3d6f58c474d9d8cd419d8ca", symbol: "DACKIE", coinGeckoId: "dackieswap", decimals: 18, price: 0.006554},
		{chain: 36, addr: "000000000000000000000000d43d8adac6a4c7d9aeece7c3151fca8f23752cf8", symbol: "ANDY", coinGeckoId: "andyerc", decimals: 9, price: 0.1165},
		{chain: 36, addr: "00000000000000000000000087e154e86fb691ab8a27116e93ed8d54e2b8c18c", symbol: "TES", coinGeckoId: "titan-trading-token", decimals: 18, price: 0.867},
		{chain: 36, addr: "000000000000000000000000870a8f46b62b8bdeda4c02530c1750cddf2ed32e", symbol: "USDC+", coinGeckoId: "usdc-plus-overnight", decimals: 18, price: 1.00},
		{chain: 36, addr: "00000000000000000000000042e12d42b3d6c4a74a88a61063856756ea2db357", symbol: "ORBIT", coinGeckoId: "orbit-protocol", decimals: 18, price: 0.3074},
		// SCROLL (tokens over $50,000 24h volume)
		{chain: 34, addr: "0000000000000000000000000018d96c579121a94307249d47f053e2d687b5e7", symbol: "MVX", coinGeckoId: "metavault-trade", decimals: 18, price: 2.06},
		{chain: 34, addr: "00000000000000000000000047c337bd5b9344a6f3d6f58c474d9d8cd419d8ca", symbol: "DACKIE", coinGeckoId: "dackieswap", decimals: 18, price: 0.00655},
		{chain: 34, addr: "0000000000000000000000005300000000000000000000000000000000000004", symbol: "WETH", coinGeckoId: "bridged-wrapped-ether-scroll", decimals: 18, price: 3145.98},
		{chain: 34, addr: "000000000000000000000000f55bec9cafdbe8730f096aa55dad6d22d44099df", symbol: "USDT", coinGeckoId: "bridged-tether-scroll", decimals: 6, price: 1.00},
		{chain: 34, addr: "00000000000000000000000006efdbff2a14a7c8e15944d1f4a48f9f95f663a4", symbol: "USDC", coinGeckoId: "bridged-usd-coin-scroll", decimals: 6, price: 1.00},
		{chain: 34, addr: "000000000000000000000000eb466342c4d449bc9f53a865d5cb90586f405215", symbol: "AXLUSDC", coinGeckoId: "bridged-axelar-wrapped-usd-coin-scroll", decimals: 6, price: 1.01},
		{chain: 34, addr: "0000000000000000000000003c1bca5a656e69edcd0d4e36bebb3fcdaca60cf1", symbol: "WBTC", coinGeckoId: "bridged-wrapped-bitcoin-scroll", decimals: 8, price: 64415.17},
		{chain: 34, addr: "00000000000000000000000060d01ec2d5e98ac51c8b4cf84dfcce98d527c747", symbol: "IZI", coinGeckoId: "izumi-finance", decimals: 18, price: 0.0142},
		{chain: 34, addr: "0000000000000000000000000a3bb08b3a15a19b4de82f8acfc862606fb69a2d", symbol: "IUSD", coinGeckoId: "izumi-bond-usd", decimals: 18, price: 0.9195},
		{chain: 34, addr: "000000000000000000000000f610a9dfb7c89644979b4a0f27063e9e7d7cda32", symbol: "WSTETH", coinGeckoId: "bridged-wrapped-lido-staked-ether-scroll", decimals: 18, price: 3659.28},
		{chain: 34, addr: "000000000000000000000000cA77eB3fEFe3725Dc33bccB54eDEFc3D9f764f97", symbol: "DAI", coinGeckoId: "dai", decimals: 18, price: 1.00},
		{chain: 34, addr: "00000000000000000000000053878B874283351D26d206FA512aEcE1Bef6C0dD", symbol: "RETH", coinGeckoId: "rocket-pool-eth", decimals: 18, price: 3475.55},
		// X LAYER (tokens over $50,000 24h volume)
		{chain: 37, addr: "0000000000000000000000001e4a5963abfd975d8c9021ce480b42188849d41d", symbol: "USDT", coinGeckoId: "polygon-hermez-bridged-usdt-x-layer", decimals: 6, price: 0.9969},
		{chain: 37, addr: "000000000000000000000000e538905cf8410324e03a5a23c1c177a474d59b2b", symbol: "WOKB", coinGeckoId: "wrapped-okb", decimals: 18, price: 48.76},
		{chain: 37, addr: "0000000000000000000000005a77f1443d16ee5761d310e38b62f77f726bc71c", symbol: "WETH", coinGeckoId: "weth", decimals: 18, price: 2994.60},
		{chain: 37, addr: "00000000000000000000000074b7f16337b8972027f6196a17a631ac6de26d22", symbol: "USDC", coinGeckoId: "polygon-hermez-bridged-usdc-x-layer", decimals: 6, price: 0.9949},
		{chain: 37, addr: "000000000000000000000000ea034fb02eb1808c2cc3adbc15f447b93cbe08e1", symbol: "WBTC", coinGeckoId: "polygon-hermez-bridged-wbtc-x-layer", decimals: 8, price: 57029},
		{chain: 37, addr: "000000000000000000000000c5015b9d9161dca7e18e32f6f25c4ad850731fd4", symbol: "DAI", coinGeckoId: "polygon-hermez-bridged-dai-x-layer", decimals: 18, price: 1.0006},
		// MANTLE (tokens over $50,000 24h volume)
		{chain: 35, addr: "000000000000000000000000deaddeaddeaddeaddeaddeaddeaddeaddead0000", symbol: "MNT", coinGeckoId: "mantle", decimals: 18, price: 1.01},
		{chain: 35, addr: "00000000000000000000000078c1b0c915c4faa5fffa6cabf0219da63d7f4cb8", symbol: "WMNT", coinGeckoId: "wrapped-mantle", decimals: 18, price: 1.01},
		{chain: 35, addr: "00000000000000000000000009bc4e0d864854c6afb6eb9a9cdf58ac190d0df9", symbol: "USDC", coinGeckoId: "mantle-bridged-usdc-mantle", decimals: 6, price: 1},
		{chain: 35, addr: "000000000000000000000000201EBa5CC46D216Ce6DC03F6a759e8E766e956aE", symbol: "USDT", coinGeckoId: "mantle-bridged-usdt-mantle", decimals: 6, price: 0.9973},
		{chain: 35, addr: "000000000000000000000000cDA86A272531e8640cD7F1a92c01839911B90bb0", symbol: "METH", coinGeckoId: "mantle-staked-ether", decimals: 18, price: 3934.06},
		{chain: 35, addr: "000000000000000000000000deaddeaddeaddeaddeaddeaddeaddeaddead1111", symbol: "WETH", coinGeckoId: "wrapped-ether-mantle-bridge", decimals: 18, price: 3825.65},
		{chain: 35, addr: "000000000000000000000000371c7ec6d8039ff7933a2aa28eb827ffe1f52f07", symbol: "JOE", coinGeckoId: "joe", decimals: 18, price: 0.4911},
	}
}
