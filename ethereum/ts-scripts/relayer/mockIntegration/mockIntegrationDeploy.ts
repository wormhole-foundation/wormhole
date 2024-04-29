import {
  getMockIntegration,
  ChainInfo,
} from "../helpers/env";
import { buildOverrides } from "../helpers/deployments";

export interface EmitterRegistration {
  chainId: number;
  addr: string;
}

export async function registerMockIntegration(
  chain: ChainInfo,
  emitters: EmitterRegistration[],
) {
  console.log(`Registering emitters for chainId ${chain.chainId}`);
  const mockIntegration = await getMockIntegration(chain);

  const updateEmitters: EmitterRegistration[] = [];
  for (const emitter of emitters) {
    const currentEmitter = await mockIntegration.getRegisteredContract(
      emitter.chainId,
    );
    if (currentEmitter.toLowerCase() !== emitter.addr.toLowerCase()) {
      updateEmitters.push(emitter);
    }
  }

  if (updateEmitters.length > 0) {
    const overrides = await buildOverrides(
      () => mockIntegration.estimateGas.registerEmitters(updateEmitters),
      chain,
    );
    console.log(`About to send emitter registration for chain ${chain.chainId}`);
    const tx = await mockIntegration.registerEmitters(updateEmitters, overrides);
    const receipt = await tx.wait();

    if (receipt.status !== 1) {
      throw new Error(
        `Mock integration emitter registration failed for chain ${chain.chainId}, tx id ${tx.hash}`,
      );
    }
  }

  return { chain, updateEmitters };
}

export function printRegistration(emitters: EmitterRegistration[], chain: ChainInfo) {
  console.log(
    `MockIntegration emitters registered for chain ${chain.chainId}:`,
  );
  for (const emitter of emitters) {
    console.log(`  Target chain ${emitter.chainId}: ${emitter.addr}`);
  }
}