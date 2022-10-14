import { Connection, PublicKey } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { ChainGrpcWasmApi } from "@injectivelabs/sdk-ts";
import { Algodv2 } from "algosdk";
import { AptosClient } from "aptos";
import { ethers } from "ethers";
import { fromUint8Array } from "js-base64";
import { Account as nearAccount } from "near-api-js";
import { calcLogicSigAccount, decodeLocalState, hexToNativeAssetBigIntAlgorand } from "../algorand";
import { Bridge__factory } from "../ethers-contracts";
import { importTokenWasm } from "../solana/wasm";
import {
  callFunctionNear,
  ChainId,
  ChainName,
  CHAIN_ID_ALGORAND,
  coalesceChainId,
  getAssetFullyQualifiedType,
  getForeignAssetAddress,
} from "../utils";
import { Provider } from "near-api-js/lib/providers";
import { LCDClient as XplaLCDClient } from "@xpla/xpla.js";

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
    return await tokenBridge.wrappedAsset(coalesceChainId(originChain), originAsset);
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
    const result: { address: string } = await client.wasm.contractQuery(tokenBridgeAddress, {
      wrapped_registry: {
        chain: coalesceChainId(originChain),
        address: fromUint8Array(originAsset),
      },
    });
    return result.address;
  } catch (e) {
    return null;
  }
}

/**
 * Returns the address of the foreign asset
 * @param tokenBridgeAddress Address of token bridge contact
 * @param client Holds the wallet and signing information
 * @param originChain The chainId of the origin of the asset
 * @param originAsset The address of the origin asset
 * @returns The foreign asset address or null
 */
export async function getForeignAssetInjective(
  tokenBridgeAddress: string,
  client: ChainGrpcWasmApi,
  originChain: ChainId | ChainName,
  originAsset: Uint8Array
): Promise<string | null> {
  try {
    const queryResult = await client.fetchSmartContractState(
      tokenBridgeAddress,
      Buffer.from(
        JSON.stringify({
          wrapped_registry: {
            chain: coalesceChainId(originChain),
            address: fromUint8Array(originAsset),
          },
        })
      ).toString("base64")
    );
    let result: any = null;
    if (typeof queryResult.data === "string") {
      result = JSON.parse(
        Buffer.from(queryResult.data, "base64").toString("utf-8")
      );
    }
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
 * @returns
 */
export async function getForeignAssetSolana(
  connection: Connection,
  tokenBridgeAddress: string,
  originChain: ChainId | ChainName,
  originAsset: Uint8Array
): Promise<string | null> {
  const { wrapped_address } = await importTokenWasm();
  const wrappedAddress = wrapped_address(
    tokenBridgeAddress,
    originAsset,
    coalesceChainId(originChain)
  );
  const wrappedAddressPK = new PublicKey(wrappedAddress);
  const wrappedAssetAccountInfo = await connection.getAccountInfo(wrappedAddressPK);
  return wrappedAssetAccountInfo ? wrappedAddressPK.toString() : null;
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
    let asset: Uint8Array = await decodeLocalState(client, tokenBridgeId, lsa.address());
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

export async function getForeignAssetAptos(
  client: AptosClient,
  tokenBridgeAddress: string,
  originChain: ChainId | ChainName,
  originAddress: string
): Promise<string | null> {
  const originChainId = coalesceChainId(originChain);
  const assetAddress = getForeignAssetAddress(tokenBridgeAddress, originChainId, originAddress);
  if (!assetAddress) {
    return null;
  }

  try {
    // check if asset exists and throw if it doesn't
    await client.getAccountResource(assetAddress, `0x1::coin::CoinInfo<${assetAddress}::coin::T>`);
    return assetAddress;
  } catch (e) {
    return null;
  }
}
