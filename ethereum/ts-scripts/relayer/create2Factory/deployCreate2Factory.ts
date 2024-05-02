import { inspect } from "util";
import {
  init,
  saveDeployments,
  getOperationDescriptor,
  Deployment,
} from "../helpers/env";
import { deployCreate2Factory } from "../helpers/deployments";

const processName = "deployCreate2Factory";
init();
const operation = getOperationDescriptor();

async function run() {
  console.log("Start!");

  const tasks = await Promise.allSettled(
    operation.operatingChains.map(deployCreate2Factory),
  );
  const create2Factories: Deployment[] = [];

  let failed = false;
  for (const task of tasks) {
    if (task.status === "rejected") {
      // TODO: add chain as context
      // These get discarded and need to be retried later with a separate invocation.
      console.log(
        `Deployment failed: ${task.reason?.stack || inspect(task.reason)}`,
      );
      failed = true;
    } else {
      create2Factories.push(task.value);
    }
  }

  saveDeployments({ create2Factories }, processName);

  // We throw here to ensure non zero exit code and communicate failure to shell
  if (failed) {
    throw new Error("One or more errors happened during execution. See messages above.");
  }
}

run().then(() => console.log("Done!"));
