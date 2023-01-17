import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { TransactionResponse } from "@solana/web3.js";
import { ethers } from "ethers";
import {
  getEmitterAddressEth,
  getEmitterAddressSolana,
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
} from "../../../bridge";
import { getSignedVAAWithRetry } from "../../../rpc";
import { CHAIN_ID_ETH, CHAIN_ID_SOLANA, CONTRACTS } from "../../../utils";
import { WORMHOLE_RPC_HOSTS } from "../consts";

export async function waitUntilTransactionObservedEthereum(
  receipt: ethers.ContractReceipt
): Promise<Uint8Array> {
  // get the sequence from the logs (needed to fetch the vaa)
  let sequence = parseSequenceFromLogEth(
    receipt,
    CONTRACTS.DEVNET.ethereum.core
  );
  let emitterAddress = getEmitterAddressEth(
    CONTRACTS.DEVNET.ethereum.nft_bridge
  );
  // poll until the guardian(s) witness and sign the vaa
  const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
    WORMHOLE_RPC_HOSTS,
    CHAIN_ID_ETH,
    emitterAddress,
    sequence,
    {
      transport: NodeHttpTransport(),
    }
  );
  return signedVAA;
}

export async function waitUntilTransactionObservedSolana(
  response: TransactionResponse
): Promise<Uint8Array> {
  // get the sequence from the logs (needed to fetch the vaa)
  const sequence = parseSequenceFromLogSolana(response);
  const emitterAddress = await getEmitterAddressSolana(
    CONTRACTS.DEVNET.solana.nft_bridge
  );
  // poll until the guardian(s) witness and sign the vaa
  const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
    WORMHOLE_RPC_HOSTS,
    CHAIN_ID_SOLANA,
    emitterAddress,
    sequence,
    {
      transport: NodeHttpTransport(),
    }
  );
  return signedVAA;
}
