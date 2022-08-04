// npx pretty-quick

import Web3 from "web3";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";

import { TestLib } from "./testlib";

import { Account as nearAccount } from "@certusone/wormhole-sdk/node_modules/near-api-js";
const BN = require("bn.js");
const fetch = require("node-fetch");

const nearAPI = require("near-api-js");

export function textToHexString(name: string): string {
  return Buffer.from(name, "binary").toString("hex");
}

export const hexToUint8Array = (h: string): Uint8Array =>
  new Uint8Array(Buffer.from(h, "hex"));

export const uint8ArrayToHex = (a: Uint8Array): string =>
  Buffer.from(a).toString("hex");

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
        userAccount:
          Math.floor(Math.random() * 10000).toString() + "user.test.near",
        user2Account:
          Math.floor(Math.random() * 10000).toString() + "user.test.near",
      };
  }
  return {};
}

async function testNearMsg() {
  let config = getConfig(process.env.NEAR_ENV || "sandbox");

  // Retrieve the validator key directly in the Tilt environment
  const response = await fetch("http://localhost:3031/validator_key.json");

  const keyFile = await response.json();

  let masterKey = nearAPI.utils.KeyPair.fromString(
    keyFile.secret_key || keyFile.private_key
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

  while (true) {
    let myAddress = nearAPI.providers.getTransactionLastResult(
      await masterAccount.functionCall({
        contractId: "test.test.near",
        methodName: "publish_message",
        args: { core: "wormhole.test.near", p: "00" },
        gas: new BN("100000000000000"),
        attachedDeposit: new BN("2000000000000000000000"), // 0.002 NEAR
      })
    );
    await new Promise((f) => setTimeout(f, 1000));
  }
}

testNearMsg();
