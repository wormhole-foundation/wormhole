import {
  env,
  getSigner,
  init,
  loadChains,
} from "../helpers/env";
import { readFileSync, writeFileSync } from "fs";
import { toChain } from "@wormhole-foundation/sdk-base";

const processName = "checkNetworks";

init();
const chains = loadChains();

async function main() {
  console.log(`Env: ${env}`);
  console.log(`Start ${processName}!`);

  console.log("Checking networks before deploying...");
  for (const chain of chains) {
    console.log(`Checking network ${chain.chainId}...`);
    const signer = await getSigner(chain);
    const network = await signer.provider?.getNetwork();
    const balance = await signer.getBalance();
    if (!network?.name || !balance) {
      console.log(
        "Failed to get network for chain " + chain.chainId + ". Exiting..."
      );
      process.exit(1);
    }
    console.log(`Balance ${balance.toString()}`);
    console.log(`Network ${toChain(chain.chainId)} (${chain.chainId}) checked`);
  }
  console.log("");
  console.log("Networks checked");
  console.log("");
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
