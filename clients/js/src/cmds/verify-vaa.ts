// The verify-vaa command invokes the parseAndVerifyVM method on the core contract on Ethereum to verify the specified VAA.

import { Implementation__factory } from "@certusone/wormhole-sdk/lib/esm/ethers-contracts";
import { CONTRACTS } from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { ethers } from "ethers";
import yargs from "yargs";
import { NETWORKS } from "../networks";

export const command = "verify-vaa";
export const desc = "Verifies a VAA by querying the core contract on Ethereum";
export const builder = (y: typeof yargs) => {
  return y
    .option("vaa", {
      alias: "v",
      describe: "vaa in hex format",
      type: "string",
      required: true,
    })
    .option("network", {
      alias: "n",
      describe: "network",
      type: "string",
      choices: ["mainnet", "testnet", "devnet"],
      required: true,
    });
};
export const handler = async (argv) => {
  const network = argv.network.toUpperCase();
  if (network !== "MAINNET" && network !== "TESTNET" && network !== "DEVNET") {
    throw Error(`Unknown network: ${network}`);
  }

  const buf = Buffer.from(String(argv.vaa), "hex");
  let n = NETWORKS[network]["ethereum"];
  let contract_address = CONTRACTS[network]["ethereum"].core;

  if (contract_address === undefined) {
    throw Error(`Unknown core contract on ${network} for ethereum`);
  }

  const provider = new ethers.providers.JsonRpcProvider(n.rpc);
  const contract = Implementation__factory.connect(contract_address, provider);
  const result = await contract.parseAndVerifyVM(buf);

  if (result[1]) {
    console.log("Verification succeeded!");
  } else {
    console.log(`Verification failed: ${result[2]}`);
  }
};
