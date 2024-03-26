import { tryNativeToHexString } from "@certusone/wormhole-sdk";
import { inspect } from "util";

import {
  init,
  loadChains,
  ChainInfo,
  getWormholeRelayer,
  getWormholeRelayerAddress,
} from "../helpers/env";
import { buildOverrides } from "../helpers/deployments";
import { wait } from "../helpers/utils";
import { createRegisterChainVAA } from "../helpers/vaa";
import type { WormholeRelayer } from "../../../ethers-contracts";

const processName = "registerChainsWormholeRelayerSelfSign";
init();
const allChains = loadChains();

const zeroBytes32 =
  "0x0000000000000000000000000000000000000000000000000000000000000000";

async function run() {
  console.log("Start! " + processName);

  const results = await Promise.allSettled(allChains.map((chain) => registerChainsWormholeRelayerIfUnregistered(chain)));
  for (const result of results) {
    if (result.status === "rejected") {
      console.log(
        `Registration failed: ${result.reason?.stack || inspect(result.reason)}`
      );
    }
  }
}

async function registerChainsWormholeRelayerIfUnregistered(
  operatingChain: ChainInfo
) {
  console.log(
    `Registering all the WormholeRelayer contracts in chain ${operatingChain.chainId}`
  );

  const wormholeRelayer = await getWormholeRelayer(operatingChain);
  for (const targetChain of allChains) {
    await registerWormholeRelayer(
      wormholeRelayer,
      operatingChain,
      targetChain
    );
  }

  console.log(
    `Did all contract registrations for the WormholeRelayer in chain ${operatingChain.chainId}`
  );
}

async function registerWormholeRelayer(
  wormholeRelayer: WormholeRelayer,
  operatingChain: ChainInfo,
  targetChain: ChainInfo
) {
  const registration = await wormholeRelayer.getRegisteredWormholeRelayerContract(targetChain.chainId);
  if (registration !== zeroBytes32) {
    const registrationAddress = await getWormholeRelayerAddress(targetChain);
    const expectedRegistration =
      "0x" + tryNativeToHexString(registrationAddress, "ethereum");
    if (registration.toLowerCase() !== expectedRegistration.toLowerCase()) {
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
    throw new Error(
      `Error in registering chain ${targetChain.chainId} onto ${operatingChain.chainId}
Details: ${(error as any)?.stack || inspect(error)}`
    );
  }
}

run().then(() => console.log("Done! " + processName));
