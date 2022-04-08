import { OfflineSigner, Registry } from "@cosmjs/proto-signing";
import {
  defaultRegistryTypes,
  SigningStargateClient,
  SigningStargateClientOptions,
} from "@cosmjs/stargate";
import * as tokenBridgeModule from "../modules/certusone.wormholechain.tokenbridge";
import * as coreModule from "../modules/certusone.wormholechain.wormhole";

//Rip the types out of their modules. These are private fields on the module.
//@ts-ignore
const coreTypes = coreModule.registry.types;
//@ts-ignore
const tokenBridgeTypes = tokenBridgeModule.registry.types;

const aggregateTypes = [
  ...coreTypes,
  ...tokenBridgeTypes,
  ...defaultRegistryTypes, //There are no interface-level changes to the default modules at this time
];

export const MissingWalletError = new Error("wallet is required");

const registry = new Registry(<any>aggregateTypes);

export const getWormchainSigningClient = async (
  tendermintAddress: string,
  wallet: OfflineSigner,
  options?: SigningStargateClientOptions
) => {
  if (!wallet) throw MissingWalletError;

  const coreClient = await coreModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const tokenBridgeClient = await tokenBridgeModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const client = await SigningStargateClient.connectWithSigner(
    tendermintAddress,
    wallet,
    {
      ...options,
      registry,
    }
  );

  //We have to declare signAndBroadcast as undefined here, because it uses a type which the typescript compiler can't resolve.
  //It's unused, so this is fine.
  const wormholeFunctions = {
    ...coreClient,
    ...tokenBridgeClient,
    signAndBroadcast: undefined,
  };

  const fields = Object.keys(wormholeFunctions);
  fields.forEach((key) => {
    //We do not want to overwrite signAndBroadcast from the main client.
    if (key !== "signAndBroadcast") {
      //@ts-ignore
      client[key] = wormholeFunctions[key];
    }
  });

  //We have to put together the output type, as typescript will not be able to interpolate it.
  //The Wormholefunction signAndBroadcast is now an undefined, so we want the stargate signer's type to override it.
  type WormholeFunction = Omit<typeof wormholeFunctions, "signAndBroadcast">;
  type WormholeSigningClient = WormholeFunction & SigningStargateClient;
  const combinedClient: WormholeSigningClient = client as WormholeSigningClient;

  return combinedClient;
};
