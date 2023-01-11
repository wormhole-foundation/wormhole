import { Connection, PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { BN, Program, Provider } from "@project-serum/anchor";
import { createReadOnlyProvider } from "../utils";
import { NftBridgeCoder } from "./coder";
import { NftBridge } from "../types/nftBridge";

import IDL from "../../anchor-idl/nft_bridge.json";

export const NFT_TRANSFER_NATIVE_TOKEN_ADDRESS = Buffer.alloc(32, 1);

export function createNftBridgeProgramInterface(
  programId: PublicKeyInitData,
  provider?: Provider
): Program<NftBridge> {
  return new Program<NftBridge>(
    IDL as NftBridge,
    new PublicKey(programId),
    provider === undefined ? ({ connection: null } as any) : provider,
    coder()
  );
}

export function createReadOnlyNftBridgeProgramInterface(
  programId: PublicKeyInitData,
  connection?: Connection
): Program<NftBridge> {
  return createNftBridgeProgramInterface(
    programId,
    createReadOnlyProvider(connection)
  );
}

export function coder(): NftBridgeCoder {
  return new NftBridgeCoder(IDL as NftBridge);
}

export function tokenIdToMint(tokenId: bigint) {
  return new PublicKey(new BN(tokenId.toString()).toArrayLike(Buffer));
}

export function mintToTokenId(mint: PublicKeyInitData) {
  return BigInt(new BN(new PublicKey(mint).toBuffer()).toString());
}
