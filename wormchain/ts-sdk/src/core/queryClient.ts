import { Api as authApi } from "../modules/cosmos.auth.v1beta1/rest";
import { Api as bankApi } from "../modules/cosmos.bank.v1beta1/rest";
import { Api as baseApi } from "../modules/cosmos.base.tendermint.v1beta1/rest";
import { Api as crisisApi } from "../modules/cosmos.crisis.v1beta1/rest";
import { Api as distributionApi } from "../modules/cosmos.distribution.v1beta1/rest";
import { Api as evidenceApi } from "../modules/cosmos.evidence.v1beta1/rest";
import { Api as govApi } from "../modules/cosmos.gov.v1beta1/rest";
import { Api as mintApi } from "../modules/cosmos.mint.v1beta1/rest";
import { Api as paramsApi } from "../modules/cosmos.params.v1beta1/rest";
import { Api as slashingApi } from "../modules/cosmos.slashing.v1beta1/rest";
import { Api as stakingApi } from "../modules/cosmos.staking.v1beta1/rest";
import { Api as txApi } from "../modules/cosmos.tx.v1beta1/rest";
import { Api as upgradeApi } from "../modules/cosmos.upgrade.v1beta1/rest";
import { Api as wasmApi } from "../modules/cosmwasm.wasm.v1/rest";
import { Api as coreApi } from "../modules/wormchain.wormhole/rest";


export type WormchainQueryClient = {
  core: coreApi<any>;
  auth: authApi<any>;
  bank: bankApi<any>;
  base: baseApi<any>;
  crisis: crisisApi<any>;
  distribution: distributionApi<any>;
  evidence: evidenceApi<any>;
  gov: govApi<any>;
  mint: mintApi<any>;
  params: paramsApi<any>;
  slashing: slashingApi<any>;
  staking: stakingApi<any>;
  tx: txApi<any>;
  upgrade: upgradeApi<any>;
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

  const core = new coreApi({ baseURL: lcdAddress });
  const auth = new authApi({ baseURL: lcdAddress });
  const bank = new bankApi({ baseURL: lcdAddress });
  const base = new baseApi({ baseURL: lcdAddress });
  const crisis = new crisisApi({ baseURL: lcdAddress });
  const distribution = new distributionApi({ baseURL: lcdAddress });
  const evidence = new evidenceApi({ baseURL: lcdAddress });
  const gov = new govApi({ baseURL: lcdAddress });
  const mint = new mintApi({ baseURL: lcdAddress });
  const params = new paramsApi({ baseURL: lcdAddress });
  const slashing = new slashingApi({ baseURL: lcdAddress });
  const staking = new stakingApi({ baseURL: lcdAddress });
  const tx = new txApi({ baseURL: lcdAddress });
  const upgrade = new upgradeApi({ baseURL: lcdAddress });
  const wasm = new wasmApi({ baseURL: lcdAddress });

  return {
    core,
    auth,
    bank,
    base,
    crisis,
    distribution,
    evidence,
    gov,
    mint,
    params,
    slashing,
    staking,
    tx,
    upgrade,
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
