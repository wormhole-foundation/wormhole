import { GetSignedVAAResponse } from "@certusone/wormhole-sdk-proto-web/lib/cjs/publicrpc/v1/publicrpc";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { TransactionResponse } from "@solana/web3.js";
import { AptosClient, Types } from "aptos";
import { ethers } from "ethers";
import { NftBridgeState } from "../../../aptos/types";
import {
  getEmitterAddressEth,
  getEmitterAddressSolana,
  parseSequenceFromLogAptos,
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
} from "../../../bridge";
import { getSignedVAAWithRetry } from "../../../rpc";
import {
  ChainId,
  CHAIN_ID_APTOS,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CONTRACTS,
} from "../../../utils";
import { WORMHOLE_RPC_HOSTS } from "./consts";

// TODO(aki): implement getEmitterAddressAptos and sub here
export async function getSignedVaaAptos(
  client: AptosClient,
  result: Types.UserTransaction
): Promise<Uint8Array> {
  // get the sequence from the logs (needed to fetch the vaa)
  const sequence = parseSequenceFromLogAptos(
    CONTRACTS.DEVNET.aptos.core,
    result
  );
  if (sequence === null) {
    throw new Error("aptos: Could not parse sequence from logs");
  }

  const nftBridgeAddress = CONTRACTS.DEVNET.aptos.nft_bridge;
  const state = (await client.getAccountResources(nftBridgeAddress)).find(
    (r) => r.type === `${nftBridgeAddress}::state::State`
  );
  if (!state) {
    throw new Error("aptos: Could not find State resource");
  }

  // TODO: ensure 0x is stripped if exists
  const emitterAddress = (
    state.data as NftBridgeState
  ).emitter_cap.emitter.padStart(64, "0");

  // poll until the guardian(s) witness and sign the vaa
  return getSignedVaa(CHAIN_ID_APTOS, emitterAddress, sequence);
}

export async function getSignedVaaEthereum(
  receipt: ethers.ContractReceipt
): Promise<Uint8Array> {
  // get the sequence from the logs (needed to fetch the vaa)
  const sequence = parseSequenceFromLogEth(
    receipt,
    CONTRACTS.DEVNET.ethereum.core
  );
  const emitterAddress = getEmitterAddressEth(
    CONTRACTS.DEVNET.ethereum.nft_bridge
  );

  // poll until the guardian(s) witness and sign the vaa
  return getSignedVaa(CHAIN_ID_ETH, emitterAddress, sequence);
}

export async function getSignedVaaSolana(
  response: TransactionResponse
): Promise<Uint8Array> {
  // get the sequence from the logs (needed to fetch the vaa)
  const sequence = parseSequenceFromLogSolana(response);
  const emitterAddress = getEmitterAddressSolana(
    CONTRACTS.DEVNET.solana.nft_bridge
  );

  // poll until the guardian(s) witness and sign the vaa
  return getSignedVaa(CHAIN_ID_SOLANA, emitterAddress, sequence);
}

const getSignedVaa = async (
  chain: ChainId,
  emitterAddress: string,
  sequence: string
): Promise<Uint8Array> => {
  const { vaaBytes: signedVAA }: GetSignedVAAResponse =
    await getSignedVAAWithRetry(
      WORMHOLE_RPC_HOSTS,
      chain,
      emitterAddress,
      sequence,
      {
        transport: NodeHttpTransport(),
      }
    );
  return signedVAA;
};
