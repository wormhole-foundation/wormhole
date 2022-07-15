const nearAPI = require("near-api-js");
const BN = require("bn.js");
const bs58 = require("bs58");
const fs = require("fs");
const fetch = require("node-fetch");
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";

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
  keyStore.setKey(config.networkId, config.userAccount, userKey);

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
  const userAccount = new nearAPI.Account(near.connection, config.userAccount);

  const wormholeContract = await fs.readFileSync(
    "../contracts/wormhole/target/wasm32-unknown-unknown/release/near_wormhole.wasm"
  );

  //console.log("sending money to cover the cost of deploying this contract.. so that we fail for the right reasons");
  //await userAccount.sendMoney(config.wormholeAccount, new BN("10000000000000000000000000"));

  keyStore.setKey(config.networkId, config.wormholeAccount, masterKey);

  let wormholeAccount = new nearAPI.Account(
    near.connection,
    config.wormholeAccount
  );

  try {
    console.log("redeploying wormhole contract using standard deployment API");
    let resp = await wormholeAccount.deployContract(wormholeContract);
    console.log(resp);
    console.log("This should have thrown a exception..");
    process.exit(1);
  } catch {
    console.log("Exception thrown.. nice.. we dont suck");
  }

  console.log("calling upgradeContract");
  let result = await userAccount.functionCall({
    contractId: config.wormholeAccount,
    methodName: "update_contract",
    args: wormholeContract,
    attachedDeposit: "12500000000000000000000",
    gas: 300000000000000,
  });
  console.log("done");

  console.log(result);
}

testDeploy();
