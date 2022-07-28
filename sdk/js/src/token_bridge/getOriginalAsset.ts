import { Connection, PublicKey } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { Algodv2 } from "algosdk";
import { ethers } from "ethers";
import { arrayify, zeroPad } from "ethers/lib/utils";
import { decodeLocalState } from "../algorand";
import { buildTokenId } from "../cosmwasm/address";
import { TokenImplementation__factory } from "../ethers-contracts";
import { importTokenWasm } from "../solana/wasm";
import { buildNativeId, isNativeDenom } from "../terra";
import { canonicalAddress } from "../cosmos";
import {
  ChainId,
  ChainName,
  CHAIN_ID_ALGORAND,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  coalesceChainId,
  hexToUint8Array,
} from "../utils";
import { safeBigIntToNumber } from "../utils/bigint";
import {
  getIsWrappedAssetAlgorand,
  getIsWrappedAssetEth,
} from "./getIsWrappedAsset";

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
  client: LCDClient,
  wrappedAddress: string
) {
  return getOriginalAssetCosmWasm(client, wrappedAddress, CHAIN_ID_TERRA);
}

export async function getOriginalAssetCosmWasm(
  client: LCDClient,
  wrappedAddress: string,
  lookupChain: ChainId | ChainName
): Promise<WormholeWrappedInfo> {
  const chainId = coalesceChainId(lookupChain);
  if (isNativeDenom(wrappedAddress)) {
    return {
      isWrapped: false,
      chainId: chainId,
      assetAddress:
        chainId === CHAIN_ID_TERRA
          ? buildNativeId(wrappedAddress)
          : hexToUint8Array(buildTokenId(wrappedAddress)),
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
        : hexToUint8Array(buildTokenId(wrappedAddress)),
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
