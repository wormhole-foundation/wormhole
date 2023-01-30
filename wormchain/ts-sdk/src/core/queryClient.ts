import { Api as authApi } from "../modules/cosmos.auth.v1beta1/rest";
import { Api as bankApi } from "../modules/cosmos.bank.v1beta1/rest";
import { Api as baseApi } from "../modules/cosmos.base.tendermint.v1beta1/rest";
import { Api as crisisApi } from "../modules/cosmos.crisis.v1beta1/rest";
import { Api as distributionApi } from "../modules/cosmos.distribution.v1beta1/rest";
import { Api as evidenceApi } from "../modules/cosmos.evidence.v1beta1/rest";
import { Api as feegrantApi } from "../modules/cosmos.feegrant.v1beta1/rest";
import { Api as govApi } from "../modules/cosmos.gov.v1beta1/rest";
import { Api as mintApi } from "../modules/cosmos.mint.v1beta1/rest";
import { Api as paramsApi } from "../modules/cosmos.params.v1beta1/rest";
import { Api as slashingApi } from "../modules/cosmos.slashing.v1beta1/rest";
import { Api as stakingApi } from "../modules/cosmos.staking.v1beta1/rest";
import { Api as txApi } from "../modules/cosmos.tx.v1beta1/rest";
import { Api as upgradeApi } from "../modules/cosmos.upgrade.v1beta1/rest";
import { Api as vestingApi } from "../modules/cosmos.vesting.v1beta1/rest";
import { Api as wasmApi } from "../modules/cosmwasm.wasm.v1/rest";
import { Api as coreApi } from "../modules/wormhole_foundation.wormchain.wormhole/rest";

export type WormchainQueryClient = {
  core: coreApi<any>;
  auth: authApi<any>;
  bank: bankApi<any>;
  base: baseApi<any>;
  crisis: crisisApi<any>;
  distribution: distributionApi<any>;
  evidence: evidenceApi<any>;
  feegrant: feegrantApi<any>;
  gov: govApi<any>;
  mint: mintApi<any>;
  params: paramsApi<any>;
  slashing: slashingApi<any>;
  staking: stakingApi<any>;
  tx: txApi<any>;
  upgrade: upgradeApi<any>;
  vesting: vestingApi<any>;
  wasm: wasmApi<any>;
};

export function getWormholeQueryClient(
  lcdAddress: string,
  nodejs?: boolean
): WormchainQueryClient {
  if (nodejs) {
    var fetch = require("node-fetch");
    //@ts-ignore
    globalThis.fetch = fetch;
  }
  const core = new coreApi({ baseUrl: lcdAddress });
  const auth = new authApi({ baseUrl: lcdAddress });
  const bank = new bankApi({ baseUrl: lcdAddress });
  const base = new baseApi({ baseUrl: lcdAddress });
  const crisis = new crisisApi({ baseUrl: lcdAddress });
  const distribution = new distributionApi({ baseUrl: lcdAddress });
  const evidence = new evidenceApi({ baseUrl: lcdAddress });
  const feegrant = new feegrantApi({ baseUrl: lcdAddress });
  const gov = new govApi({ baseUrl: lcdAddress });
  const mint = new mintApi({ baseUrl: lcdAddress });
  const params = new paramsApi({ baseUrl: lcdAddress });
  const slashing = new slashingApi({ baseUrl: lcdAddress });
  const staking = new stakingApi({ baseUrl: lcdAddress });
  const tx = new txApi({ baseUrl: lcdAddress });
  const upgrade = new upgradeApi({ baseUrl: lcdAddress });
  const vesting = new vestingApi({ baseUrl: lcdAddress });
  const wasm = new wasmApi({ baseUrl: lcdAddress });

  return {
    core,
    auth,
    bank,
    base,
    crisis,
    distribution,
    evidence,
    feegrant,
    gov,
    mint,
    params,
    slashing,
    staking,
    tx,
    upgrade,
    vesting,
    wasm,
  };
}

// export async function getStargateQueryClient(tendermintAddress: string) {
//   const tmClient = await Tendermint34Client.connect(tendermintAddress);
//   const client = QueryClient.withExtensions(
//     tmClient,
//     setupTxExtension,
//     setupGovExtension,
//     setupIbcExtension,
//     setupAuthExtension,
//     setupBankExtension,
//     setupMintExtension,
//     setupStakingExtension
//   );

//   return client;
// }
