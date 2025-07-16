package governor

// manualTokenList() returns a list of mainnet tokens that are added manually because they cannot be auto generated.
func manualTokenList() []TokenConfigEntry {
	return []TokenConfigEntry{
		{Chain: 2, Addr: "0000000000000000000000006c3ea9036406852006290770bedfcaba0e23a0e8", Symbol: "PYUSD", CoinGeckoId: "paypal-usd", Decimals: 6, Price: 1.00},
		{Chain: 2, Addr: "00000000000000000000000085f17cf997934a597031b2e18a9ab6ebd4b9f6a4", Symbol: "NEAR", CoinGeckoId: "near", Decimals: 24, Price: 4.34},   // Near on ethereum
		{Chain: 8, Addr: "000000000000000000000000000000000000000000000000000000000004c5c1", Symbol: "USDt", CoinGeckoId: "tether", Decimals: 6, Price: 1.002}, // Addr: 312769, Notional: 22.31747085
		{Chain: 12, Addr: "0000000000000000000000000000000000000000000100000000000000000002", Symbol: "DOT", CoinGeckoId: "polkadot", Decimals: 10, Price: 6.48},
		{Chain: 12, Addr: "0000000000000000000000000000000000000000000100000000000000000000", Symbol: "ACA", CoinGeckoId: "acala", Decimals: 12, Price: 0.05694},
		{Chain: 13, Addr: "0000000000000000000000005fff3a6c16c2208103f318f4713d4d90601a7313", Symbol: "KLEVA", CoinGeckoId: "kleva", Decimals: 18, Price: 0.086661},
		{Chain: 13, Addr: "0000000000000000000000005096db80b21ef45230c9e423c373f1fc9c0198dd", Symbol: "WEMIX", CoinGeckoId: "wemix-token", Decimals: 18, Price: 1.74},
		{Chain: 15, Addr: "0000000000000000000000000000000000000000000000000000000000000000", Symbol: "NEAR", CoinGeckoId: "near", Decimals: 24, Price: 4.34},
		{Chain: 32, Addr: "01881043998ff2b738519d444d2dd0da3da4545de08290c1076746538d5333df", Symbol: "SEI", CoinGeckoId: "sei-network", Decimals: 6, Price: 0.3},
		// BLAST (tokens over $50,000 24h volume)
		{Chain: 36, Addr: "0000000000000000000000004300000000000000000000000000000000000003", Symbol: "USDB", CoinGeckoId: "usdb", Decimals: 18, Price: 1.00},
		{Chain: 36, Addr: "0000000000000000000000004300000000000000000000000000000000000004", Symbol: "WETH", CoinGeckoId: "weth", Decimals: 18, Price: 3157.42},
		{Chain: 36, Addr: "0000000000000000000000002416092f143378750bb29b79ed961ab195cceea5", Symbol: "EZETH", CoinGeckoId: "renzo-restaked-eth", Decimals: 18, Price: 3092.32},
		{Chain: 36, Addr: "0000000000000000000000004fee793d435c6d2c10c135983bb9d6d4fc7b9bbd", Symbol: "USD+", CoinGeckoId: "usd", Decimals: 18, Price: 1.00},
		{Chain: 36, Addr: "000000000000000000000000818a92bc81aad0053d72ba753fb5bc3d0c5c0923", Symbol: "JUICE", CoinGeckoId: "juice-finance", Decimals: 18, Price: 0.1051},
		{Chain: 36, Addr: "0000000000000000000000009e20461bc2c4c980f62f1b279d71734207a6a356", Symbol: "OMNI", CoinGeckoId: "omnicat", Decimals: 18, Price: 0.0004575},
		{Chain: 36, Addr: "000000000000000000000000764933fbad8f5d04ccd088602096655c2ed9879f", Symbol: "AI", CoinGeckoId: "any-inu", Decimals: 18, Price: 0.00002742},
		{Chain: 36, Addr: "0000000000000000000000005ffd9ebd27f2fcab044c0f0a26a45cb62fa29c06", Symbol: "PAC", CoinGeckoId: "pacmoon", Decimals: 18, Price: 0.05459},
		{Chain: 36, Addr: "00000000000000000000000020fe91f17ec9080e3cac2d688b4ecb48c5ac3a9c", Symbol: "YES", CoinGeckoId: "yes-money", Decimals: 18, Price: 3.96},
		{Chain: 36, Addr: "00000000000000000000000076da31d7c9cbeae102aff34d3398bc450c8374c1", Symbol: "MIM", CoinGeckoId: "magic-internet-money", Decimals: 18, Price: 0.9935},
		{Chain: 36, Addr: "00000000000000000000000015d24de366f69b835be19f7cf9447e770315dd80", Symbol: "KAP", CoinGeckoId: "kapital-dao", Decimals: 18, Price: 0.1143},
		{Chain: 36, Addr: "000000000000000000000000b9dfcd4cf589bb8090569cb52fac1b88dbe4981f", Symbol: "BAG", CoinGeckoId: "bag", Decimals: 18, Price: 0.002972},
		{Chain: 36, Addr: "00000000000000000000000068449870eea84453044bd430822827e21fd8f101", Symbol: "ZAI", CoinGeckoId: "zaibot", Decimals: 18, Price: 0.2348},
		{Chain: 36, Addr: "00000000000000000000000047c337bd5b9344a6f3d6f58c474d9d8cd419d8ca", Symbol: "DACKIE", CoinGeckoId: "dackieswap", Decimals: 18, Price: 0.006554},
		{Chain: 36, Addr: "000000000000000000000000d43d8adac6a4c7d9aeece7c3151fca8f23752cf8", Symbol: "ANDY", CoinGeckoId: "andyerc", Decimals: 9, Price: 0.1165},
		{Chain: 36, Addr: "00000000000000000000000087e154e86fb691ab8a27116e93ed8d54e2b8c18c", Symbol: "TES", CoinGeckoId: "titan-trading-token", Decimals: 18, Price: 0.867},
		{Chain: 36, Addr: "000000000000000000000000870a8f46b62b8bdeda4c02530c1750cddf2ed32e", Symbol: "USDC+", CoinGeckoId: "usdc-plus-overnight", Decimals: 18, Price: 1.00},
		{Chain: 36, Addr: "00000000000000000000000042e12d42b3d6c4a74a88a61063856756ea2db357", Symbol: "ORBIT", CoinGeckoId: "orbit-protocol", Decimals: 18, Price: 0.3074},
		// SCROLL (tokens over $50,000 24h volume)
		{Chain: 34, Addr: "0000000000000000000000005300000000000000000000000000000000000004", Symbol: "WETH", CoinGeckoId: "bridged-wrapped-ether-scroll", Decimals: 18, Price: 1905.44},
		{Chain: 34, Addr: "0000000000000000000000000018d96c579121a94307249d47f053e2d687b5e7", Symbol: "MVX", CoinGeckoId: "metavault-trade", Decimals: 18, Price: 2.06},
		{Chain: 34, Addr: "00000000000000000000000047c337bd5b9344a6f3d6f58c474d9d8cd419d8ca", Symbol: "DACKIE", CoinGeckoId: "dackieswap", Decimals: 18, Price: 0.00655},
		{Chain: 34, Addr: "000000000000000000000000f55bec9cafdbe8730f096aa55dad6d22d44099df", Symbol: "USDT", CoinGeckoId: "bridged-tether-scroll", Decimals: 6, Price: 1.00},
		{Chain: 34, Addr: "00000000000000000000000006efdbff2a14a7c8e15944d1f4a48f9f95f663a4", Symbol: "USDC", CoinGeckoId: "bridged-usd-coin-scroll", Decimals: 6, Price: 1.00},
		{Chain: 34, Addr: "000000000000000000000000eb466342c4d449bc9f53a865d5cb90586f405215", Symbol: "AXLUSDC", CoinGeckoId: "bridged-axelar-wrapped-usd-coin-scroll", Decimals: 6, Price: 1.01},
		{Chain: 34, Addr: "0000000000000000000000003c1bca5a656e69edcd0d4e36bebb3fcdaca60cf1", Symbol: "WBTC", CoinGeckoId: "bridged-wrapped-bitcoin-scroll", Decimals: 8, Price: 64415.17},
		{Chain: 34, Addr: "00000000000000000000000060d01ec2d5e98ac51c8b4cf84dfcce98d527c747", Symbol: "IZI", CoinGeckoId: "izumi-finance", Decimals: 18, Price: 0.0142},
		{Chain: 34, Addr: "0000000000000000000000000a3bb08b3a15a19b4de82f8acfc862606fb69a2d", Symbol: "IUSD", CoinGeckoId: "izumi-bond-usd", Decimals: 18, Price: 0.9195},
		{Chain: 34, Addr: "000000000000000000000000f610a9dfb7c89644979b4a0f27063e9e7d7cda32", Symbol: "WSTETH", CoinGeckoId: "bridged-wrapped-lido-staked-ether-scroll", Decimals: 18, Price: 3659.28},
		{Chain: 34, Addr: "000000000000000000000000cA77eB3fEFe3725Dc33bccB54eDEFc3D9f764f97", Symbol: "DAI", CoinGeckoId: "dai", Decimals: 18, Price: 1.00},
		{Chain: 34, Addr: "00000000000000000000000053878B874283351D26d206FA512aEcE1Bef6C0dD", Symbol: "RETH", CoinGeckoId: "rocket-pool-eth", Decimals: 18, Price: 3475.55},
		// X LAYER (tokens over $50,000 24h volume)
		{Chain: 37, Addr: "0000000000000000000000001e4a5963abfd975d8c9021ce480b42188849d41d", Symbol: "USDT", CoinGeckoId: "polygon-hermez-bridged-usdt-x-layer", Decimals: 6, Price: 0.9969},
		{Chain: 37, Addr: "000000000000000000000000e538905cf8410324e03a5a23c1c177a474d59b2b", Symbol: "WOKB", CoinGeckoId: "wrapped-okb", Decimals: 18, Price: 48.76},
		{Chain: 37, Addr: "0000000000000000000000005a77f1443d16ee5761d310e38b62f77f726bc71c", Symbol: "WETH", CoinGeckoId: "weth", Decimals: 18, Price: 2994.60},
		{Chain: 37, Addr: "00000000000000000000000074b7f16337b8972027f6196a17a631ac6de26d22", Symbol: "USDC", CoinGeckoId: "polygon-hermez-bridged-usdc-x-layer", Decimals: 6, Price: 0.9949},
		{Chain: 37, Addr: "000000000000000000000000ea034fb02eb1808c2cc3adbc15f447b93cbe08e1", Symbol: "WBTC", CoinGeckoId: "polygon-hermez-bridged-wbtc-x-layer", Decimals: 8, Price: 57029},
		{Chain: 37, Addr: "000000000000000000000000c5015b9d9161dca7e18e32f6f25c4ad850731fd4", Symbol: "DAI", CoinGeckoId: "polygon-hermez-bridged-dai-x-layer", Decimals: 18, Price: 1.0006},
		// MANTLE (tokens over $50,000 24h volume)
		{Chain: 35, Addr: "000000000000000000000000deaddeaddeaddeaddeaddeaddeaddeaddead0000", Symbol: "MNT", CoinGeckoId: "mantle", Decimals: 18, Price: 1.01},
		{Chain: 35, Addr: "00000000000000000000000078c1b0c915c4faa5fffa6cabf0219da63d7f4cb8", Symbol: "WMNT", CoinGeckoId: "wrapped-mantle", Decimals: 18, Price: 1.01},
		{Chain: 35, Addr: "00000000000000000000000009bc4e0d864854c6afb6eb9a9cdf58ac190d0df9", Symbol: "USDC", CoinGeckoId: "mantle-bridged-usdc-mantle", Decimals: 6, Price: 1},
		{Chain: 35, Addr: "000000000000000000000000201EBa5CC46D216Ce6DC03F6a759e8E766e956aE", Symbol: "USDT", CoinGeckoId: "mantle-bridged-usdt-mantle", Decimals: 6, Price: 0.9973},
		{Chain: 35, Addr: "000000000000000000000000cDA86A272531e8640cD7F1a92c01839911B90bb0", Symbol: "METH", CoinGeckoId: "mantle-staked-ether", Decimals: 18, Price: 3934.06},
		{Chain: 35, Addr: "000000000000000000000000deaddeaddeaddeaddeaddeaddeaddeaddead1111", Symbol: "WETH", CoinGeckoId: "wrapped-ether-mantle-bridge", Decimals: 18, Price: 3825.65},
		{Chain: 35, Addr: "000000000000000000000000371c7ec6d8039ff7933a2aa28eb827ffe1f52f07", Symbol: "JOE", CoinGeckoId: "joe", Decimals: 18, Price: 0.4911},
		// BERACHAIN (non-bridged tokens over $1,000,000 24h volume)
		{Chain: 39, Addr: "0000000000000000000000006969696969696969696969696969696969696969", Symbol: "WBERA", CoinGeckoId: "wrapped-bera", Decimals: 18, Price: 6.62},
		{Chain: 39, Addr: "000000000000000000000000fcbd14dc51f0a4d49d5e53c2e0950e0bc26d0dce", Symbol: "HONEY", CoinGeckoId: "honey-3", Decimals: 18, Price: 0.9985},
		{Chain: 39, Addr: "0000000000000000000000006fc6545d5cde268d5c7f1e476d444f39c995120d", Symbol: "BERAETH", CoinGeckoId: "berachain-staked-eth", Decimals: 18, Price: 2713.26},
		{Chain: 39, Addr: "00000000000000000000000036e9fe653e673fda3857dbe5afbc884af8a316a2", Symbol: "BERAFI", CoinGeckoId: "berafi", Decimals: 18, Price: 0.00117},
		// SEIEVM (tokens over $500,000 24h volume)
		{Chain: 40, Addr: "0000000000000000000000009151434b16b9763660705744891fA906F660EcC5", Symbol: "USDT0", CoinGeckoId: "usdt0", Decimals: 6, Price: 1.00},
		{Chain: 40, Addr: "000000000000000000000000e30fedd158a2e3b13e9badaeabafc5516e95e8c7", Symbol: "WSEI", CoinGeckoId: "wrapped-sei", Decimals: 18, Price: 0.2236},
		{Chain: 40, Addr: "0000000000000000000000003894085ef7ff0f0aedf52e2a2704928d1ec074f1", Symbol: "USDC", CoinGeckoId: "ibc-bridged-usdc", Decimals: 6, Price: 1.00},
		{Chain: 40, Addr: "000000000000000000000000541fd749419ca806a8bc7da8ac23d346f2df8b77", Symbol: "SOLVBTC", CoinGeckoId: "solv-btc", Decimals: 18, Price: 106222},
		{Chain: 40, Addr: "000000000000000000000000cc0966d8418d412c599a6421b760a847eb169a8c", Symbol: "XSOLVBTC", CoinGeckoId: "solv-protocol-solvbtc-bbn", Decimals: 18, Price: 105810},
		// UNICHAIN (tokens over $1,000,000 24h volume)
		{Chain: 44, Addr: "000000000000000000000000078D782b760474a361dDA0AF3839290b0EF57AD6", Symbol: "USDC", CoinGeckoId: "usd-coin", Decimals: 6, Price: 1.00},
		{Chain: 44, Addr: "0000000000000000000000004200000000000000000000000000000000000006", Symbol: "WETH", CoinGeckoId: "unichain-bridged-weth-unichain", Decimals: 18, Price: 2722.24},
		{Chain: 44, Addr: "0000000000000000000000008f187aA05619a017077f5308904739877ce9eA21", Symbol: "UNI", CoinGeckoId: "uniswap", Decimals: 18, Price: 9.43},
		{Chain: 44, Addr: "00000000000000000000000020CAb320A855b39F724131C69424240519573f81", Symbol: "DAI", CoinGeckoId: "dai", Decimals: 18, Price: 1.0},
		// WORLDCHAIN (tokens over $50,000 24h volume)
		{Chain: 45, Addr: "0000000000000000000000002cFc85d8E48F8EAB294be644d9E25C3030863003", Symbol: "WLD", CoinGeckoId: "worldcoin-wld", Decimals: 18, Price: 2.47},
		{Chain: 45, Addr: "00000000000000000000000003C7054BCB39f7b2e5B2c7AcB37583e32D70Cfa3", Symbol: "WBTC", CoinGeckoId: "bridged-wrapped-bitcoin-worldchain", Decimals: 8, Price: 86683.84},
		{Chain: 45, Addr: "0000000000000000000000004200000000000000000000000000000000000006", Symbol: "WETH", CoinGeckoId: "wrapped-eth-world-chain", Decimals: 18, Price: 3311.13},
		{Chain: 45, Addr: "00000000000000000000000079A02482A880bCE3F13e09Da970dC34db4CD24d1", Symbol: "USDC.e", CoinGeckoId: "bridged-usdc-world-chain", Decimals: 6, Price: 1.00},
		// INK (tokens over $500,000 24h volume)
		{Chain: 46, Addr: "0000000000000000000000000200c29006150606b650577bbe7b6248f58470c1", Symbol: "USDT0", CoinGeckoId: "usdt0", Decimals: 6, Price: 1.00},
		{Chain: 46, Addr: "000000000000000000000000f1815bd50389c46847f0bda824ec8da914045d14", Symbol: "USDC.E", CoinGeckoId: "stargate-bridged-usdc-ink", Decimals: 6, Price: 1.00},
		{Chain: 46, Addr: "000000000000000000000000ae4efbc7736f963982aacb17efa37fcbab924cb3", Symbol: "SOLVBTC", CoinGeckoId: "solv-btc", Decimals: 18, Price: 106222},
		{Chain: 46, Addr: "000000000000000000000000c99f5c922dae05b6e2ff83463ce705ef7c91f077", Symbol: "XSOLVBTC", CoinGeckoId: "solv-protocol-solvbtc-bbn", Decimals: 18, Price: 105810},
	}
}
