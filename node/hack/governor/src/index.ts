import {
  tryNativeToHexString,
  ChainId,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  hexToUint8Array,
  uint8ArrayToHex,
  parseTransferPayload,
  getEmitterAddressEth,
  getEmitterAddressSolana,
  getEmitterAddressTerra,
} from "@certusone/wormhole-sdk";

const MinNotional = 1000000

const axios = require('axios');
const fs = require("fs");

/*
  '2Kc38rfQ49DFaKHQaWbijkE7fcymUMLY5guUiUsDmFfn': {
    Symbol: 'KURO',
    Name: 'Kurobi',
    Address: '2Kc38rfQ49DFaKHQaWbijkE7fcymUMLY5guUiUsDmFfn',
    CoinGeckoId: 'kurobi',
    Amount: 200,
    Notional: 1.52,
    TokenPrice: 0.00757962
  },
*/

axios
  .get('https://europe-west3-wormhole-315720.cloudfunctions.net/mainnet-notionaltvl')
  .then(async res => {
    if (res.status != 200) {
        console.error("failed to read symbols, statusCode: %o", res.status)
        process.exit
    }

    var content = ""

    content += "// This file was generated: " + (new(Date)).toString() + " using a min notional of " + MinNotional + "\n"
    content += "package governor\n\n"
    content += "func tokenList() []tokenConfigEntry {\n"
    content += "\treturn [] tokenConfigEntry {\n"

    for (let chain in res.data.AllTime) {
        for (let addr in res.data.AllTime[chain]) {
            if (addr !== "*") {
                let data = res.data.AllTime[chain][addr]
                let notional = parseInt(data.Notional)
                if (notional > MinNotional) {
                  if (data.Address == "ust") {
                    continue
                  }
                  let chainId = parseInt(chain) as ChainId
                  const wormholeAddr = tryNativeToHexString(
                    data.Address,
                    chainId
                  );

                  content += "\t\ttokenConfigEntry { chain: " + chain +
                  ", addr: \"" + wormholeAddr +
                  "\", symbol: \"" + data.Symbol +
                  "\", coinGeckoId: \"" + data.CoinGeckoId +
                  "\", decimals: 18, price: " +
                  data.TokenPrice +
                  " }, // Addr: " +
                  data.Address + ", Notional: " + notional +
                  "\n"

                    //console.log("chain: " + chain + ", addr: " + data.Address + ", symbol: " + data.Symbol + ", notional: " + notional + ", price: " + data.TokenPrice + ", amount: " + data.Amount)
                }
            }
        }
    }

    content += "\t}\n"
    content += "}\n"

    await fs.writeFileSync("../../pkg/governor/tokens.go", content, {
      flag: "w+",
    });
  })
  .catch(error => {
    console.error(error);
  });
