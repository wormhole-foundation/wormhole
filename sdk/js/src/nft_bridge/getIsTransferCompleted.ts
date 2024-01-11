import { Commitment, Connection, PublicKeyInitData } from "@solana/web3.js";
import { AptosClient } from "aptos";
import { ethers } from "ethers";
import { ensureHexPrefix } from "..";
import { NftBridgeState } from "../aptos/types";
import { getSignedVAAHash } from "../bridge";
import { NFTBridge__factory } from "../ethers-contracts";
import { getClaim } from "../solana/wormhole";
import { parseVaa, SignedVaa } from "../vaa/wormhole";

export async function getIsTransferCompletedEth(
  nftBridgeAddress: string,
  provider: ethers.Signer | ethers.providers.Provider,
  signedVAA: Uint8Array
) {
  const nftBridge = NFTBridge__factory.connect(nftBridgeAddress, provider);
  const signedVAAHash = getSignedVAAHash(signedVAA);
  return await nftBridge.isTransferCompleted(signedVAAHash);
}

export async function getIsTransferCompletedSolana(
  nftBridgeAddress: PublicKeyInitData,
  signedVAA: SignedVaa,
  connection: Connection,
  commitment?: Commitment
) {
  const parsed = parseVaa(signedVAA);
  return getClaim(
    connection,
    nftBridgeAddress,
    parsed.emitterAddress,
    parsed.emitterChain,
    parsed.sequence,
    commitment
  ).catch((e) => false);
}

export async function getIsTransferCompletedAptos(
  client: AptosClient,
  nftBridgeAddress: string,
  transferVaa: Uint8Array
): Promise<boolean> {
  // get handle
  nftBridgeAddress = ensureHexPrefix(nftBridgeAddress);
  const state = (
    await client.getAccountResource(
      nftBridgeAddress,
      `${nftBridgeAddress}::state::State`
    )
  ).data as NftBridgeState;
  const handle = state.consumed_vaas.elems.handle;

  // check if vaa hash is in consumed_vaas
  const transferVaaHash = getSignedVAAHash(transferVaa);
  try {
    // when accessing Set<T>, key is type T and value is 0
    await client.getTableItem(handle, {
      key_type: "vector<u8>",
      value_type: "u8",
      key: transferVaaHash,
    });
    return true;
  } catch {
    return false;
  }
}
