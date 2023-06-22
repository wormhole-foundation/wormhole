import {
  assertChain,
  assertEVMChain,
  ChainName,
  CHAINS,
  CONTRACTS,
  isEVMChain,
  toChainName,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { ethers } from "ethers";
import { homedir } from "os";
import yargs from "yargs";
import { NETWORK_OPTIONS, NETWORKS } from "../consts";
import {
  getImplementation,
  hijack_evm,
  query_contract_evm,
  setStorageAt,
} from "../evm";
import { runCommand, VALIDATOR_OPTIONS } from "../startValidator";
import { assertNetwork, evm_address } from "../utils";

export const command = "evm";
export const desc = "EVM utilities";
export const builder = function (y: typeof yargs) {
  return y
    .option("rpc", {
      describe: "RPC endpoint",
      type: "string",
      demandOption: false,
    })
    .command(
      "address-from-secret <secret>",
      "Compute a 20 byte eth address from a 32 byte private key",
      (yargs) =>
        yargs.positional("secret", {
          type: "string",
          describe: "Secret key (32 bytes)",
          demandOption: true,
        } as const),
      (argv) => {
        console.log(ethers.utils.computeAddress(argv["secret"]));
      }
    )
    .command(
      "storage-update",
      "Update a storage slot on an EVM fork during testing (anvil or hardhat)",
      (yargs) =>
        yargs
          .option("contract-address", {
            alias: "a",
            describe: "Contract address",
            type: "string",
            demandOption: true,
          })
          .option("storage-slot", {
            alias: "k",
            describe: "Storage slot to modify",
            type: "string",
            demandOption: true,
          })
          .option("value", {
            alias: "v",
            describe: "Value to write into the slot (32 bytes)",
            type: "string",
            demandOption: true,
          }),
      async (argv) => {
        if (!argv.rpc) {
          throw new Error("RPC required");
        }

        const result = await setStorageAt(
          argv.rpc,
          evm_address(argv["contract-address"]),
          argv["storage-slot"],
          ["uint256"],
          [argv.value]
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
      (yargs) =>
        yargs
          .option("chain", {
            alias: "c",
            describe: "Chain to query",
            choices: Object.keys(CHAINS) as ChainName[],
            demandOption: true,
          } as const)
          .option("module", {
            alias: "m",
            describe: "Module to query",
            choices: ["Core", "NFTBridge", "TokenBridge"],
            demandOption: true,
          } as const)
          .option("network", NETWORK_OPTIONS)
          .option("contract-address", {
            alias: "a",
            describe: "Contract to query (override config)",
            type: "string",
            demandOption: false,
          })
          .option("implementation-only", {
            alias: "i",
            describe: "Only query implementation (faster)",
            type: "boolean",
            default: false,
            demandOption: false,
          }),
      async (argv) => {
        const chain = argv.chain;
        assertChain(chain);
        assertEVMChain(chain);
        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const module = argv.module;
        const rpc = argv.rpc ?? NETWORKS[network][chain].rpc;
        if (argv["implementation-only"]) {
          console.log(
            await getImplementation(
              network,
              chain,
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
                chain,
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
      (yargs) =>
        yargs
          .option("core-contract-address", {
            alias: "a",
            describe: "Core contract address",
            type: "string",
            default: CONTRACTS.MAINNET.ethereum.core,
          })
          .option("guardian-address", {
            alias: "g",
            demandOption: true,
            describe: "Guardians' public addresses (CSV)",
            type: "string",
          })
          .option("guardian-set-index", {
            alias: "i",
            demandOption: false,
            describe:
              "New guardian set index (if unspecified, default to overriding the current index)",
            type: "number",
          }),
      async (argv) => {
        const guardian_addresses = argv["guardian-address"].split(",");
        let rpc = argv.rpc ?? NETWORKS.DEVNET.ethereum.rpc;
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
      (yargs) => yargs.option("validator-args", VALIDATOR_OPTIONS),
      (argv) => {
        const cmd = `cd ${homedir()} && npx ganache-cli --wallet.defaultBalance 10000 --wallet.deterministic --chain.time="1970-01-01T00:00:00+00:00"`;
        runCommand(cmd, argv["validator-args"]);
      }
    )
    .strict()
    .demandCommand();
};
export const handler = () => {};
