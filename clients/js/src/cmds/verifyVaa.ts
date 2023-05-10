// The verify-vaa command invokes the parseAndVerifyVM method on the core contract on Ethereum to verify the specified VAA.

import { Implementation__factory } from "@certusone/wormhole-sdk/lib/esm/ethers-contracts";
import { CONTRACTS } from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { ethers } from "ethers";
import yargs from "yargs";
import { NETWORKS, NETWORK_OPTIONS } from "../consts";
import { assertNetwork } from "../utils";

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
    .option("network", NETWORK_OPTIONS);
export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  const network = argv.network.toUpperCase();
  assertNetwork(network);

  const buf = Buffer.from(String(argv.vaa), "hex");
  const contract_address = CONTRACTS[network].ethereum.core;
  if (!contract_address) {
    throw Error(`Unknown core contract on ${network} for ethereum`);
  }

  const provider = new ethers.providers.JsonRpcProvider(
    NETWORKS[network].ethereum.rpc
  );
  const contract = Implementation__factory.connect(contract_address, provider);
  const result = await contract.parseAndVerifyVM(buf);
  if (result[1]) {
    console.log("Verification succeeded!");
  } else {
    console.log(`Verification failed: ${result[2]}`);
  }
};
