import { BN } from "@project-serum/anchor";
import { PublicKeyInitData } from "@solana/web3.js";
import {
  ApiError,
  AptosClient,
  HexString,
  TokenClient,
  TokenTypes,
} from "aptos";
import { ethers } from "ethers";
import { isBytes } from "ethers/lib/utils";
import { CHAIN_ID_SOLANA } from "..";
import { CreateTokenDataEvent } from "../aptos/types";
import { NFTBridge__factory } from "../ethers-contracts";
import { deriveWrappedMintKey } from "../solana/nftBridge";
import {
  ChainId,
  ChainName,
  CHAIN_ID_APTOS,
  coalesceChainId,
  deriveResourceAccountAddress,
  ensureHexPrefix,
  getTokenIdFromTokenHash,
  hexToUint8Array,
} from "../utils";

/**
 * Returns a foreign asset address on Ethereum for a provided native chain and asset address, AddressZero if it does not exist
 * @param nftBridgeAddress
 * @param provider
 * @param originChain
 * @param originAsset zero pad to 32 bytes
 * @returns
 */
export async function getForeignAssetEth(
  nftBridgeAddress: string,
  provider: ethers.Signer | ethers.providers.Provider,
  originChain: ChainId | ChainName,
  originAsset: Uint8Array
): Promise<string | null> {
  const originChainId = coalesceChainId(originChain);
  const tokenBridge = NFTBridge__factory.connect(nftBridgeAddress, provider);
  try {
    if (originChainId === CHAIN_ID_SOLANA) {
      // All NFTs from Solana are minted to the same address, the originAsset is encoded as the tokenId as
      // BigNumber.from(new PublicKey(originAsset).toBytes()).toString()
      const addr = await tokenBridge.wrappedAsset(
        originChain,
        "0x0101010101010101010101010101010101010101010101010101010101010101"
      );
      return addr;
    }
    return await tokenBridge.wrappedAsset(originChainId, originAsset);
  } catch (e) {
    return null;
  }
}

/**
 * Returns a foreign asset address on Solana for a provided native chain and asset address
 * @param nftBridgeAddress
 * @param originChain
 * @param originAsset zero pad to 32 bytes
 * @returns
 */
export async function getForeignAssetSolana(
  nftBridgeAddress: PublicKeyInitData,
  originChain: ChainId | ChainName,
  originAsset: string | Uint8Array | Buffer,
  tokenId: Uint8Array | Buffer | bigint
): Promise<string> {
  // we don't require NFT accounts to exist, so don't check them.
  return deriveWrappedMintKey(
    nftBridgeAddress,
    coalesceChainId(originChain) as number,
    originAsset,
    isBytes(tokenId) ? BigInt(new BN(tokenId).toString()) : tokenId
  ).toString();
}

export const getForeignAssetSol = getForeignAssetSolana;

/**
 * Get the token id of a foreign asset on Aptos. Tokens on Aptos are identified
 * by the tuple (creatorAddress, collectionName, tokenName, propertyVersion),
 * which this method returns as an object.
 *
 * This method also supports native assets, in which case it expects the token
 * hash (which can be obtained from `deriveTokenHashFromTokenId`).
 * @param client
 * @param nftBridgeAddress
 * @param originChain
 * @param originAddress External address of token on origin chain, or token hash
 * if origin chain is Aptos
 * @param tokenId Token id of token on origin chain, unnecessary if origin
 * chain is Aptos
 * @returns Unique token identifier on Aptos
 */
export async function getForeignAssetAptos(
  client: AptosClient,
  nftBridgeAddress: string,
  originChain: ChainId | ChainName,
  originAddress: Uint8Array,
  tokenId?: Uint8Array | Buffer | bigint
): Promise<TokenTypes.TokenId | null> {
  const originChainId = coalesceChainId(originChain);
  if (originChainId === CHAIN_ID_APTOS) {
    return getTokenIdFromTokenHash(client, nftBridgeAddress, originAddress);
  }

  const creatorAddress = await deriveResourceAccountAddress(
    nftBridgeAddress,
    originChainId,
    originAddress
  );
  if (!creatorAddress) {
    throw new Error("Could not derive creator account address");
  }

  if (typeof tokenId === "bigint") {
    tokenId = hexToUint8Array(BigInt(tokenId).toString(16).padStart(64, "0"));
  }

  if (!tokenId) {
    throw new Error("Invalid token ID");
  }

  const tokenIdAsUint8Array = new Uint8Array(tokenId);

  // Each creator account should contain a single collection that contains the
  // corresponding token creation events. Return if we find it in the first
  // page, otherwise reconstruct the token id from the first event.
  const PAGE_SIZE = 25;
  const events = (await client.getEventsByEventHandle(
    creatorAddress,
    "0x3::token::Collections",
    "create_token_data_events",
    { limit: PAGE_SIZE }
  )) as CreateTokenDataEvent[];
  const event = events.find(
    (e) =>
      ensureHexPrefix((e as CreateTokenDataEvent).data.id.name) ===
      HexString.fromUint8Array(tokenIdAsUint8Array).hex()
  );
  if (event) {
    return {
      token_data_id: event.data.id,
      property_version: "0", // property version always "0" for wrapped tokens
    };
  }

  // Skip pagination, reconstruct token id, and check to see if it exists
  try {
    const tokenIdObj = {
      token_data_id: {
        ...events[0].data.id,
        name: HexString.fromUint8Array(tokenIdAsUint8Array).noPrefix(),
      },
      property_version: "0",
    };
    await new TokenClient(client).getTokenData(
      tokenIdObj.token_data_id.creator,
      tokenIdObj.token_data_id.collection,
      tokenIdObj.token_data_id.name
    );
    return tokenIdObj;
  } catch (e) {
    if (
      e instanceof ApiError &&
      e.status === 404 &&
      e.errorCode === "table_item_not_found"
    ) {
      return null;
    }

    throw e;
  }
}
