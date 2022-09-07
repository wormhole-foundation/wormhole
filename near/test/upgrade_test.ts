// npx pretty-quick

const nearAPI = require("near-api-js");
const BN = require("bn.js");
const fs = require("fs");
const fetch = require("node-fetch");
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { TestLib } from "./testlib";
const { createHash } = require("crypto");

import { ChainId, CHAIN_ID_NEAR } from "@certusone/wormhole-sdk/lib/cjs/utils";

function hash(string: any) {
  return createHash("sha256").update(string).digest("hex");
}

function getConfig(env: any) {
  switch (env) {
    case "sandbox":
    case "local":
      return {
        networkId: "sandbox",
        nodeUrl: "http://localhost:3030",
        masterAccount: "test.near",
        wormholeAccount:
          Math.floor(Math.random() * 10000).toString() + "wormhole.test.near",
        tokenAccount:
          Math.floor(Math.random() * 10000).toString() + "token.test.near",
      };
    case "testnet":
      return {
        networkId: "testnet",
        nodeUrl: "https://rpc.testnet.near.org",
        masterAccount: "wormhole.testnet",
        wormholeAccount: "wormhole.wormhole.testnet",
        tokenAccount: "token.wormhole.testnet",
      };
    case "mainnet":
      return {
        networkId: "mainnet",
        nodeUrl: "https://rpc.mainnet.near.org",
        wormholeMasterAccount: "wormhole_crypto.near",
        wormholeAccount: "contract.wormhole_crypto.near",
        portalMasterAccount: "portalbridge.near",
        tokenAccount: "contract.portalbridge.near",
      };
  }
  return {};
}

async function initNear() {
  let e = process.env.NEAR_ENV || "sandbox";

  let config = getConfig(e);
  let p = process.env.PROD_ENV || "mainnet";
  let prod_config = getConfig(p);

  let masterKey: any;

  if (e === "sandbox") {
    // Retrieve the validator key directly in the Tilt environment
    const response = await fetch("http://localhost:3031/validator_key.json");

    const keyFile = await response.json();

    masterKey = nearAPI.utils.KeyPair.fromString(
      keyFile.secret_key || keyFile.private_key
    );
  } else {
    masterKey = nearAPI.utils.KeyPair.fromString(process.env.NEAR_PK);
  }
  let masterPubKey = masterKey.getPublicKey();

  let keyStore = new nearAPI.keyStores.InMemoryKeyStore();
  keyStore.setKey(config.networkId, config.masterAccount, masterKey);
  keyStore.setKey(config.networkId, config.wormholeAccount, masterKey);
  keyStore.setKey(config.networkId, config.tokenAccount, masterKey);

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

  let prod_near = await nearAPI.connect({
    deps: {
      keyStore,
    },
    networkId: prod_config.networkId,
    nodeUrl: prod_config.nodeUrl,
  });

  console.log(
    "Finish init NEAR masterAccount: " +
      JSON.stringify(await masterAccount.getAccountBalance())
  );

  console.log("reading local wasms we want to upgrade to...");

  const wormholeContract = await fs.readFileSync(
    "artifacts/near_wormhole.wasm"
  );
  const tokenContract = await fs.readFileSync(
    "artifacts/near_token_bridge.wasm"
  );

  console.log("reading current production code ("+p+") we want to upgrade from..");

  let wormhole_prod_code;
  if (process.env.WORMHOLE_CONTRACT) {
      console.log("Reading " + process.env.WORMHOLE_CONTRACT);
    wormhole_prod_code = await fs.readFileSync(process.env.WORMHOLE_CONTRACT);
  } else {
    wormhole_prod_code = Buffer.from(
      (
        await prod_near.connection.provider.query({
          request_type: "view_code",
          finality: "final",
          account_id: prod_config.wormholeAccount,
        })
      ).code_base64,
      "base64"
    );
  }

  let token_prod_code;

  if (process.env.TOKEN_CONTRACT) {
      console.log("Reading " + process.env.TOKEN_CONTRACT);
    token_prod_code = await fs.readFileSync(process.env.TOKEN_CONTRACT);
  } else {
    token_prod_code = Buffer.from(
      (
        await prod_near.connection.provider.query({
          request_type: "view_code",
          finality: "final",
          account_id: prod_config.tokenAccount,
        })
      ).code_base64,
      "base64"
    );
  }

  console.log(
    "Deploying production ("+p+") wormhole contract to " + config.wormholeAccount
  );
  let wormholeAccount = await masterAccount.createAndDeployContract(
    config.wormholeAccount,
    masterKey.getPublicKey(),
    wormhole_prod_code,
    new BN("20000000000000000000000000")
  );

  await wormholeAccount.functionCall({
    contractId: config.wormholeAccount,
    methodName: "register_emitter",
    args: { emitter: config.tokenAccount },
    attachedDeposit: new BN("30000000000000000000000"),
    gas: new BN("100000000000000"),
  });

  let tokenAccount: any;

  console.log("Deploying production ("+p+") token contract to " + config.tokenAccount);
  tokenAccount = await masterAccount.createAndDeployContract(
    config.tokenAccount,
    masterKey.getPublicKey(),
    token_prod_code,
    new BN("20000000000000000000000000")
  );

  let signers: any[] = [];

  let vaasToken: any[] = [];
  let vaasNFT: any[] = [];

  let lines = fs.readFileSync(".env", "utf-8").split("\n");

  lines.forEach((line: any) => {
    let f = line.split("=");
    if (f[0] === "INIT_SIGNERS") {
      signers = eval(f[1]);
    }
    if (f[0].startsWith("REGISTER_") && f[0].endsWith("TOKEN_BRIDGE_VAA")) {
      vaasToken.push(f[1]);
    } else if (f[0].endsWith("TOKEN_BRIDGE_VAA_REGISTER")) {
      vaasToken.push(f[1]);
    }

    if (f[0].startsWith("REGISTER_") && f[0].endsWith("NFT_BRIDGE_VAA")) {
      vaasNFT.push(f[1]);
    } else if (
      f[0].endsWith("NFT_BRIDGE_VAA") ||
      f[0].endsWith("NFT_BRIDGE_VAA_REGISTER")
    ) {
      vaasNFT.push(f[1]);
    }
  });

  if (e === "sandbox") {
    let result = await masterAccount.functionCall({
      contractId: config.wormholeAccount,
      methodName: "boot_wormhole",
      args: {
        gset: 0,
        addresses: signers,
      },
      gas: 100000000000000,
    });

    console.log("Booting up the token bridge");

    result = await masterAccount.functionCall({
      contractId: config.tokenAccount,
      methodName: "boot_portal",
      args: {
        core: config.wormholeAccount,
      },
      gas: 100000000000000,
    });
  }

  console.log("token bridge booted.. now the fun begins");

  let ts = new TestLib();
  let seq = 1;
  let h = hash(wormholeContract);
  console.log(h);
  let vaa = ts.genCoreUpdate(
    ts.singleGuardianPrivKey,
    0,
    0,
    seq,
    CHAIN_ID_NEAR,
    h
  );

  console.log("submitting vaa to wormhole contract for upgrade");

  let result;

  result = await masterAccount.functionCall({
    contractId: config.wormholeAccount,
    methodName: "submit_vaa",
    args: { vaa: vaa },
    attachedDeposit: "12500000000000000000000",
    gas: new BN("150000000000000"),
  });

  console.log("calling upgradeContract");

    // result = await wormholeAccount.deployContract(wormholeContract);

  result = await masterAccount.functionCall({
    contractId: config.wormholeAccount,
    methodName: "update_contract",
    args: wormholeContract,
    attachedDeposit: "5279790000000000000000000",
    gas: 300000000000000,
  });

  console.log("---");
  console.log("lets do a health check on the wormhole contract");

  console.log(
    "message_fee returned: " +
      (await masterAccount.viewFunction(
        config.wormholeAccount,
        "message_fee",
        {}
      ))
  );

  h = hash(tokenContract);
  seq = seq + 1;
  vaa = ts.genTokenUpdate(
    ts.singleGuardianPrivKey,
    0,
    0,
    seq,
    CHAIN_ID_NEAR,
    h
  );

  console.log("submitting vaa");
  result = await masterAccount.functionCall({
    contractId: config.tokenAccount,
    methodName: "submit_vaa",
    args: { vaa: vaa },
    attachedDeposit: "12500000000000000000000",
    gas: new BN("150000000000000"),
  });

  console.log("submitting vaa again");
  result = await masterAccount.functionCall({
    contractId: config.tokenAccount,
    methodName: "submit_vaa",
    args: { vaa: vaa },
    attachedDeposit: "12500000000000000000000",
    gas: new BN("150000000000000"),
  });

  console.log("calling upgradeContract on the token bridge");

    // result = await tokenAccount.deployContract(tokenContract);

  result = await masterAccount.functionCall({
    contractId: config.tokenAccount,
    methodName: "update_contract",
    args: tokenContract,
    attachedDeposit: "22797900000000000000000000",
    gas: 300000000000000,
  });

  console.log("---");
  console.log("lets do a health check on the token bridge contract");

  console.log(
    "emitter returned: " +
      (await masterAccount.viewFunction(config.tokenAccount, "emitter", {}))
  );
}

initNear();
