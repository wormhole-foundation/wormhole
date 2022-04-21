import { OfflineSigner, Registry } from "@cosmjs/proto-signing";
import {
  defaultRegistryTypes,
  SigningStargateClient,
  SigningStargateClientOptions,
  StargateClient,
} from "@cosmjs/stargate";
import { getTypeParameterOwner } from "typescript";
import * as tokenBridgeModule from "../modules/certusone.wormholechain.tokenbridge";
import * as coreModule from "../modules/certusone.wormholechain.wormhole";
import * as authzModule from "../modules/cosmos.authz.v1beta1";
import * as bankModule from "../modules/cosmos.bank.v1beta1";
import * as crisisModule from "../modules/cosmos.crisis.v1beta1";
import * as distributionModule from "../modules/cosmos.distribution.v1beta1";
import * as evidenceModule from "../modules/cosmos.evidence.v1beta1";
import * as feegrantModule from "../modules/cosmos.feegrant.v1beta1";
import * as govModule from "../modules/cosmos.gov.v1beta1";
import * as slashingModule from "../modules/cosmos.slashing.v1beta1";
import * as stakingModule from "../modules/cosmos.staking.v1beta1";
import * as vestingModule from "../modules/cosmos.vesting.v1beta1";

//Rip the types out of their modules. These are private fields on the module.
//@ts-ignore
const coreTypes = coreModule.registry.types;
//@ts-ignore
const tokenBridgeTypes = tokenBridgeModule.registry.types;
//@ts-ignore
const authzTypes = authzModule.registry.types;
//@ts-ignore
const bankTypes = bankModule.registry.types;
//@ts-ignore
const crisisTypes = crisisModule.registry.types;
//@ts-ignore
const distributionTypes = distributionModule.registry.types;
//@ts-ignore
const evidenceTypes = evidenceModule.registry.types;
//@ts-ignore
const feegrantTypes = feegrantModule.registry.types;
//@ts-ignore
const govTypes = govModule.registry.types;
//@ts-ignore
const slashingTypes = slashingModule.registry.types;
//@ts-ignore
const stakingTypes = stakingModule.registry.types;
//@ts-ignore
const vestingTypes = vestingModule.registry.types;

const aggregateTypes = [
  ...defaultRegistryTypes, //There are no interface-level changes to the default modules at this time
  ...coreTypes,
  ...tokenBridgeTypes,
  ...authzTypes,
  ...bankTypes,
  ...crisisTypes,
  ...distributionTypes,
  ...evidenceTypes,
  ...feegrantTypes,
  ...govTypes,
  ...slashingTypes,
  ...stakingTypes,
  ...vestingTypes,
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

  const authzClient = await authzModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const bankClient = await bankModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const crisisClient = await crisisModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const distributionClient = await distributionModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const evidenceClient = await evidenceModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const feegrantClient = await feegrantModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const govClient = await govModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const slashingClient = await slashingModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const stakingClient = await stakingModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const vestingClient = await vestingModule.txClient(wallet, {
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

  //This has some relevant info, but doesn't get us all the way there
  //https://github.com/cosmos/cosmjs/pull/712/files

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

  const authzShell = {
    ...authzClient,
    signAndBroadcast: undefined,
  };
  delete authzShell.signAndBroadcast;

  const bankShell = {
    ...bankClient,
    signAndBroadcast: undefined,
  };
  delete bankShell.signAndBroadcast;

  const crisisShell = {
    ...crisisClient,
    signAndBroadcast: undefined,
  };
  delete crisisShell.signAndBroadcast;

  const distributionShell = {
    ...distributionClient,
    signAndBroadcast: undefined,
  };
  delete distributionShell.signAndBroadcast;

  const evidenceShell = {
    ...evidenceClient,
    signAndBroadcast: undefined,
  };
  delete evidenceShell.signAndBroadcast;

  const feegrantShell = {
    ...feegrantClient,
    signAndBroadcast: undefined,
  };
  delete feegrantShell.signAndBroadcast;

  const govShell = {
    ...govClient,
    signAndBroadcast: undefined,
  };
  delete govShell.signAndBroadcast;

  const slashingShell = {
    ...slashingClient,
    signAndBroadcast: undefined,
  };
  delete slashingShell.signAndBroadcast;

  const stakingShell = {
    ...stakingClient,
    signAndBroadcast: undefined,
  };
  delete stakingShell.signAndBroadcast;

  const vestingShell = {
    ...vestingClient,
    signAndBroadcast: undefined,
  };
  delete vestingShell.signAndBroadcast;

  type CoreType = Omit<typeof coreShell, "signAndBroadcast">;
  type TokenBridgeType = Omit<typeof tokenBridgeShell, "signAndBroadcast">;
  type AuthzType = Omit<typeof authzShell, "signAndBroadcast">;
  type BankType = Omit<typeof bankShell, "signAndBroadcast">;
  type CrisisType = Omit<typeof crisisShell, "signAndBroadcast">;
  type DistributionType = Omit<typeof distributionShell, "signAndBroadcast">;
  type EvidenceType = Omit<typeof evidenceShell, "signAndBroadcast">;
  type FeegrantType = Omit<typeof feegrantShell, "signAndBroadcast">;
  type GovType = Omit<typeof govShell, "signAndBroadcast">;
  type SlashingType = Omit<typeof slashingShell, "signAndBroadcast">;
  type StakingType = Omit<typeof stakingShell, "signAndBroadcast">;
  type VestingType = Omit<typeof vestingShell, "signAndBroadcast">;
  type WormholeFunctions = {
    core: CoreType;
    tokenbridge: TokenBridgeType;
    authz: AuthzType;
    bank: BankType;
    crisis: CrisisType;
    distribution: DistributionType;
    evidence: EvidenceType;
    feegrant: FeegrantType;
    gov: GovType;
    slashing: SlashingType;
    staking: StakingType;
    vesting: VestingType;
  };
  type WormholeSigningClient = SigningStargateClient & WormholeFunctions;

  const output: WormholeSigningClient = client as WormholeSigningClient;

  output.core = coreShell;
  output.tokenbridge = tokenBridgeShell;
  output.bank = bankShell;
  output.crisis = crisisShell;
  output.distribution = distributionShell;
  output.evidence = evidenceShell;
  output.feegrant = feegrantShell;
  output.gov = govShell;
  output.slashing = slashingShell;
  output.staking = stakingShell;
  output.vesting = vestingShell;

  return output;
};
