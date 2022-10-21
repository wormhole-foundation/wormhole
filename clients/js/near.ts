import { impossible, Payload } from "./vaa";
import { NETWORKS } from "./networks";
import { CONTRACTS } from "@certusone/wormhole-sdk/lib/cjs/utils/consts";
const { parseSeedPhrase, generateSeedPhrase } = require("near-seed-phrase");
const fs = require("fs");

const BN = require("bn.js");
const nearAPI = require("near-api-js");

function default_near_args(argv) {
  let network = argv["n"].toUpperCase();
  let contracts = CONTRACTS[network]["near"];
  let n = NETWORKS[network]["near"];

  if (!("rpc" in argv)) {
    argv["rpc"] = n["rpc"];
  }

  if (!("target" in argv) && "module" in argv) {
    if (argv["module"] == "Core") {
      argv["target"] = contracts["core"];
      console.log("Setting target to core");
    }
    if (argv["module"] == "TokenBridge") {
      argv["target"] = contracts["token_bridge"];
      console.log("Setting target to token_bridge");
    }
  }

  if (!("key" in argv)) {
    if (n["key"]) {
      argv["key"] = n["key"];
    }
  }

  if (!("key" in argv)) {
    if ("mnemonic" in argv) {
      let k = parseSeedPhrase(argv["mnemonic"]);
      argv["key"] = k["secretKey"];
    }
  }
}

export async function deploy_near(argv) {
  default_near_args(argv);

  let masterKey = nearAPI.utils.KeyPair.fromString(argv["key"]);
  let keyStore = new nearAPI.keyStores.InMemoryKeyStore();
  keyStore.setKey(argv["networkId"], argv["account"], masterKey);
  keyStore.setKey(argv["networkId"], argv["target"], masterKey);

  let near = await nearAPI.connect({
    deps: {
      keyStore,
    },
    networkId: argv["networkId"],
    nodeUrl: argv["rpc"],
  });

  let masterAccount = new nearAPI.Account(near.connection, argv["account"]);
  let targetAccount = new nearAPI.Account(near.connection, argv["target"]);

  console.log(argv);

  if ("attach" in argv) {
    console.log(
      "Sending money: " +
        argv["target"] +
        " from " +
        argv["account"] +
        " being sent " +
        argv["attach"]
    );
    console.log(await masterAccount.sendMoney(argv["target"], argv["attach"]));
  }

  console.log("deploying contract");
  console.log(
    await targetAccount.deployContract(await fs.readFileSync(argv["file"]))
  );
}

export async function upgrade_near(argv) {
  default_near_args(argv);

  let masterKey = nearAPI.utils.KeyPair.fromString(argv["key"]);
  let keyStore = new nearAPI.keyStores.InMemoryKeyStore();
  keyStore.setKey(argv["networkId"], argv["account"], masterKey);

  let near = await nearAPI.connect({
    deps: {
      keyStore,
    },
    networkId: argv["networkId"],
    nodeUrl: argv["rpc"],
  });

  let masterAccount = new nearAPI.Account(near.connection, argv["account"]);

  let result = await masterAccount.functionCall({
    contractId: argv["target"],
    methodName: "update_contract",
    args: await fs.readFileSync(argv["file"]),
    attachedDeposit: "22797900000000000000000000",
    gas: 300000000000000,
  });
  console.log(result);
}

export async function execute_near(
  payload: Payload,
  vaa: string,
  network: "MAINNET" | "TESTNET" | "DEVNET"
) {
  let n = NETWORKS[network]["near"];
  let contracts = CONTRACTS[network]["near"];

  let target_contract = "";
  let numSubmits = 1;

  switch (payload.module) {
    case "Core":
      if (contracts.core === undefined) {
        throw new Error("Core bridge not supported yet for near");
      }
      target_contract = contracts.core;
      switch (payload.type) {
        case "GuardianSetUpgrade":
          console.log("Submitting new guardian set");
          break;
        case "ContractUpgrade":
          console.log("Upgrading core contract");
          break;
        default:
          impossible(payload);
      }
      break;
    case "NFTBridge":
      if (contracts.nft_bridge === undefined) {
        throw new Error("NFT bridge not supported yet for near");
      }
      numSubmits = 2;
      target_contract = contracts.nft_bridge;
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          break;
        case "RegisterChain":
          console.log("Registering chain");
          break;
        case "Transfer":
          console.log("Completing transfer");
          break;
        default:
          impossible(payload);
      }
      break;
    case "TokenBridge":
      if (contracts.token_bridge === undefined) {
        throw new Error("Token bridge not supported yet for near");
      }
      numSubmits = 2;
      target_contract = contracts.token_bridge;
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          break;
        case "RegisterChain":
          console.log("Registering chain");
          break;
        case "Transfer":
          console.log("Completing transfer");
          break;
        case "AttestMeta":
          console.log("Creating wrapped token");
          break;
        case "TransferWithPayload":
          throw Error("Can't complete payload 3 transfer from CLI");
        default:
          impossible(payload);
          break;
      }
      break;
    default:
      impossible(payload);
  }

  let key = nearAPI.utils.KeyPair.fromString(n.key);

  let keyStore = new nearAPI.keyStores.InMemoryKeyStore();
  keyStore.setKey(n.networkId, n.deployerAccount, key);

  let near = await nearAPI.connect({
    keyStore,
    networkId: n.networkId,
    nodeUrl: n.rpc,
  });

  let nearAccount = new nearAPI.Account(near.connection, n.deployerAccount);

  console.log("submitting vaa the first time");
  let result1 = await nearAccount.functionCall({
    contractId: target_contract,
    methodName: "submit_vaa",
    args: {
      vaa: vaa,
    },
    attachedDeposit: new BN("100000000000000000000000"),
    gas: new BN("300000000000000"),
  });

  if (numSubmits <= 1) {
    console.log("Hash: " + result1.transaction.hash);
    return;
  }

  // You have to feed a vaa twice into the contract (two submits),
  // The first time, it checks if it has been seen at all.
  // The second time, it executes.
  console.log("submitting vaa the second time");
  let result2 = await nearAccount.functionCall({
    contractId: target_contract,
    methodName: "submit_vaa",
    args: {
      vaa: vaa,
    },
    attachedDeposit: new BN("100000000000000000000000"),
    gas: new BN("300000000000000"),
  });

  let txHash = result1.transaction.hash + ":" + result2.transaction.hash;
  console.log("Hash: " + txHash);
}
