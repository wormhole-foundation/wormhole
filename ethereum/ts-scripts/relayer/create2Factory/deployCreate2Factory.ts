import {
  init,
  writeOutputFiles,
  getOperationDescriptor,
  getCreate2FactoryAddress,
  Deployment,
} from "../helpers/env";
import { deployCreate2Factory } from "../helpers/deployments";

const processName = "deployCreate2Factory";
init();
const operation = getOperationDescriptor();

async function run() {
  console.log("Start!");

  const newDeployments = await Promise.all(
    operation.operatingChains.map(deployCreate2Factory),
  );

  const oldDeployments = operation.supportedChains.map((chain) => {
    return {
      chainId: chain.chainId,
      address: getCreate2FactoryAddress(chain),
    };
  });

  const create2Factories = oldDeployments.concat(
    newDeployments,
  ) satisfies Deployment[];

  writeOutputFiles({ create2Factories }, processName);
}

run().then(() => console.log("Done!"));
