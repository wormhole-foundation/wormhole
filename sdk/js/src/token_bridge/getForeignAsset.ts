import { Connection, PublicKey } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { Algodv2 } from "algosdk";
import { ethers } from "ethers";
import { fromUint8Array } from "js-base64";
import {
  calcLogicSigAccount,
  decodeLocalState,
  hexToNativeAssetBigIntAlgorand,
} from "../algorand";
import { Bridge__factory } from "../ethers-contracts";
import { importTokenWasm } from "../solana/wasm";
import { ChainId, CHAIN_ID_ALGORAND } from "../utils";

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
  originChain: ChainId,
  originAsset: Uint8Array
) {
  const tokenBridge = Bridge__factory.connect(tokenBridgeAddress, provider);
  try {
    return await tokenBridge.wrappedAsset(originChain, originAsset);
  } catch (e) {
    return null;
  }
}

export async function getForeignAssetTerra(
  tokenBridgeAddress: string,
  client: LCDClient,
  originChain: ChainId,
  originAsset: Uint8Array
) {
  try {
    const result: { address: string } = await client.wasm.contractQuery(
      tokenBridgeAddress,
      {
        wrapped_registry: {
          chain: originChain,
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
  originChain: ChainId,
  originAsset: Uint8Array
) {
  const { wrapped_address } = await importTokenWasm();
  const wrappedAddress = wrapped_address(
    tokenBridgeAddress,
    originAsset,
    originChain
  );
  const wrappedAddressPK = new PublicKey(wrappedAddress);
  const wrappedAssetAccountInfo = await connection.getAccountInfo(
    wrappedAddressPK
  );
  return wrappedAssetAccountInfo ? wrappedAddressPK.toString() : null;
}

export async function getForeignAssetAlgorand(
  client: Algodv2,
  tokenBridgeId: bigint,
  chain: ChainId,
  contract: string
): Promise<bigint | null> {
  if (chain === CHAIN_ID_ALGORAND) {
    return hexToNativeAssetBigIntAlgorand(contract);
  } else {
    let { lsa, doesExist } = await calcLogicSigAccount(
      client,
      tokenBridgeId,
      BigInt(chain),
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
