import { OfflineSigner, Registry } from "@cosmjs/proto-signing";
import {
  defaultRegistryTypes,
  SigningStargateClient,
  SigningStargateClientOptions,
  StargateClient,
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

  //The signAndBroadcast function needs to be undefined here because it uses and interface which can't be
  //resolved by typescript.
  const coreShell = {
    ...coreClient,
    signAndBroadcast: undefined,
  };
  delete coreShell.signAndBroadcast;

  const tokenBridgeShell = {
    ...tokenBridgeClient,
    signAndBroadcast: undefined,
  };
  delete tokenBridgeShell.signAndBroadcast;

  type CoreType = Omit<typeof coreShell, "signAndBroadcast">;
  type TokenBridgeType = Omit<typeof tokenBridgeShell, "signAndBroadcast">;
  type WormholeFunctions = {
    core: CoreType;
    tokenbridge: TokenBridgeType;
  };
  type WormholeSigningClient = SigningStargateClient & WormholeFunctions;

  //@ts-ignore
  client.core = coreShell;
  //@ts-ignore
  client.tokenbridge = tokenBridgeShell;

  return client as WormholeSigningClient;
};
