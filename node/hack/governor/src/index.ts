import {
  tryNativeToHexString,
  ChainId,
} from "@certusone/wormhole-sdk";

const MinNotional = 1000000

const axios = require('axios');
const fs = require("fs");
const execSync = require('child_process').execSync;

/*
  "2Kc38rfQ49DFaKHQaWbijkE7fcymUMLY5guUiUsDmFfn": {
    "Symbol": "KURO",
    "Name": "Kurobi",
    "Address": "2Kc38rfQ49DFaKHQaWbijkE7fcymUMLY5guUiUsDmFfn",
    "CoinGeckoId": "kurobi",
    "Amount": 200,
    "Notional": 1.39,
    "TokenPrice": 0.00694548,
    "TokenDecimals": 6
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

    content += "// This file contains the token config to be used in the mainnet environment.\n"
    content += "//\n"
    content += "// This file was generated: " + (new(Date)).toString() + " using a min notional of " + MinNotional + "\n\n"
    content += "package governor\n\n"
    content += "func tokenList() []tokenConfigEntry {\n"
    content += "\treturn []tokenConfigEntry {\n"

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
                  "\", decimals: " + data.TokenDecimals +
                  ", price: " + data.TokenPrice +
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

    await fs.writeFileSync("../../pkg/governor/mainnet_tokens.go", content, {
      flag: "w+",
    });

    execSync("go fmt ../../pkg/governor/mainnet_tokens.go")
  })
  .catch(error => {
    console.error(error);
  });
