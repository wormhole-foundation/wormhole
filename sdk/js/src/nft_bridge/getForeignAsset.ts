import { BN } from "@project-serum/anchor";
import { PublicKeyInitData } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { AptosClient, HexString, TokenTypes, Types } from "aptos";
import { ethers } from "ethers";
import { isBytes } from "ethers/lib/utils";
import { fromUint8Array } from "js-base64";
import { CHAIN_ID_SOLANA } from "..";
import { CreateTokenDataEvent, NftBridgeState } from "../aptos/types";
import { NFTBridge__factory } from "../ethers-contracts";
import { deriveWrappedMintKey } from "../solana/nftBridge";
import {
  ChainId,
  ChainName,
  CHAIN_ID_APTOS,
  coalesceChainId,
  deriveResourceAccountAddress,
  ensureHexPrefix,
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
 * Returns a foreign asset address on Terra for a provided native chain and asset address
 * @param nftBridgeAddress
 * @param client
 * @param originChain
 * @param originAsset
 * @returns
 */
export async function getForeignAssetTerra(
  nftBridgeAddress: string,
  client: LCDClient,
  originChain: ChainId,
  originAsset: Uint8Array
): Promise<string | null> {
  const originChainId = coalesceChainId(originChain);
  try {
    const address =
      originChain == CHAIN_ID_SOLANA
        ? "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="
        : fromUint8Array(originAsset);
    const result: { address: string } = await client.wasm.contractQuery(
      nftBridgeAddress,
      {
        wrapped_registry: {
          chain: originChainId,
          address,
        },
      }
    );
    return result.address;
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
    const state = (
      await client.getAccountResource(
        nftBridgeAddress,
        `${nftBridgeAddress}::state::State`
      )
    ).data as NftBridgeState;
    const handle = state.native_infos.handle;
    const { token_data_id, property_version } = (await client.getTableItem(
      handle,
      {
        key_type: `${nftBridgeAddress}::token_hash::TokenHash`,
        value_type: `0x3::token::TokenId`,
        key: {
          hash: HexString.fromUint8Array(originAddress).hex(),
        },
      }
    )) as TokenTypes.TokenId & { __headers: unknown };
    return { token_data_id, property_version };
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
  // token creation event with the token id that we're looking for.
  const PAGE_SIZE = 25;
  let curr = 0;
  let numEvents = PAGE_SIZE;
  let event: Types.Event | undefined = undefined;
  while (numEvents === PAGE_SIZE && !event) {
    const events = await client.getEventsByEventHandle(
      creatorAddress,
      "0x3::token::Collections",
      "create_token_data_events",
      { limit: PAGE_SIZE, start: curr }
    );
    event = events.find(
      (e) =>
        ensureHexPrefix((e as CreateTokenDataEvent).data.id.name) ===
        HexString.fromUint8Array(tokenIdAsUint8Array).hex()
    );
    numEvents = events.length;
    curr += numEvents;
  }

  if (!event) {
    throw new Error("Invalid token ID");
  }

  return {
    token_data_id: event.data.id,
    property_version: "0",
  };
}
