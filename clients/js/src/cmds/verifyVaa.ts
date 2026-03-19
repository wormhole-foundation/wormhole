// The verify-vaa command invokes the parseAndVerifyVM method on the core contract on Ethereum to verify the specified VAA.

import { Implementation__factory } from "@certusone/wormhole-sdk/lib/esm/ethers-contracts";
import { ethers } from "ethers";
import yargs from "yargs";
import { NETWORKS, NETWORK_OPTIONS } from "../consts";
import { assertEVMChain, getNetwork } from "../utils";
import { contracts } from "@wormhole-foundation/sdk";
import { toChain } from "@wormhole-foundation/sdk-base";

export const command = "verify-vaa";
export const desc = "Verifies a VAA by querying the core contract on Ethereum";
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
      describe:
        "Chain to verify on (e.g., Sepolia, ArbitrumSepolia, BaseSepolia)",
      type: "string",
    })
    .option("rpc", {
      describe: "Custom RPC endpoint (overrides network default)",
      type: "string",
    });
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  const network = getNetwork(argv.network);

  // Determine which chain to use
  const chainName =
    argv.chain || (network === "Testnet" ? "Sepolia" : "Ethereum");
  const chain = toChain(chainName);
  assertEVMChain(chain);

  const buf = Buffer.from(String(argv.vaa), "hex");
  const contract_address = contracts.coreBridge.get(network, chain);
  if (!contract_address) {
    throw Error(`Unknown core contract on ${network} for ${chain}`);
  }

  // Get RPC from argv, or from NETWORKS config for the specified chain
  const rpc = argv.rpc || (NETWORKS[network] as any)[chain]?.rpc;
  if (!rpc) {
    throw Error(`No RPC defined for ${chain} on ${network}`);
  }

  const provider = new ethers.providers.JsonRpcProvider(rpc);
  const contract = Implementation__factory.connect(contract_address, provider);
  const result = await contract.parseAndVerifyVM(buf);
  if (result[1]) {
    console.log("Verification succeeded!");
  } else {
    console.log(`Verification failed: ${result[2]}`);
  }
};
