import { ChainGrpcWasmApi } from "@injectivelabs/sdk-ts";
import { Connection, PublicKey } from "@solana/web3.js";
import { LCDClient as TerraLCDClient } from "@terra-money/terra.js";
import { Algodv2 } from "algosdk";
import { ethers } from "ethers";
import { arrayify, sha256, zeroPad } from "ethers/lib/utils";
import { decodeLocalState } from "../algorand";
import { buildTokenId, isNativeCosmWasmDenom } from "../cosmwasm/address";
import { TokenImplementation__factory } from "../ethers-contracts";
import { importTokenWasm } from "../solana/wasm";
import { buildNativeId } from "../terra";
import { canonicalAddress } from "../cosmos";
import {
  assertChain,
  ChainId,
  ChainName,
  CHAIN_ID_ALGORAND,
  CHAIN_ID_APTOS,
  CHAIN_ID_NEAR,
  CHAIN_ID_INJECTIVE,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  coalesceChainId,
  CosmWasmChainId,
  CosmWasmChainName,
  hexToUint8Array,
  coalesceCosmWasmChainId,
  tryHexToNativeAssetString,
  callFunctionNear,
} from "../utils";
import { safeBigIntToNumber } from "../utils/bigint";
import {
  getIsWrappedAssetAlgorand,
  getIsWrappedAssetEth,
  getIsWrappedAssetNear,
} from "./getIsWrappedAsset";
import { Provider } from "near-api-js/lib/providers";
import { LCDClient as XplaLCDClient } from "@xpla/xpla.js";
import { AptosClient } from "aptos";

// TODO: remove `as ChainId` and return number in next minor version as we can't ensure it will match our type definition
export interface WormholeWrappedInfo {
  isWrapped: boolean;
  chainId: ChainId;
  assetAddress: Uint8Array;
}

/**
 * Returns a origin chain and asset address on {originChain} for a provided Wormhole wrapped address
 * @param tokenBridgeAddress
 * @param provider
 * @param wrappedAddress
 * @returns
 */
export async function getOriginalAssetEth(
  tokenBridgeAddress: string,
  provider: ethers.Signer | ethers.providers.Provider,
  wrappedAddress: string,
  lookupChain: ChainId | ChainName
): Promise<WormholeWrappedInfo> {
  const isWrapped = await getIsWrappedAssetEth(
    tokenBridgeAddress,
    provider,
    wrappedAddress
  );
  if (isWrapped) {
    const token = TokenImplementation__factory.connect(
      wrappedAddress,
      provider
    );
    const chainId = (await token.chainId()) as ChainId; // origin chain
    const assetAddress = await token.nativeContract(); // origin address
    return {
      isWrapped: true,
      chainId,
      assetAddress: arrayify(assetAddress),
    };
  }
  return {
    isWrapped: false,
    chainId: coalesceChainId(lookupChain),
    assetAddress: zeroPad(arrayify(wrappedAddress), 32),
  };
}

export async function getOriginalAssetTerra(
  client: TerraLCDClient,
  wrappedAddress: string
) {
  return getOriginalAssetCosmWasm(client, wrappedAddress, CHAIN_ID_TERRA);
}

/**
 * Returns information about the asset
 * @param wrappedAddress Address of the asset in wormhole wrapped format (hex string)
 * @param client WASM api client
 * @returns Information about the asset
 */
export async function getOriginalAssetInjective(
  wrappedAddress: string,
  client: ChainGrpcWasmApi
): Promise<WormholeWrappedInfo> {
  const chainId = CHAIN_ID_INJECTIVE;
  if (isNativeCosmWasmDenom(CHAIN_ID_INJECTIVE, wrappedAddress)) {
    return {
      isWrapped: false,
      chainId: chainId,
      assetAddress: hexToUint8Array(buildTokenId(chainId, wrappedAddress)),
    };
  }
  try {
    const injWrappedAddress = tryHexToNativeAssetString(
      wrappedAddress,
      CHAIN_ID_INJECTIVE
    );
    const queryResult = await client.fetchSmartContractState(
      injWrappedAddress,
      Buffer.from(
        JSON.stringify({
          wrapped_asset_info: {},
        })
      ).toString("base64")
    );
    let result: any = null;
    if (typeof queryResult.data === "string") {
      result = JSON.parse(
        Buffer.from(queryResult.data, "base64").toString("utf-8")
      );
      return {
        isWrapped: true,
        chainId: result.asset_chain,
        assetAddress: new Uint8Array(
          Buffer.from(result.asset_address, "base64")
        ),
      };
    }
  } catch (e) {
    console.error("getOriginalAssetInjective() failed:", e);
  }
  return {
    isWrapped: false,
    chainId: chainId,
    assetAddress: hexToUint8Array(buildTokenId(chainId, wrappedAddress)),
  };
}

export async function getOriginalAssetXpla(
  client: XplaLCDClient,
  wrappedAddress: string
) {
  return getOriginalAssetCosmWasm(client, wrappedAddress, "xpla");
}

export async function getOriginalAssetCosmWasm(
  client: TerraLCDClient | XplaLCDClient,
  wrappedAddress: string,
  lookupChain: CosmWasmChainId | CosmWasmChainName
): Promise<WormholeWrappedInfo> {
  const chainId = coalesceCosmWasmChainId(lookupChain);
  if (isNativeCosmWasmDenom(chainId, wrappedAddress)) {
    return {
      isWrapped: false,
      chainId: chainId,
      assetAddress:
        chainId === CHAIN_ID_TERRA
          ? buildNativeId(wrappedAddress)
          : hexToUint8Array(buildTokenId(chainId, wrappedAddress)),
    };
  }
  try {
    const result: {
      asset_address: string;
      asset_chain: ChainId;
      bridge: string;
    } = await client.wasm.contractQuery(wrappedAddress, {
      wrapped_asset_info: {},
    });
    if (result) {
      return {
        isWrapped: true,
        chainId: result.asset_chain,
        assetAddress: new Uint8Array(
          Buffer.from(result.asset_address, "base64")
        ),
      };
    }
  } catch (e) {}
  return {
    isWrapped: false,
    chainId: chainId,
    assetAddress:
      chainId === CHAIN_ID_TERRA
        ? zeroPad(canonicalAddress(wrappedAddress), 32)
        : hexToUint8Array(buildTokenId(chainId, wrappedAddress)),
  };
}

/**
 * Returns a origin chain and asset address on {originChain} for a provided Wormhole wrapped address
 * @param connection
 * @param tokenBridgeAddress
 * @param mintAddress
 * @returns
 */
export async function getOriginalAssetSol(
  connection: Connection,
  tokenBridgeAddress: string,
  mintAddress: string
): Promise<WormholeWrappedInfo> {
  if (mintAddress) {
    // TODO: share some of this with getIsWrappedAssetSol, like a getWrappedMetaAccountAddress or something
    const { parse_wrapped_meta, wrapped_meta_address } =
      await importTokenWasm();
    const wrappedMetaAddress = wrapped_meta_address(
      tokenBridgeAddress,
      new PublicKey(mintAddress).toBytes()
    );
    const wrappedMetaAddressPK = new PublicKey(wrappedMetaAddress);
    const wrappedMetaAccountInfo = await connection.getAccountInfo(
      wrappedMetaAddressPK
    );
    if (wrappedMetaAccountInfo) {
      const parsed = parse_wrapped_meta(wrappedMetaAccountInfo.data);
      return {
        isWrapped: true,
        chainId: parsed.chain,
        assetAddress: parsed.token_address,
      };
    }
  }
  try {
    return {
      isWrapped: false,
      chainId: CHAIN_ID_SOLANA,
      assetAddress: new PublicKey(mintAddress).toBytes(),
    };
  } catch (e) {}
  return {
    isWrapped: false,
    chainId: CHAIN_ID_SOLANA,
    assetAddress: new Uint8Array(32),
  };
}

/**
 * Returns an origin chain and asset address on {originChain} for a provided Wormhole wrapped address
 * @param client Algodv2 client
 * @param tokenBridgeId Application ID of the token bridge
 * @param assetId Algorand asset index
 * @returns wrapped wormhole information structure
 */
export async function getOriginalAssetAlgorand(
  client: Algodv2,
  tokenBridgeId: bigint,
  assetId: bigint
): Promise<WormholeWrappedInfo> {
  let retVal: WormholeWrappedInfo = {
    isWrapped: false,
    chainId: CHAIN_ID_ALGORAND,
    assetAddress: new Uint8Array(),
  };
  retVal.isWrapped = await getIsWrappedAssetAlgorand(
    client,
    tokenBridgeId,
    assetId
  );
  if (!retVal.isWrapped) {
    retVal.assetAddress = zeroPad(arrayify(ethers.BigNumber.from(assetId)), 32);
    return retVal;
  }
  const assetInfo = await client.getAssetByID(safeBigIntToNumber(assetId)).do();
  const lsa = assetInfo.params.creator;
  const dls = await decodeLocalState(client, tokenBridgeId, lsa);
  const dlsBuffer: Buffer = Buffer.from(dls);
  retVal.chainId = dlsBuffer.readInt16BE(92) as ChainId;
  retVal.assetAddress = new Uint8Array(dlsBuffer.slice(60, 60 + 32));
  return retVal;
}

export async function getOriginalAssetNear(
  provider: Provider,
  tokenAccount: string,
  assetAccount: string
): Promise<WormholeWrappedInfo> {
  const retVal: WormholeWrappedInfo = {
    isWrapped: false,
    chainId: CHAIN_ID_NEAR,
    assetAddress: new Uint8Array(),
  };
  retVal.isWrapped = await getIsWrappedAssetNear(tokenAccount, assetAccount);
  if (!retVal.isWrapped) {
    retVal.assetAddress = assetAccount
      ? arrayify(sha256(Buffer.from(assetAccount)))
      : zeroPad(arrayify("0x"), 32);
    return retVal;
  }

  const buf = await callFunctionNear(
    provider,
    tokenAccount,
    "get_original_asset",
    {
      token: assetAccount,
    }
  );

  retVal.chainId = buf[1];
  retVal.assetAddress = hexToUint8Array(buf[0]);

  return retVal;
}

export async function getOriginalAssetAptos(
  client: AptosClient,
  tokenBridgeAddress: string,
  assetAddress: string,
): Promise<WormholeWrappedInfo> {
  const originInfo = (
    await client.getAccountResource(assetAddress, `${tokenBridgeAddress}::state::OriginInfo`)
  ).data as OriginInfo;
  if (!!originInfo) {
    // wrapped asset
    const chainId = originInfo.token_chain.number;
    assertChain(chainId);
    const assetAddress = Uint8Array.from(Buffer.from(originInfo.token_address.external_address));
    return {
      isWrapped: true,
      chainId,
      assetAddress,
    };
  } else {
    // native asset
    return {
      isWrapped: false,
      chainId: CHAIN_ID_APTOS,
      // TODO: should we return address or fully qualified type?
      assetAddress: Uint8Array.from(Buffer.from(assetAddress)),
    };
  }
}
