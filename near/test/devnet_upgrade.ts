const nearAPI = require("near-api-js");
const BN = require("bn.js");
const bs58 = require("bs58");
const fs = require("fs");
const fetch = require("node-fetch");
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
const { createHash } = require('crypto');
import { TestLib } from "./testlib";


import {
  ChainId,
  CHAIN_ID_NEAR,
} from "@certusone/wormhole-sdk/lib/cjs/utils";

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
        userAccount:
          Math.floor(Math.random() * 10000).toString() + "user.test.near",
      };
  }
  return {};
}

function hash(string: any) {
  return createHash('sha256').update(string).digest('hex');
}

async function testDeploy() {
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

  let userKey = nearAPI.utils.KeyPair.fromRandom("ed25519");

  console.log(
    "creating a user account: " +
      config.userAccount +
      " with key " +
      userKey.getPublicKey()
  );

  await masterAccount.createAccount(
    config.userAccount,
    userKey.getPublicKey(),
    new BN(10).pow(new BN(27))
  );

  // A whole new world...
  keyStore = new nearAPI.keyStores.InMemoryKeyStore();
  keyStore.setKey(config.networkId, config.userAccount, userKey);

  near = await nearAPI.connect({
    deps: {
      keyStore,
    },
    networkId: config.networkId,
    nodeUrl: config.nodeUrl,
  });

  const userAccount = new nearAPI.Account(near.connection, config.userAccount);

  const wormholeContract = await fs.readFileSync(
    "../contracts/wormhole/target/wasm32-unknown-unknown/release/near_wormhole.wasm"
  );

  let h = hash(wormholeContract);

  let ts = new TestLib();
  let seq = 1;
  let vaa = ts.genCoreUpdate(ts.singleGuardianPrivKey, 0, 0, seq, CHAIN_ID_NEAR, h);

  let wormholeAccount = new nearAPI.Account(
    near.connection,
    config.wormholeAccount
  );

  console.log("submitting vaa");
  let result = await userAccount.functionCall({
    contractId: config.wormholeAccount,
    methodName: "submit_vaa",
    args: { vaa: vaa },
    attachedDeposit: "12500000000000000000000",
    gas: new BN("150000000000000"),
  });

  console.log("calling upgradeContract");

  result = await userAccount.functionCall({
    contractId: config.wormholeAccount,
    methodName: "update_contract",
    args: wormholeContract,
    attachedDeposit: "5279790000000000000000000",
    gas: 300000000000000,
  });
  console.log("done");

  const tokenContract = await fs.readFileSync(
    "../contracts/token-bridge/target/wasm32-unknown-unknown/release/near_token_bridge.wasm"
  );

  h = hash(tokenContract);
  seq = seq + 1
  vaa = ts.genTokenUpdate(ts.singleGuardianPrivKey, 0, 0, seq, CHAIN_ID_NEAR, h);

  console.log("submitting vaa");
  result = await userAccount.functionCall({
    contractId: config.tokenAccount,
    methodName: "submit_vaa",
    args: { vaa: vaa },
    attachedDeposit: "12500000000000000000000",
    gas: new BN("150000000000000"),
  });

  console.log("submitting vaa again");
  result = await userAccount.functionCall({
    contractId: config.tokenAccount,
    methodName: "submit_vaa",
    args: { vaa: vaa },
    attachedDeposit: "12500000000000000000000",
    gas: new BN("150000000000000"),
  });

  console.log("calling upgradeContract on the token bridge");

  result = await userAccount.functionCall({
    contractId: config.tokenAccount,
    methodName: "update_contract",
    args: tokenContract,
    attachedDeposit: "22797900000000000000000000",
    gas: 300000000000000,
  });

}

testDeploy();
