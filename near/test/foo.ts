const nearAPI = require("near-api-js");
const BN = require("bn.js");
const fs = require("fs");
const fetch = require("node-fetch");
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";

import {
  CONTRACTS,
  attestNearFromNear,
  attestTokenFromNear,
  attestFromAlgorand,
  createWrappedOnAlgorand,
  tryNativeToUint8Array,
  createWrappedOnNear,
  getEmitterAddressAlgorand,
  getForeignAssetAlgorand,
  getForeignAssetNear,
  getIsTransferCompletedNear,
  getIsWrappedAssetNear,
  getOriginalAssetNear,
  getSignedVAAWithRetry,
  redeemOnAlgorand,
  redeemOnNear,
  transferFromAlgorand,
  transferNearFromNear,
  transferTokenFromNear,
} from "@certusone/wormhole-sdk";

const wh = require("@certusone/wormhole-sdk");

import {
  CHAIN_ID_ALGORAND,
  CHAIN_ID_NEAR,
  ChainId,
  ChainName,
  textToHexString,
  textToUint8Array,
} from "@certusone/wormhole-sdk/lib/cjs/utils";

import {
  _parseVAAAlgorand,
} from "@certusone/wormhole-sdk/lib/cjs/algorand";


function getConfig(env: any) {
  switch (env) {
    case "sandbox":
    case "local":
      return {
        networkId: "sandbox",
        nodeUrl: "http://localhost:3030",
        masterAccount: "test.near",
        wormholeAccount: "wormhole.test.near",
        tokenAccount: "token.test.near",
        nftAccount: "nft.test.near",
        testAccount: "test.test.near",
      };
    case "testnet":
      return {
        networkId: "testnet",
        nodeUrl: "https://rpc.testnet.near.org",
        masterAccount: "wormhole.testnet",
        wormholeAccount: "wormhole.wormhole.testnet",
        tokenAccount: "token.wormhole.testnet",
        nftAccount: "nft.wormhole.testnet",
        testAccount: "test.wormhole.testnet",
      };
  }
  return {};
}

async function initNear() {
  let e = "testnet";

  let config = getConfig(e);

  let masterKey = nearAPI.utils.KeyPair.fromString(
    "ed25519:5dJ7Nsq4DQBdiGvZLPyjRVmhtRaScahsREpEPtaAyE9Z3CgyZFsaBwpybCRBMugiwhbFCUkqHk7PJ3BVcgZZ9Lgk"
  );
  let masterPubKey = masterKey.getPublicKey();

  let keyStore = new nearAPI.keyStores.InMemoryKeyStore();
  keyStore.setKey(config.networkId, config.masterAccount, masterKey);

  let near = await nearAPI.connect({
    deps: {
      keyStore,
    },
    networkId: config.networkId,
    nodeUrl: config.nodeUrl,
  });
  let masterAccount = new nearAPI.Account(
    near.connection,
    config.masterAccount
  );

  console.log(
    "Finish init NEAR masterAccount: " +
      JSON.stringify(await masterAccount.getAccountBalance())
  );

  console.log("setting key for new wormhole contract");
  keyStore.setKey(config.networkId, config.wormholeAccount, masterKey);
  keyStore.setKey(config.networkId, config.tokenAccount, masterKey);
  keyStore.setKey(config.networkId, config.nftAccount, masterKey);

  let tokenAccount = new nearAPI.Account(near.connection, config.tokenAccount);

  let token_bridge = "token.wormhole.testnet";
  let core_bridge = "wormhole.wormhole.testnet";

  console.log(
    await tokenAccount.viewFunction(config.tokenAccount, "emitter", {})
  );

  console.log("attesting Near itself");
  let s = await attestNearFromNear(masterAccount, core_bridge, token_bridge);

  const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
    ["https://wormhole-v2-testnet-api.certus.one"],
    CHAIN_ID_NEAR,
    s[1],
    s[0].toString(),
    {
      transport: NodeHttpTransport(),
    }
  );

  console.log("vaa: " + Buffer.from(signedVAA).toString("hex"));

  s = await transferNearFromNear(
    masterAccount,
    core_bridge,
    token_bridge,
    BigInt(1000000000000000000000000),
    tryNativeToUint8Array("0x3bC7f2e458aC4E55F941C458cfD8c6851a591B4F", "ethereum"),
    2,
    BigInt(0)
  );

  const { vaaBytes: signedTrans } = await getSignedVAAWithRetry(
    ["https://wormhole-v2-testnet-api.certus.one"],
    CHAIN_ID_NEAR,
    s[1],
    s[0].toString(),
    {
      transport: NodeHttpTransport(),
    }
  );

  console.log("vaa: " + Buffer.from(signedTrans).toString("hex"));

    let p = _parseVAAAlgorand(signedTrans);
    console.log(p);
}

initNear();
