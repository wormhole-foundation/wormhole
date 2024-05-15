import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { calculateFee } from "@cosmjs/stargate";
import { MsgExecuteContractEncodeObject } from "@cosmjs/cosmwasm-stargate";
import { toUtf8 } from "@cosmjs/encoding";
import { MsgExecuteContract } from "cosmjs-types/cosmwasm/wasm/v1/tx";
import { getSigningCosmWasmClient } from "@sei-js/core";

import { NETWORKS } from "../../consts";
import { impossible, Payload } from "../../vaa";
import { contracts, Network } from "@wormhole-foundation/sdk";

export const submit = async (
  payload: Payload,
  vaa: Buffer,
  network: Network,
  rpc?: string
) => {
  // const contracts = CONTRACTS[network].Sei;
  const networkInfo = NETWORKS[network].Sei;
  rpc = rpc || networkInfo.rpc;
  const key = networkInfo.key;
  if (!key) {
    throw Error(`No ${network} key defined for Sei`);
  }

  if (!rpc) {
    throw Error(`No ${network} rpc defined for Sei`);
  }

  let target_contract: string;
  let execute_msg: object;
  switch (payload.module) {
    case "Core": {
      const core = contracts.coreBridge.get(network, "Sei");
      if (!core) {
        throw new Error(`Core bridge address not defined for Sei ${network}`);
      }

      target_contract = core;
      // sigh...
      execute_msg = {
        submit_v_a_a: {
          vaa: vaa.toString("base64"),
        },
      };
      switch (payload.type) {
        case "GuardianSetUpgrade":
          console.log("Submitting new guardian set");
          break;
        case "ContractUpgrade":
          console.log("Upgrading core contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on sei");
        default:
          impossible(payload);
      }

      break;
    }
    case "NFTBridge": {
      const nft = contracts.nftBridge.get(network, "Sei");
      if (!nft) {
        // NOTE: this code can safely be removed once the sei NFT bridge is
        // released, but it's fine for it to stay, as the condition will just be
        // skipped once 'contracts.nft_bridge' is defined
        throw new Error("NFT bridge not supported yet for Sei");
      }

      target_contract = nft;
      execute_msg = {
        submit_vaa: {
          data: vaa.toString("base64"),
        },
      };
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on sei");
        case "RegisterChain":
          console.log("Registering chain");
          break;
        case "Transfer":
          console.log("Completing transfer");
          break;
        default:
          impossible(payload);
      }

      break;
    }
    case "TokenBridge": {
      const tb = contracts.tokenBridge.get(network, "Sei");
      if (!tb) {
        throw new Error(`Token bridge address not defined for Sei ${network}`);
      }

      target_contract = tb;
      execute_msg = {
        submit_vaa: {
          data: vaa.toString("base64"),
        },
      };
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on sei");
        case "RegisterChain":
          console.log("Registering chain");
          break;
        case "Transfer":
          console.log("Completing transfer");
          break;
        case "AttestMeta":
          console.log("Creating wrapped token");
          break;
        case "TransferWithPayload":
          throw Error("Can't complete payload 3 transfer from CLI");
        default:
          impossible(payload);
          break;
      }

      break;
    }
    case "WormholeRelayer":
      throw Error("Wormhole Relayer not supported on Sei");
    default:
      target_contract = impossible(payload);
      execute_msg = impossible(payload);
  }

  const wallet = await DirectSecp256k1HdWallet.fromMnemonic(key, {
    prefix: "sei",
  });
  const [account] = await wallet.getAccounts();
  const client = await getSigningCosmWasmClient(rpc, wallet);

  const executeContractMsg: MsgExecuteContractEncodeObject = {
    typeUrl: "/cosmwasm.wasm.v1.MsgExecuteContract",
    value: MsgExecuteContract.fromPartial({
      sender: account.address,
      contract: target_contract,
      msg: toUtf8(JSON.stringify(execute_msg)),
      funds: [],
    }),
  };
  // For some reason, the simulation only provides the gas used but no events
  // so we can't determine whether it worked unless we do away with cosmjs and query the node ourselves.
  // See https://github.com/cosmos/cosmjs/issues/1148#issuecomment-1129259646
  const gasUsed = await client.simulate(
    account.address,
    [executeContractMsg],
    undefined
  );
  // It looks like the simulation is a bit lacking when it comes to estimating gas.
  // See https://github.com/cosmos/cosmos-sdk/issues/4938
  // That's why we multiply it by a factor of 1.3
  const estimatedGas = Math.floor((gasUsed * 130) / 100);

  const fee = calculateFee(estimatedGas, "0.1usei");
  const result = await client.execute(
    account.address,
    target_contract,
    execute_msg,
    fee
  );

  console.log(`TX hash: ${result.transactionHash}`);
};
