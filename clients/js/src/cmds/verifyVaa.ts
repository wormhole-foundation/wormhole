// The verify-vaa command invokes the parseAndVerifyVM method on the core contract on the specified EVM chain to verify the specified VAA.

import { Implementation__factory } from "@certusone/wormhole-sdk/lib/esm/ethers-contracts";
import {
  CONTRACTS,
  CHAINS,
  ChainName,
  assertChain,
  assertEVMChain,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { ethers } from "ethers";
import yargs from "yargs";
import { NETWORKS, NETWORK_OPTIONS } from "../consts";
import { assertNetwork } from "../utils";

export const command = "verify-vaa";
export const desc =
  "Verifies a VAA by querying the core contract on the specified EVM chain";
export const builder = (y: typeof yargs) =>
  y
    .option("vaa", {
      alias: "v",
      describe: "vaa in hex format",
      type: "string",
      demandOption: true,
    })
    .option("network", NETWORK_OPTIONS)
    .option("chain", {
      alias: "c",
      describe: "chain name",
      choices: Object.keys(CHAINS) as ChainName[],
      demandOption: true,
    } as const)
    .option("contract-address", {
      alias: "a",
      describe: "Contract to verify VAA on (override config)",
      type: "string",
      demandOption: false,
    })
    .option("rpc", {
      describe: "RPC endpoint",
      type: "string",
      demandOption: false,
    });
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  const chain = argv.chain;
  assertChain(chain);
  assertEVMChain(chain);

  const network = (argv.network ?? "mainnet").toUpperCase();
  assertNetwork(network);

  const rpc = argv.rpc ?? NETWORKS[network][chain].rpc;
  const contract_address =
    argv["contract-address"] ?? CONTRACTS[network][chain].core;
  if (!contract_address) {
    throw Error(`Unknown core contract on ${network} for ${chain}`);
  }
  const buf = Buffer.from(String(argv.vaa), "hex");
  const provider = new ethers.providers.JsonRpcProvider(rpc);
  const contract = Implementation__factory.connect(contract_address, provider);
  const result = await contract.parseAndVerifyVM(buf);
  if (result[1]) {
    console.log("Verification succeeded!");
  } else {
    console.log(`Verification failed: ${result[2]}`);
  }
};
