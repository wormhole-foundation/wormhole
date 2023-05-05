import {
  assertChain,
  assertEVMChain,
  CHAINS,
  CONTRACTS,
  isEVMChain,
  toChainName,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { ethers } from "ethers";
import { homedir } from "os";
import yargs from "yargs";
import {
  getImplementation,
  hijack_evm,
  query_contract_evm,
  setStorageAt,
} from "../evm";
import { NETWORKS } from "../networks";
import { runCommand, validator_args } from "../start-validator";
import { evm_address } from "../utils";

export const command = "evm";
export const desc = "EVM utilities";
export const builder = function (y: typeof yargs) {
  return y
    .option("rpc", {
      describe: "RPC endpoint",
      type: "string",
      required: false,
    })
    .command(
      "address-from-secret <secret>",
      "Compute a 20 byte eth address from a 32 byte private key",
      (yargs) => {
        return yargs.positional("secret", {
          type: "string",
          describe: "Secret key (32 bytes)",
        });
      },
      (argv) => {
        console.log(ethers.utils.computeAddress(argv["secret"]));
      }
    )
    .command(
      "storage-update",
      "Update a storage slot on an EVM fork during testing (anvil or hardhat)",
      (yargs) => {
        return yargs
          .option("contract-address", {
            alias: "a",
            describe: "Contract address",
            type: "string",
            required: true,
          })
          .option("storage-slot", {
            alias: "k",
            describe: "Storage slot to modify",
            type: "string",
            required: true,
          })
          .option("value", {
            alias: "v",
            describe: "Value to write into the slot (32 bytes)",
            type: "string",
            required: true,
          });
      },
      async (argv) => {
        const result = await setStorageAt(
          argv["rpc"],
          evm_address(argv["contract-address"]),
          argv["storage-slot"],
          ["uint256"],
          [argv["value"]]
        );
        console.log(result);
      }
    )
    .command("chains", "Return all EVM chains", async (_) => {
      console.log(
        Object.values(CHAINS)
          .map((id) => toChainName(id))
          .filter((name) => isEVMChain(name))
          .join(" ")
      );
    })
    .command(
      "info",
      "Query info about the on-chain state of the contract",
      (yargs) => {
        return yargs
          .option("chain", {
            alias: "c",
            describe: "Chain to query",
            type: "string",
            choices: Object.keys(CHAINS),
            required: true,
          })
          .option("module", {
            alias: "m",
            describe: "Module to query",
            type: "string",
            choices: ["Core", "NFTBridge", "TokenBridge"],
            required: true,
          })
          .option("network", {
            alias: "n",
            describe: "network",
            type: "string",
            choices: ["mainnet", "testnet", "devnet"],
            required: true,
          })
          .option("contract-address", {
            alias: "a",
            describe: "Contract to query (override config)",
            type: "string",
            required: false,
          })
          .option("implementation-only", {
            alias: "i",
            describe: "Only query implementation (faster)",
            type: "boolean",
            default: false,
            required: false,
          });
      },
      async (argv) => {
        assertChain(argv["chain"]);
        assertEVMChain(argv["chain"]);
        const network = argv.network.toUpperCase();
        if (
          network !== "MAINNET" &&
          network !== "TESTNET" &&
          network !== "DEVNET"
        ) {
          throw Error(`Unknown network: ${network}`);
        }
        let module = argv["module"] as "Core" | "NFTBridge" | "TokenBridge";
        let rpc = argv["rpc"] ?? NETWORKS[network][argv["chain"]].rpc;
        if (argv["implementation-only"]) {
          console.log(
            await getImplementation(
              network,
              argv["chain"],
              module,
              argv["contract-address"],
              rpc
            )
          );
        } else {
          console.log(
            JSON.stringify(
              await query_contract_evm(
                network,
                argv["chain"],
                module,
                argv["contract-address"],
                rpc
              ),
              null,
              2
            )
          );
        }
      }
    )
    .command(
      "hijack",
      "Override the guardian set of the core bridge contract during testing (anvil or hardhat)",
      (yargs) => {
        return yargs
          .option("core-contract-address", {
            alias: "a",
            describe: "Core contract address",
            type: "string",
            default: CONTRACTS.MAINNET.ethereum.core,
          })
          .option("guardian-address", {
            alias: "g",
            required: true,
            describe: "Guardians' public addresses (CSV)",
            type: "string",
          })
          .option("guardian-set-index", {
            alias: "i",
            required: false,
            describe:
              "New guardian set index (if unspecified, default to overriding the current index)",
            type: "number",
          });
      },
      async (argv) => {
        const guardian_addresses = argv["guardian-address"].split(",");
        let rpc = argv["rpc"] ?? NETWORKS.DEVNET.ethereum.rpc;
        await hijack_evm(
          rpc,
          argv["core-contract-address"],
          guardian_addresses,
          argv["guardian-set-index"]
        );
      }
    )
    .command(
      "start-validator",
      "Start a local EVM validator",
      (yargs) => {
        return yargs.option("validator-args", validator_args);
      },
      (argv) => {
        const cmd = `cd ${homedir()} && npx ganache-cli -e 10000 --deterministic --time="1970-01-01T00:00:00+00:00"`;
        runCommand(cmd, argv["validator-args"]);
      }
    )
    .strict()
    .demandCommand();
};

export const handler = (argv) => {};
