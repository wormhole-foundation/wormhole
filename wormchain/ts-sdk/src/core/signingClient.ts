import { OfflineSigner, Registry } from "@cosmjs/proto-signing";
import {
  SigningStargateClient,
  SigningStargateClientOptions,
} from "@cosmjs/stargate";
import * as authModule from "../modules/cosmos.auth.v1beta1";
import * as bankModule from "../modules/cosmos.bank.v1beta1";
import * as baseModule from "../modules/cosmos.base.tendermint.v1beta1";
import * as crisisModule from "../modules/cosmos.crisis.v1beta1";
import * as distributionModule from "../modules/cosmos.distribution.v1beta1";
import * as evidenceModule from "../modules/cosmos.evidence.v1beta1";
import * as feegrantModule from "../modules/cosmos.feegrant.v1beta1";
import * as govModule from "../modules/cosmos.gov.v1beta1";
import * as mintModule from "../modules/cosmos.mint.v1beta1";
import * as paramsModule from "../modules/cosmos.params.v1beta1";
import * as slashingModule from "../modules/cosmos.slashing.v1beta1";
import * as stakingModule from "../modules/cosmos.staking.v1beta1";
import * as txModule from "../modules/cosmos.tx.v1beta1";
import * as upgradeModule from "../modules/cosmos.upgrade.v1beta1";
import * as vestingModule from "../modules/cosmos.vesting.v1beta1";
import * as wasmModule from "../modules/cosmwasm.wasm.v1";
import * as coreModule from "../modules/wormhole_foundation.wormchain.wormhole";

//protobuf isn't guaranteed to have long support, which is used by the stargate signing client,
//so we're going to use an independent long module and shove it into the globals of protobuf
var Long = require("long");
var protobuf = require("protobufjs");
protobuf.util.Long = Long;
protobuf.configure();

//Rip the types out of their modules. These are private fields on the module.
//@ts-ignore
const coreTypes = coreModule.registry.types;
//@ts-ignore
const authTypes = authModule.registry.types;
//@ts-ignore
const bankTypes = bankModule.registry.types;
//@ts-ignore
const baseTypes = baseModule.registry.types;
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
const mintTypes = mintModule.registry.types;
//@ts-ignore
const paramsTypes = paramsModule.registry.types;
//@ts-ignore
const slashingTypes = slashingModule.registry.types;
//@ts-ignore
const stakingTypes = stakingModule.registry.types;
//@ts-ignore
const txTypes = txModule.registry.types;
//@ts-ignore
const upgradeTypes = upgradeModule.registry.types;
//@ts-ignore
const vestingTypes = vestingModule.registry.types;
//@ts-ignore
const wasmTypes = wasmModule.registry.types;

const aggregateTypes = [
  ...coreTypes,
  ...authTypes,
  ...bankTypes,
  ...baseTypes,
  ...crisisTypes,
  ...distributionTypes,
  ...evidenceTypes,
  ...feegrantTypes,
  ...govTypes,
  ...mintTypes,
  ...paramsTypes,
  ...slashingTypes,
  ...stakingTypes,
  ...txTypes,
  ...upgradeTypes,
  ...vestingTypes,
  ...wasmTypes,
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

  const authClient = await authModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const bankClient = await bankModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const baseClient = await baseModule.txClient(wallet, {
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

  const mintClient = await mintModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const paramsClient = await paramsModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const slashingClient = await slashingModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const stakingClient = await stakingModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const txClient = await txModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const upgradeClient = await upgradeModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const vestingClient = await vestingModule.txClient(wallet, {
    addr: tendermintAddress,
  });

  const wasmClient = await wasmModule.txClient(wallet, {
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

  const authShell = {
    ...authClient,
    signAndBroadcast: undefined,
  };
  delete authShell.signAndBroadcast;

  const bankShell = {
    ...bankClient,
    signAndBroadcast: undefined,
  };
  delete bankShell.signAndBroadcast;

  const baseShell = {
    ...baseClient,
    signAndBroadcast: undefined,
  };
  delete baseShell.signAndBroadcast;

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

  const mintShell = {
    ...mintClient,
    signAndBroadcast: undefined,
  };
  delete mintShell.signAndBroadcast;

  const paramsShell = {
    ...paramsClient,
    signAndBroadcast: undefined,
  };
  delete paramsShell.signAndBroadcast;

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

  const txShell = {
    ...txClient,
    signAndBroadcast: undefined,
  };
  delete txShell.signAndBroadcast;

  const upgradeShell = {
    ...upgradeClient,
    signAndBroadcast: undefined,
  };
  delete upgradeShell.signAndBroadcast;

  const vestingShell = {
    ...vestingClient,
    signAndBroadcast: undefined,
  };
  delete vestingShell.signAndBroadcast;

  const wasmShell = {
    ...wasmClient,
    signAndBroadcast: undefined,
  };
  delete wasmShell.signAndBroadcast;

  type CoreType = Omit<typeof coreShell, "signAndBroadcast">;
  type AuthzType = Omit<typeof authShell, "signAndBroadcast">;
  type BankType = Omit<typeof bankShell, "signAndBroadcast">;
  type BaseType = Omit<typeof baseShell, "signAndBroadcast">;
  type CrisisType = Omit<typeof crisisShell, "signAndBroadcast">;
  type DistributionType = Omit<typeof distributionShell, "signAndBroadcast">;
  type EvidenceType = Omit<typeof evidenceShell, "signAndBroadcast">;
  type FeegrantType = Omit<typeof feegrantShell, "signAndBroadcast">;
  type GovType = Omit<typeof govShell, "signAndBroadcast">;
  type MintType = Omit<typeof mintShell, "signAndBroadcast">;
  type ParamsType = Omit<typeof paramsShell, "signAndBroadcast">;
  type SlashingType = Omit<typeof slashingShell, "signAndBroadcast">;
  type StakingType = Omit<typeof stakingShell, "signAndBroadcast">;
  type TxType = Omit<typeof txShell, "signAndBroadcast">;
  type UpgradeType = Omit<typeof upgradeShell, "signAndBroadcast">;
  type VestingType = Omit<typeof vestingShell, "signAndBroadcast">;
  type WasmType = Omit<typeof wasmShell, "signAndBroadcast">;
  type WormholeFunctions = {
    core: CoreType;
    auth: AuthzType;
    bank: BankType;
    base: BaseType;
    crisis: CrisisType;
    distribution: DistributionType;
    evidence: EvidenceType;
    feegrant: FeegrantType;
    gov: GovType;
    mint: MintType;
    params: ParamsType;
    slashing: SlashingType;
    staking: StakingType;
    tx: TxType;
    upgrade: UpgradeType;
    vesting: VestingType;
    wasm: WasmType;
  };
  type WormholeSigningClient = SigningStargateClient & WormholeFunctions;

  const output: WormholeSigningClient = client as WormholeSigningClient;

  output.core = coreShell;
  output.auth = authShell;
  output.bank = bankShell;
  output.base = baseShell;
  output.crisis = crisisShell;
  output.distribution = distributionShell;
  output.evidence = evidenceShell;
  output.feegrant = feegrantShell;
  output.gov = govShell;
  output.mint = mintShell;
  output.params = paramsShell;
  output.slashing = slashingShell;
  output.staking = stakingShell;
  output.tx = txShell;
  output.upgrade = upgradeShell;
  output.vesting = vestingShell;
  output.wasm = wasmShell;

  return output;
};
