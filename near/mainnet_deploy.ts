// npx pretty-quick

const nearAPI = require("near-api-js");
const BN = require("bn.js");
const fs = require("fs");
const fetch = require("node-fetch");
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";

const { parseSeedPhrase, generateSeedPhrase } = require("near-seed-phrase");

function getConfig(env: any) {
  switch (env) {
    case "mainnet":
      return {
        networkId: "mainnet",
        nodeUrl: "https://rpc.mainnet.near.org",
        wormholeMasterAccount: "wormhole_crypto.near",
        wormholeAccount: "contract.wormhole_crypto.near",
        portalMasterAccount: "portalbridge.near",
        portalAccount: "contract.portalbridge.near",
      };
  }
  return {};
}

async function initNear() {
  let config = getConfig("mainnet");

  let wormholeKeys = parseSeedPhrase(process.env.WORMHOLE_KEYS);
  let portalKeys = parseSeedPhrase(process.env.PORTAL_KEYS);

  let wormholeMasterKey = nearAPI.utils.KeyPair.fromString(
    wormholeKeys["secretKey"]
  );
  let portalMasterKey = nearAPI.utils.KeyPair.fromString(
    portalKeys["secretKey"]
  );

  let keyStore = new nearAPI.keyStores.InMemoryKeyStore();
  keyStore.setKey(
    config.networkId,
    config.wormholeMasterAccount,
    wormholeMasterKey
  );
  keyStore.setKey(config.networkId, config.wormholeAccount, wormholeMasterKey);
  keyStore.setKey(
    config.networkId,
    config.portalMasterAccount,
    portalMasterKey
  );
  keyStore.setKey(config.networkId, config.portalAccount, portalMasterKey);

  let near = await nearAPI.connect({
    keyStore,
    networkId: config.networkId,
    nodeUrl: config.nodeUrl,
  });

  let wormholeMasterAccount = new nearAPI.Account(
    near.connection,
    config.wormholeMasterAccount
  );

  let portalMasterAccount = new nearAPI.Account(
    near.connection,
    config.portalMasterAccount
  );

  console.log(
    "wormhole account: " +
      JSON.stringify(await wormholeMasterAccount.getAccountBalance())
  );
  console.log(
    "portal account: " +
      JSON.stringify(await portalMasterAccount.getAccountBalance())
  );

  const wormholeContract = await fs.readFileSync(
    "contracts/wormhole/target/wasm32-unknown-unknown/release/near_wormhole.wasm"
  );
  const portalContract = await fs.readFileSync(
    "contracts/token-bridge/target/wasm32-unknown-unknown/release/near_token_bridge.wasm"
  );

  console.log("setting key for new wormhole contract");

  keyStore.setKey(config.networkId, config.wormholeAccount, wormholeMasterKey);
  keyStore.setKey(config.networkId, config.portalAccount, portalMasterKey);

  console.log("Deploying core/wormhole contract: " + config.wormholeAccount);

  let wormholeAccount = await wormholeMasterAccount.createAndDeployContract(
    config.wormholeAccount,
    wormholeMasterKey.getPublicKey(),
    wormholeContract,
    new BN("5000000000000000000000000")
  );

  console.log("Deploying core/portal contract: " + config.portalAccount);

  let portalAccount = await portalMasterAccount.createAndDeployContract(
    config.portalAccount,
    portalMasterKey.getPublicKey(),
    portalContract,
    new BN("12000000000000000000000000")
  );

  let result = await wormholeMasterAccount.functionCall({
    contractId: config.wormholeAccount,
    methodName: "boot_wormhole",
    args: {
      gset: 0,
      addresses: ["58CC3AE5C097b213cE3c81979e1B9f9570746AA5"],
    },
    gas: 100000000000000,
  });

  result = await portalMasterAccount.functionCall({
    contractId: config.portalAccount,
    methodName: "boot_portal",
    args: {
      core: config.wormholeAccount,
    },
    gas: 100000000000000,
  });

  await wormholeMasterAccount.functionCall({
    contractId: config.wormholeAccount,
    methodName: "register_emitter",
    args: { emitter: config.portalAccount },
    attachedDeposit: new BN("30000000000000000000000"),
    gas: new BN("100000000000000"),
  });

  console.log("deleting the master key from the token contract");
  await portalAccount.deleteKey(portalMasterKey.getPublicKey());

  console.log("deleting the master key from the wormhole contract");
  await wormholeAccount.deleteKey(wormholeMasterKey.getPublicKey());
}

initNear();
