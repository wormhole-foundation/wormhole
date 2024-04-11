import {
  env,
  getSigner,
  init,
  loadChains,
  loadPrivateKey,
} from "../helpers/env";
import { readFileSync, writeFileSync } from "fs";

const processName = "checkNetworks";

init();
const chains = loadChains();

async function main() {
  console.log(`Env: ${env}`);
  console.log(`Start ${processName}!`);

  console.log("Checking networks before deploying...");
  for (const chain of chains) {
    const signer = getSigner(chain);
    const network = await signer.provider?.getNetwork();
    const balance = await signer.getBalance();
    if (!network?.name || !balance) {
      console.log(
        "Failed to get network for chain " + chain.chainId + ". Exiting..."
      );
      process.exit(1);
    }
    console.log(`Balance ${balance.toString()}`);
    console.log(`Network ${network.name} checked`);
  }
  console.log("");
  console.log("Networks checked");
  console.log("");

  if (process.argv.find((arg) => arg == "--set-last-run")) {
    const path = `./ts-scripts/relayer/config/${env}/contracts.json`;
    const contractsFile = readFileSync(path);
    if (!contractsFile) {
      throw Error("Failed to find contracts file for this process!");
    }
    const contracts = JSON.parse(contractsFile.toString());
    contracts.useLastRun = true;
    writeFileSync(path, JSON.stringify(contracts, undefined, 2));
  }
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
