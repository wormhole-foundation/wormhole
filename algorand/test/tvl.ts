const algosdk = require("@certusone/wormhole-sdk/node_modules/algosdk");

import { calcLogicSigAccount } from "@certusone/wormhole-sdk/lib/cjs/algorand";

import {
  coalesceChainName,
  ChainId,
  coalesceChainId,
  uint8ArrayToHex,
  toChainName,
  getOriginalAssetAlgorand,
  CONTRACTS
} from "@certusone/wormhole-sdk";

export async function getNativeAlgoAddress(
  algoClient: any,
  token_bridge: any,
  assetId: any
) {
  const { doesExist, lsa } = await calcLogicSigAccount(
    algoClient,
    BigInt(token_bridge),
    BigInt(assetId),
    Buffer.from("native", "binary").toString("hex")
  );
  return lsa.address();
}

async function firstTransaction() {
  let algodToken;
  let algodServer;
  let algodPort;
  let server;
  let port;
  let token;
  let appid;

  const mainnet = true;

  if (mainnet) {
    appid = 842126029;
    algodToken = "";
    algodServer = "https://mainnet-api.algonode.cloud";
    algodPort = 443;
    server = "https://mainnet-idx.algonode.cloud";
    port = 443;
    token = "";
  } else {
    appid = CONTRACTS["DEVNET"].algorand.token_bridge;
    algodToken =
      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
    algodServer = "http://localhost";
    algodPort = 4001;
    server = "http://localhost";
    port = 8980;
    token = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
  }

  let algodClient = new algosdk.Algodv2(algodToken, algodServer, algodPort);
  let indexerClient = new algosdk.Indexer(token, server, port);
  let addr = algosdk.getApplicationAddress(appid); // mainnet token bridge account

  console.log(await algodClient.status().do());

  //Check your balance
  let accountInfo = await algodClient.accountInformation(addr).do();
  console.log(accountInfo);

  let ret = await indexerClient
    .searchAccounts()
    .authAddr(addr)
    .applicationID(appid)
    .do();

  let wormholeAssets: any = [];
  let nativeAssets: any = [];

  while (true) {
    ret["accounts"].forEach((x: any) => {
      let amt = x["amount"];
      if (x["assets"] != undefined) {
        x["assets"].forEach((a: any) => {
          if (x["created-assets"] != undefined) {
            wormholeAssets.push(a);
          } else {
            nativeAssets.push(a);
          }
        });
      }
    });
    if (ret["next-token"] == undefined) {
      break;
    }
    ret = await indexerClient
      .searchAccounts()
      .authAddr(addr)
      .applicationID(appid)
      .nextToken(ret["next-token"])
      .do();
  }

  let nativeAlgoAddr = await getNativeAlgoAddress(algodClient, appid, 0);
  let algoInfo = await algodClient.accountInformation(nativeAlgoAddr).do();
  console.log("ALGO locked: " + (algoInfo["amount"] - 1002001));

  console.log("wormhole assets (bridged in)");
  for (let i = 0; i < wormholeAssets.length; i++) {
    let orig = await getOriginalAssetAlgorand(
      algodClient,
      BigInt(appid),
      wormholeAssets[i]["asset-id"]
    );
    let v = [
      coalesceChainName(orig["chainId"]),
      uint8ArrayToHex(orig["assetAddress"]),
      wormholeAssets[i],
      await algodClient.getAssetByID(wormholeAssets[i]["asset-id"]).do(),
    ];
    console.log(v);
  }

  console.log("native assets");
  for (let i = 0; i < nativeAssets.length; i++) {
    console.log(nativeAssets[i]);
    console.log(
      await algodClient.getAssetByID(nativeAssets[i]["asset-id"]).do()
    );

    let addr = await getNativeAlgoAddress(
      algodClient,
      appid,
      nativeAssets[i]["asset-id"]
    );
    algoInfo = await algodClient.accountInformation(addr).do();
    console.log(algoInfo);
  }
}

firstTransaction();
