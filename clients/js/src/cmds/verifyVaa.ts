// The verify-vaa command invokes the parseAndVerifyVM method on the core contract on Ethereum to verify the specified VAA.

import { Implementation__factory } from "@certusone/wormhole-sdk/lib/esm/ethers-contracts";
import { ethers } from "ethers";
import yargs from "yargs";
import { NETWORKS, NETWORK_OPTIONS } from "../consts";
import { getNetwork } from "../utils";
import { contracts } from "@wormhole-foundation/sdk";

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
    .option("rpc", {
      describe: "Custom RPC endpoint (overrides network default)",
      type: "string",
    });
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  const network = getNetwork(argv.network);

  const buf = Buffer.from(String(argv.vaa), "hex");
  const contract_address =
    network === "Testnet"
      ? contracts.coreBridge(network, "Sepolia")
      : contracts.coreBridge(network, "Ethereum");
  if (!contract_address) {
    throw Error(`Unknown core contract on ${network} for ethereum`);
  }

  const provider = new ethers.providers.JsonRpcProvider(
    argv.rpc || (network === "Testnet"
      ? NETWORKS[network].Sepolia.rpc
      : NETWORKS[network].Ethereum.rpc)
  );
  const contract = Implementation__factory.connect(contract_address, provider);
  const result = await contract.parseAndVerifyVM(buf);
  if (result[1]) {
    console.log("Verification succeeded!");
  } else {
    console.log(`Verification failed: ${result[2]}`);
  }
};
