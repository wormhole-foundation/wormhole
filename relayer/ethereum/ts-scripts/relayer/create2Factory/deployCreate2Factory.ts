import {
  init,
  writeOutputFiles,
  getOperatingChains,
} from "../helpers/env";
import { deployCreate2Factory } from "../helpers/deployments";

const processName = "deployCreate2Factory";
init();
const operatingChains = getOperatingChains();

async function run() {
  console.log("Start!");

  const create2Factories = await Promise.all(
    operatingChains.map(deployCreate2Factory)
  );

  writeOutputFiles({ create2Factories }, processName);
}

run().then(() => console.log("Done!"));
