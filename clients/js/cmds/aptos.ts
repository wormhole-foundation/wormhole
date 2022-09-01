import { CHAIN_ID_APTOS, CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
import { AptosAccount, BCS, FaucetClient } from "aptos";
import { ethers } from "ethers";
import yargs from "yargs";
import { callEntryFunc } from "../aptos";

type Network = "MAINNET" | "TESTNET" | "DEVNET"

const network_options = {
  alias: "n",
  describe: "network",
  type: "string",
  choices: ["mainnet", "testnet", "devnet"],
  required: true,
} as const;

// TODO(csongor): this could be useful elsewhere
function assertNetwork(n: string): asserts n is Network {
  if (
    n !== "MAINNET" &&
    n !== "TESTNET" &&
    n !== "DEVNET"
  ) {
    throw Error(`Unknown network: ${n}`);
  }
}

exports.command = 'aptos';
exports.desc = 'Aptos utilities ';
exports.builder = function (y: typeof yargs) {
  return y
    .command("init-wormhole", "Init Wormhole core contract", (yargs) => {
      return yargs
        .option("network", network_options)
        .option("chain-id", {
          describe: "Chain id",
          type: "number",
          default: CHAIN_ID_APTOS,
          required: false
        })
        .option("governance-chain-id", {
          describe: "Governance chain id",
          type: "number",
          default: CHAIN_ID_SOLANA,
          required: false
        })
        .option("governance-address", {
          describe: "Governance address",
          type: "string",
          default: "0x0000000000000000000000000000000000000000000000000000000000000004",
          required: false
        })
        // TODO(csongor): once the sdk has this, just use it from there
        .option("contract-address", {
          alias: "a",
          required: true,
          describe: "Address where the wormhole module is deployed",
          type: "string",
        })
        .option("guardian-address", {
          alias: "g",
          required: true,
          describe: "Initial guardian's address",
          type: "string",
        })
    }, async (argv) => {
      const network = argv.network.toUpperCase();
      assertNetwork(network);

      const contract_address = evm_address(argv["contract-address"]);
      const guardian_address = evm_address(argv["guardian-address"]).substring(24);
      const chain_id = argv["chain-id"];
      const governance_address = evm_address(argv["governance-address"]);
      const governance_chain_id = argv["governance-chain-id"];

      const args = [
        BCS.bcsSerializeUint64(chain_id),
        BCS.bcsSerializeUint64(governance_chain_id),
        BCS.bcsSerializeBytes(Buffer.from(governance_address, "hex")),
        BCS.bcsSerializeBytes(Buffer.from(guardian_address, "hex"))
      ]

      await callEntryFunc(network, `${contract_address}::wormhole`, "init", args);
    })
    .command("deploy", "Deploy an Aptos package", (_yargs) => {
    }, (argv) => {
      console.log("hi")
    })
    .command("upgrade", "Upgrade an Aptos package", (_yargs) => {
    }, (argv) => {
      console.log("hi")
    })
    .command("faucet", "Request money from the faucet for the deployer wallet (only local validator)", (_yargs) => {
    }, async (_argv) => {
      const NODE_URL = "http://0.0.0.0:8080/v1";
      const FAUCET_URL = "http://0.0.0.0:8081";
      const faucetClient = new FaucetClient(NODE_URL, FAUCET_URL);

      const coins = 100000;
      await faucetClient.fundAccount("0x277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b", coins);
      console.log(`Funded account with ${coins} coins`);
    })
    .strict().demandCommand();
}

function hex(x: string): string {
  return ethers.utils.hexlify(x, { allowMissingPrefix: true });
}
function evm_address(x: string): string {
  return hex(x).substring(2).padStart(64, "0");
}
