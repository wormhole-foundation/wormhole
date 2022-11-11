import {
  tryNativeToHexString,
  ChainId,
  CHAIN_ID_ALGORAND,
  CHAIN_ID_APTOS,
} from "@certusone/wormhole-sdk";

const MinNotional = 0

const axios = require('axios');
const fs = require("fs");
const execSync = require('child_process').execSync;

const IncludeFileName = "./include_list.csv"
let includedTokens = new Map();
if (fs.existsSync(IncludeFileName)) {
  console.log("loading included symbols from file " + IncludeFileName)
  const data = fs.readFileSync(IncludeFileName, 'utf-8');
  const lines = data.toString().replace(/\r\n/g,'\n').split('\n');
  for(let line of lines) {
    if (line !== "" && line[0] !== '#') {
      let fields = line.split(",", 10)
      if (fields.length < 2) {
        throw Error("line in include list does not contain enough fields")
      }

      includedTokens.set(fields[0] + ":" + fields[1].toLowerCase(), true)
    }
  }
}

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
    content += "func generatedMainnetTokenList() []tokenConfigEntry {\n"
    content += "\treturn []tokenConfigEntry {\n"

    for (let chain in res.data.AllTime) {
        for (let addr in res.data.AllTime[chain]) {
            if (addr !== "*") {
                let data = res.data.AllTime[chain][addr]
                let notional = parseInt(data.Notional)
                let key = chain + ":" + data.Address.toLowerCase()
                let includeIt = true;
                if (notional > MinNotional) {
                  includeIt = true
                } else {
                  if (includedTokens.has(key)) {
                    includeIt = true
                  }
                }
                if (includeIt) {
                  includedTokens.delete(key)
                  let chainId = parseInt(chain) as ChainId
                  let wormholeAddr: string
                  try {
                    wormholeAddr = tryNativeToHexString(
                      data.Address,
                      chainId
                    );
                  } catch (e) {
                    if (chainId != CHAIN_ID_APTOS) {
                     if (chainId == CHAIN_ID_ALGORAND) {
                        wormholeAddr = ""
                        if ((data.Address === "algo") || (data.Address === "0")) {
                          wormholeAddr = "0000000000000000000000000000000000000000000000000000000000000000"
                        } else if (data.Address === "31566704") {
                          wormholeAddr = "0000000000000000000000000000000000000000000000000000000001e1ab70"
                        } else if (data.Address === "312769") {
                          wormholeAddr = "000000000000000000000000000000000000000000000000000000000004c5c1"
                        }
                      }
                    }
                    if (wormholeAddr === "") {
                      console.log(`Ignoring symbol '${data.Symbol}' because the address '${data.Address}' is invalid`)
                      continue
                    }
                  }

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

    await fs.writeFileSync("../../pkg/governor/generated_mainnet_tokens.go", content, {
      flag: "w+",
    });

    execSync("go fmt ../../pkg/governor/generated_mainnet_tokens.go")

    if (includedTokens.size != 0) {
      for (let [key, value] of includedTokens) {
        console.error(`Did not find included token '${key}' in query result!`)
      }
    }

    console.log("\nPlease do \"go run check_query.go\" to verify the Coin Gecko query still works before doing a commit.")
  })
  .catch(error => {
    console.error(error);
  });
