import BN from "bn.js";
import { readFileSync } from "fs";
import { Account, KeyPair, connect } from "near-api-js";
import { InMemoryKeyStore } from "near-api-js/lib/key_stores";
import { parseSeedPhrase } from "near-seed-phrase";
import yargs from "yargs";
import { CONTRACTS, NETWORKS, NETWORK_OPTIONS, RPC_OPTIONS } from "../consts";
import { assertNetwork } from "../utils";

// Near utilities
export const command = "near";
export const desc = "NEAR utilities";
export const builder = function (y: typeof yargs) {
  return y
    .option("module", {
      alias: "m",
      describe: "Module to query",
      choices: ["Core", "NFTBridge", "TokenBridge"],
      demandOption: false,
    } as const)
    .option("network", NETWORK_OPTIONS)
    .option("account", {
      describe: "Near deployment account",
      type: "string",
      demandOption: true,
    })
    .option("attach", {
      describe: "Attach some near",
      type: "string",
      demandOption: false,
    })
    .option("target", {
      describe: "Near account to upgrade",
      type: "string",
      demandOption: false,
    })
    .option("mnemonic", {
      describe: "Near private keys",
      type: "string",
      demandOption: false,
    })
    .option("key", {
      describe: "Near private key",
      type: "string",
      demandOption: false,
    })
    .option("rpc", RPC_OPTIONS)
    .command(
      "contract-update <file>",
      "Submit a contract update using our specific APIs",
      (yargs) =>
        yargs.positional("file", {
          type: "string",
          describe: "wasm",
          demandOption: true,
        }),
      async (argv) => {
        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const contracts = CONTRACTS[network].near;
        const {
          rpc: defaultRpc,
          key: defaultKey,
          networkId,
        } = NETWORKS[network].near;

        const key =
          argv.key ??
          (argv.mnemonic && parseSeedPhrase(argv.mnemonic).secretKey) ??
          defaultKey;
        if (!key) {
          throw Error(`No ${network} key defined for NEAR`);
        }

        const rpc = argv.rpc ?? defaultRpc;
        if (!rpc) {
          throw Error(`No ${network} rpc defined for NEAR`);
        }

        let target = argv.target;
        if (!argv.target && argv.module) {
          if (argv.module === "Core") {
            target = contracts.core;
            console.log("Setting target to core");
          }

          if (argv.module === "TokenBridge") {
            target = contracts.token_bridge;
            console.log("Setting target to token_bridge");
          }
        }

        if (!target) {
          throw Error(`No target defined for NEAR`);
        }

        const masterKey = KeyPair.fromString(key);
        const keyStore = new InMemoryKeyStore();
        keyStore.setKey(networkId, argv.account, masterKey);
        const near = await connect({
          keyStore,
          networkId,
          nodeUrl: rpc,
          headers: {},
        });

        const masterAccount = new Account(near.connection, argv.account);
        const result = await masterAccount.functionCall({
          contractId: target,
          methodName: "update_contract",
          args: readFileSync(argv["file"]),
          attachedDeposit: new BN("22797900000000000000000000"),
          gas: new BN("300000000000000"),
        });
        console.log(result);
      }
    )
    .command(
      "deploy <file>",
      "Submit a contract update using near APIs",
      (yargs) =>
        yargs.positional("file", {
          type: "string",
          describe: "wasm",
          demandOption: true,
        }),
      async (argv) => {
        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const contracts = CONTRACTS[network].near;
        const {
          rpc: defaultRpc,
          key: defaultKey,
          networkId,
        } = NETWORKS[network].near;

        const key =
          argv.key ??
          (argv.mnemonic && parseSeedPhrase(argv.mnemonic).secretKey) ??
          defaultKey;
        if (!key) {
          throw Error(`No ${network} key defined for NEAR`);
        }

        const rpc = argv.rpc ?? defaultRpc;
        if (!rpc) {
          throw Error(`No ${network} rpc defined for NEAR`);
        }

        let target = argv.target;
        if (!argv.target && argv.module) {
          if (argv.module === "Core") {
            target = contracts.core;
            console.log("Setting target to core");
          }

          if (argv.module === "TokenBridge") {
            target = contracts.token_bridge;
            console.log("Setting target to token_bridge");
          }
        }

        if (!target) {
          throw Error(`No target defined for NEAR`);
        }

        const masterKey = KeyPair.fromString(key);
        const keyStore = new InMemoryKeyStore();
        keyStore.setKey(networkId, argv.account, masterKey);
        keyStore.setKey(networkId, target, masterKey);

        const near = await connect({
          keyStore,
          networkId: networkId,
          nodeUrl: rpc,
          headers: {},
        });
        const masterAccount = new Account(near.connection, argv.account);
        const targetAccount = new Account(near.connection, target);
        console.log({ ...argv, key, rpc, target });

        if (argv.attach) {
          console.log(
            `Sending money: ${target} from ${argv.account} being sent ${argv.attach}`
          );
          console.log(
            await masterAccount.sendMoney(target, new BN(argv.attach))
          );
        }

        console.log("deploying contract");
        console.log(
          await targetAccount.deployContract(readFileSync(argv.file))
        );
      }
    );
};
export const handler = () => {};
