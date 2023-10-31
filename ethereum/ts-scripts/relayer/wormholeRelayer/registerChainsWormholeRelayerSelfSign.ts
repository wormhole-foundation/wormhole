import { tryNativeToHexString } from "@certusone/wormhole-sdk";
import {
  init,
  loadChains,
  ChainInfo,
  getWormholeRelayer,
  getOperationDescriptor,
  getWormholeRelayerAddress,
} from "../helpers/env";
import { buildOverrides } from "../helpers/deployments";
import { wait } from "../helpers/utils";
import { createRegisterChainVAA } from "../helpers/vaa";
import type { WormholeRelayer } from "../../../ethers-contracts";

const processName = "registerChainsWormholeRelayerSelfSign";
init();
const operation = getOperationDescriptor();
const allChains = loadChains();

const zeroBytes32 =
  "0x0000000000000000000000000000000000000000000000000000000000000000";

async function run() {
  console.log("Start! " + processName);

  // TODO: to send txs concurrently, the cross-registrations need to be separated out
  // for (const operatingChain of operation.operatingChains) {
  // await registerChainsWormholeRelayer(operatingChain);
  // await registerOnExistingChainsWormholeRelayer(operatingChain);
  //}
  for (const myChain of allChains) {
    registerChainsWormholeRelayerIfUnregistered(myChain);
  }
}

async function registerChainsWormholeRelayerIfUnregistered(
  operatingChain: ChainInfo
) {
  console.log(
    "[START] Registering all the wormhole relayers onto Wormhole Relayer " +
      operatingChain.chainId
  );

  const wormholeRelayer = await getWormholeRelayer(operatingChain);
  for (const targetChain of allChains) {
    let currentRegisteredContract;
    try {
      currentRegisteredContract = (await wormholeRelayer.getRegisteredWormholeRelayerContract(
        targetChain.chainId
      ))
    } catch (e) {
      console.log(`Error getting the wormhole relayer for chain ${operatingChain.chainId}, rpc ${operatingChain.rpc}`)
    }
    if (
       currentRegisteredContract === zeroBytes32
    ) {
      console.log(
        `[start] This chain ${targetChain.chainId} is not registered onto chain ${operatingChain.chainId} yet`
      );
      try {
        await registerWormholeRelayer(
          wormholeRelayer,
          operatingChain,
          targetChain
        );
      } catch(error) {
        console.log(`[error] Registering ${targetChain.chainId} onto ${operatingChain.chainId} failed`)
        console.log(`The error was ${error}`)
      }
      console.log(
        `[done] Now this chain ${targetChain.chainId} is registered onto chain ${operatingChain.chainId}`
      );
    } else {
      // This doesn't check that the registered address is correct - only that it exists and is not zero
      console.log(
        `This chain ${targetChain.chainId} is already registered onto chain ${operatingChain.chainId}`
      );
    }
  }

  console.log(
    "Did all contract registrations for the core relayer on " +
      operatingChain.chainId
  );
}

async function registerChainsWormholeRelayer(operatingChain: ChainInfo) {
  console.log(
    "Registering all the wormhole relayers onto Wormhole Relayer " +
      operatingChain.chainId
  );

  const wormholeRelayer = await getWormholeRelayer(operatingChain);
  for (const targetChain of allChains) {
    await registerWormholeRelayer(wormholeRelayer, operatingChain, targetChain);
  }

  console.log(
    "Did all contract registrations for the core relayer on " +
      operatingChain.chainId
  );
}

async function registerOnExistingChainsWormholeRelayer(targetChain: ChainInfo) {
  console.log(
    "Registering Wormhole Relayer " +
      targetChain.chainId +
      " onto all the wormhole relayers"
  );
  const tasks = await Promise.allSettled(
    operation.supportedChains.map(async (operatingChain) => {
      const coreRelayer = await getWormholeRelayer(operatingChain);

      return registerWormholeRelayer(coreRelayer, operatingChain, targetChain);
    })
  );
  for (const task of tasks) {
    if (task.status === "rejected") {
      console.log(
        `Failed cross registration. ${task.reason?.stack || task.reason}`
      );
    }
  }

  console.log(
    "Did all contract registrations of the core relayer on " +
      targetChain.chainId +
      " onto the existing (non operating) chains"
  );
}

async function registerWormholeRelayer(
  wormholeRelayer: WormholeRelayer,
  operatingChain: ChainInfo,
  targetChain: ChainInfo
) {
  const registration =
    await wormholeRelayer.getRegisteredWormholeRelayerContract(
      targetChain.chainId
    );
  if (registration !== zeroBytes32) {
    const registrationAddress = await getWormholeRelayerAddress(targetChain);
    const expectedRegistration =
      "0x" + tryNativeToHexString(registrationAddress, "ethereum");
    if (registration !== expectedRegistration) {
      throw new Error(`Found an unexpected registration for chain ${targetChain.chainId} on chain ${operatingChain.chainId}
Expected: ${expectedRegistration}
Actual: ${registration}`);
    }

    console.log(
      `Chain ${targetChain.chainId} on chain ${operatingChain.chainId} is already registered`
    );
    return;
  }

  const vaa = await createRegisterChainVAA(targetChain);

  console.log(
    `Registering chain ${targetChain.chainId} onto chain ${operatingChain.chainId}`
  );
  try {
    const overrides = await buildOverrides(
      () => wormholeRelayer.estimateGas.registerWormholeRelayerContract(vaa),
      operatingChain
    );
    await wormholeRelayer
      .registerWormholeRelayerContract(vaa, overrides)
      .then(wait);
  } catch (error) {
    console.log(
      `Error in registering chain ${targetChain.chainId} onto ${operatingChain.chainId}`
    );
    console.log((error as any)?.stack || error);
  }
}

run().then(() => console.log("Done! " + processName));
