import {
  init,
  loadChains,
  getMockIntegration,
  getMockIntegrationAddress,
  ChainInfo,
  getOperatingChains,
} from "../helpers/env";
import { buildOverrides } from "../helpers/deployments";
import { tryNativeToHexString } from "@certusone/wormhole-sdk";
import { XAddressStruct } from "../../../ethers-contracts/MockRelayerIntegration";

const processName = "registerMockIntegration";
init();
const allChains = loadChains();
const operatingChains = getOperatingChains();

interface EmitterRegistration {
  chainId: number;
  addr: string;
}

async function run() {
  console.log(`Start! ${processName}`);

  const emitters = allChains.map((chain) => ({
    chainId: chain.chainId,
    addr: "0x" + tryNativeToHexString(getMockIntegrationAddress(chain), "ethereum"),
  })) satisfies XAddressStruct[];

  const results = await Promise.allSettled(operatingChains.map(async (chain) => registerMockIntegration(chain, emitters)));

  for (const result of results) {
    if (result.status === "rejected") {
      console.log(
        `Price update failed: ${result.reason?.stack || result.reason}`,
      );
    } else {
      printUpdate(result.value.updateEmitters, result.value.chain);
    }
  }
}

async function registerMockIntegration(chain: ChainInfo, emitters: EmitterRegistration[]) {
  console.log(`Registering emitters for chainId ${chain.chainId}`);
  const mockIntegration = await getMockIntegration(chain);

  const updateEmitters: EmitterRegistration[] = [];
  for (const emitter of emitters) {
    const currentEmitter = await mockIntegration.getRegisteredContract(emitter.chainId);
    if (currentEmitter.toLowerCase() !== emitter.addr.toLowerCase()) {
      updateEmitters.push(emitter);
    }
  }

  const overrides = await buildOverrides(
    () => mockIntegration.estimateGas.registerEmitters(updateEmitters),
    chain,
  );
  console.log(`About to send emitter registration for chain ${chain.chainId}`);
  const tx = await mockIntegration.registerEmitters(updateEmitters, overrides);
  const receipt = await tx.wait();

  if (receipt.status !== 1) {
    throw new Error(`Mock integration emitter registration failed for chain ${chain.chainId}, tx id ${tx.hash}`);
  }

  return { chain, updateEmitters };
}

function printUpdate(emitters: EmitterRegistration[], chain: ChainInfo) {
  console.log(`MockIntegration emitters registered for chain ${chain.chainId}:`);
  for (const emitter of emitters) {
    console.log(`  Target chain ${emitter.chainId}: ${emitter.addr}`);
  }
}

run().then(() => console.log(`Done! ${processName}`));
