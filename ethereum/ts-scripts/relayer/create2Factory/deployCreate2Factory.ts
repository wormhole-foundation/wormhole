import {
  init,
  loadChains,
  writeOutputFiles,
  getMockIntegration,
  Deployment,
  getOperatingChains,
  getMockIntegrationAddress,
} from "../helpers/env";
import { deployCreate2Factory } from "../helpers/deployments";
import { BigNumberish, BytesLike } from "ethers";
import { tryNativeToHexString } from "@certusone/wormhole-sdk";
import { wait } from "../helpers/utils";

const processName = "deployCreate2Factory";
init();
const chains = loadChains();
const operatingChains = getOperatingChains();

async function run() {
  console.log("Start!");

  const create2Factories = await Promise.all(
    operatingChains.map(deployCreate2Factory)
  );

  writeOutputFiles({ create2Factories }, processName);
}

run().then(() => console.log("Done!"));
