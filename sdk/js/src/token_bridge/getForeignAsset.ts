import { JsonRpcProvider } from "@mysten/sui.js";
import { Commitment, Connection, PublicKeyInitData } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { LCDClient as XplaLCDClient } from "@xpla/xpla.js";
import { Algodv2 } from "algosdk";
import { AptosClient } from "aptos";
import { ethers } from "ethers";
import { fromUint8Array } from "js-base64";
import { Provider } from "near-api-js/lib/providers";
import {
  calcLogicSigAccount,
  decodeLocalState,
  hexToNativeAssetBigIntAlgorand,
} from "../algorand";
import { Bridge__factory } from "../ethers-contracts";
import { deriveWrappedMintKey, getWrappedMeta } from "../solana/tokenBridge";
import { getTokenCoinType } from "../sui";
import {
  callFunctionNear,
  ChainId,
  ChainName,
  CHAIN_ID_ALGORAND,
  coalesceChainId,
  coalesceModuleAddress,
  getAssetFullyQualifiedType,
} from "../utils";

/**
 * Returns a foreign asset address on Ethereum for a provided native chain and asset address, AddressZero if it does not exist
 * @param tokenBridgeAddress
 * @param provider
 * @param originChain
 * @param originAsset zero pad to 32 bytes
 * @returns
 */
export async function getForeignAssetEth(
  tokenBridgeAddress: string,
  provider: ethers.Signer | ethers.providers.Provider,
  originChain: ChainId | ChainName,
  originAsset: Uint8Array
): Promise<string | null> {
  const tokenBridge = Bridge__factory.connect(tokenBridgeAddress, provider);
  try {
    return await tokenBridge.wrappedAsset(
      coalesceChainId(originChain),
      originAsset
    );
  } catch (e) {
    return null;
  }
}

export async function getForeignAssetTerra(
  tokenBridgeAddress: string,
  client: LCDClient,
  originChain: ChainId | ChainName,
  originAsset: Uint8Array
): Promise<string | null> {
  try {
    const result: { address: string } = await client.wasm.contractQuery(
      tokenBridgeAddress,
      {
        wrapped_registry: {
          chain: coalesceChainId(originChain),
          address: fromUint8Array(originAsset),
        },
      }
    );
    return result.address;
  } catch (e) {
    return null;
  }
}

export async function getForeignAssetXpla(
  tokenBridgeAddress: string,
  client: XplaLCDClient,
  originChain: ChainId | ChainName,
  originAsset: Uint8Array
): Promise<string | null> {
  try {
    const result: { address: string } = await client.wasm.contractQuery(
      tokenBridgeAddress,
      {
        wrapped_registry: {
          chain: coalesceChainId(originChain),
          address: fromUint8Array(originAsset),
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
 * @param connection
 * @param tokenBridgeAddress
 * @param originChain
 * @param originAsset zero pad to 32 bytes
 * @param [commitment]
 * @returns
 */
export async function getForeignAssetSolana(
  connection: Connection,
  tokenBridgeAddress: PublicKeyInitData,
  originChain: ChainId | ChainName,
  originAsset: Uint8Array,
  commitment?: Commitment
): Promise<string | null> {
  const mint = deriveWrappedMintKey(
    tokenBridgeAddress,
    coalesceChainId(originChain) as number,
    originAsset
  );
  return getWrappedMeta(connection, tokenBridgeAddress, mint, commitment)
    .catch((_) => null)
    .then((meta) => (meta === null ? null : mint.toString()));
}

export async function getForeignAssetAlgorand(
  client: Algodv2,
  tokenBridgeId: bigint,
  chain: ChainId | ChainName,
  contract: string
): Promise<bigint | null> {
  const chainId = coalesceChainId(chain);
  if (chainId === CHAIN_ID_ALGORAND) {
    return hexToNativeAssetBigIntAlgorand(contract);
  } else {
    let { lsa, doesExist } = await calcLogicSigAccount(
      client,
      tokenBridgeId,
      BigInt(chainId),
      contract
    );
    if (!doesExist) {
      return null;
    }
    let asset: Uint8Array = await decodeLocalState(
      client,
      tokenBridgeId,
      lsa.address()
    );
    if (asset.length > 8) {
      const tmp = Buffer.from(asset.slice(0, 8));
      return tmp.readBigUInt64BE(0);
    } else return null;
  }
}

export async function getForeignAssetNear(
  provider: Provider,
  tokenAccount: string,
  chain: ChainId | ChainName,
  contract: string
): Promise<string | null> {
  const ret = await callFunctionNear(
    provider,
    tokenAccount,
    "get_foreign_asset",
    {
      chain: coalesceChainId(chain),
      address: contract,
    }
  );
  return ret !== "" ? ret : null;
}

/**
 * Get qualified type of asset on Aptos given its origin info.
 * @param client Client used to transfer data to/from Aptos node
 * @param tokenBridgeAddress Address of token bridge
 * @param originChain Chain ID of chain asset is originally from
 * @param originAddress Asset address on origin chain
 * @returns Fully qualified type of asset on Aptos
 */
export async function getForeignAssetAptos(
  client: AptosClient,
  tokenBridgeAddress: string,
  originChain: ChainId | ChainName,
  originAddress: string
): Promise<string | null> {
  const originChainId = coalesceChainId(originChain);
  const assetFullyQualifiedType = getAssetFullyQualifiedType(
    tokenBridgeAddress,
    originChainId,
    originAddress
  );
  if (!assetFullyQualifiedType) {
    return null;
  }

  try {
    // check if asset exists and throw if it doesn't
    await client.getAccountResource(
      coalesceModuleAddress(assetFullyQualifiedType),
      `0x1::coin::CoinInfo<${assetFullyQualifiedType}>`
    );
    return assetFullyQualifiedType;
  } catch (e) {
    return null;
  }
}

export async function getForeignAssetSui(
  provider: JsonRpcProvider,
  tokenBridgeStateObjectId: string,
  originChain: ChainId | ChainName,
  originAddress: Uint8Array
): Promise<string | null> {
  const originChainId = coalesceChainId(originChain);
  return getTokenCoinType(
    provider,
    tokenBridgeStateObjectId,
    originAddress,
    originChainId
  );
}
