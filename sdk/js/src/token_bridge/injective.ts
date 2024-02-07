import {
  ChainGrpcWasmApi,
  MsgExecuteContractCompat as MsgExecuteContractInjective,
} from "@injectivelabs/sdk-ts";
import {
  buildTokenId,
  isNativeCosmWasmDenom,
  isNativeDenomInjective,
} from "../cosmwasm";
import {
  CHAIN_ID_INJECTIVE,
  ChainId,
  ChainName,
  coalesceChainId,
  hexToUint8Array,
  parseSmartContractStateResponse,
  tryNativeToHexString,
} from "../utils";
import { fromUint8Array } from "js-base64";
import { WormholeWrappedInfo } from "./getOriginalAsset";

/**
 * Creates attestation message
 * @param tokenBridgeAddress Address of Inj token bridge contract
 * @param walletAddress Address of wallet in inj format
 * @param asset Name or address of the asset to be attested
 * For native assets the asset string is the denomination.
 * For foreign assets the asset string is the inj address of the foreign asset
 * @returns Message to be broadcast
 */
export async function attestFromInjective(
  tokenBridgeAddress: string,
  walletAddress: string,
  asset: string
): Promise<MsgExecuteContractInjective> {
  const nonce = Math.round(Math.random() * 100000);
  const isNativeAsset = isNativeDenomInjective(asset);
  return MsgExecuteContractInjective.fromJSON({
    contractAddress: tokenBridgeAddress,
    sender: walletAddress,
    exec: {
      msg: {
        asset_info: isNativeAsset
          ? {
              native_token: { denom: asset },
            }
          : {
              token: {
                contract_addr: asset,
              },
            },
        nonce: nonce,
      },
      action: "create_asset_meta",
    },
  });
}

export const createWrappedOnInjective = submitVAAOnInjective;

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
    const parsed = parseSmartContractStateResponse(queryResult);
    return parsed.address;
  } catch (e) {
    return null;
  }
}

/**
 * Return if the VAA has been redeemed or not
 * @param tokenBridgeAddress The Injective token bridge contract address
 * @param signedVAA The signed VAA byte array
 * @param client Holds the wallet and signing information
 * @returns true if the VAA has been redeemed.
 */
export async function getIsTransferCompletedInjective(
  tokenBridgeAddress: string,
  signedVAA: Uint8Array,
  client: ChainGrpcWasmApi
): Promise<boolean> {
  const queryResult = await client.fetchSmartContractState(
    tokenBridgeAddress,
    Buffer.from(
      JSON.stringify({
        is_vaa_redeemed: {
          vaa: fromUint8Array(signedVAA),
        },
      })
    ).toString("base64")
  );
  const parsed = parseSmartContractStateResponse(queryResult);
  return parsed.is_redeemed;
}

/**
 * Checks if the asset is a wrapped asset
 * @param tokenBridgeAddress The address of the Injective token bridge contract
 * @param client Connection/wallet information
 * @param assetAddress Address of the asset in Injective format
 * @returns true if asset is a wormhole wrapped asset
 */
export async function getIsWrappedAssetInjective(
  tokenBridgeAddress: string,
  client: ChainGrpcWasmApi,
  assetAddress: string
): Promise<boolean> {
  const hexified = tryNativeToHexString(assetAddress, "injective");
  const result = await getForeignAssetInjective(
    tokenBridgeAddress,
    client,
    CHAIN_ID_INJECTIVE,
    new Uint8Array(Buffer.from(hexified))
  );
  if (result === null) {
    return false;
  }
  return true;
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
  if (isNativeCosmWasmDenom(chainId, wrappedAddress)) {
    return {
      isWrapped: false,
      chainId,
      assetAddress: hexToUint8Array(buildTokenId(chainId, wrappedAddress)),
    };
  }
  try {
    const response = await client.fetchSmartContractState(
      wrappedAddress,
      Buffer.from(
        JSON.stringify({
          wrapped_asset_info: {},
        })
      ).toString("base64")
    );
    const parsed = parseSmartContractStateResponse(response);
    return {
      isWrapped: true,
      chainId: parsed.asset_chain,
      assetAddress: new Uint8Array(Buffer.from(parsed.asset_address, "base64")),
    };
  } catch {}
  return {
    isWrapped: false,
    chainId: chainId,
    assetAddress: hexToUint8Array(buildTokenId(chainId, wrappedAddress)),
  };
}

/**
 * Submits the supplied VAA to Injective
 * @param tokenBridgeAddress Address of Inj token bridge contract
 * @param walletAddress Address of wallet in inj format
 * @param signedVAA VAA with the attestation message
 * @returns Message to be broadcast
 */
export async function submitVAAOnInjective(
  tokenBridgeAddress: string,
  walletAddress: string,
  signedVAA: Uint8Array
): Promise<MsgExecuteContractInjective> {
  return MsgExecuteContractInjective.fromJSON({
    contractAddress: tokenBridgeAddress,
    sender: walletAddress,
    exec: {
      msg: {
        data: fromUint8Array(signedVAA),
      },
      action: "submit_vaa",
    },
  });
}
export const redeemOnInjective = submitVAAOnInjective;

/**
 * Creates the necessary messages to transfer an asset
 * @param walletAddress Address of the Inj wallet
 * @param tokenBridgeAddress Address of the token bridge contract
 * @param tokenAddress Address of the token being transferred
 * @param amount Amount of token to be transferred
 * @param recipientChain Destination chain
 * @param recipientAddress Destination wallet address
 * @param relayerFee Relayer fee
 * @param payload Optional payload
 * @returns Transfer messages to be sent on chain
 */
export async function transferFromInjective(
  walletAddress: string,
  tokenBridgeAddress: string,
  tokenAddress: string,
  amount: string,
  recipientChain: ChainId | ChainName,
  recipientAddress: Uint8Array,
  relayerFee: string = "0",
  payload: Uint8Array | null = null
) {
  const recipientChainId = coalesceChainId(recipientChain);
  const nonce = Math.round(Math.random() * 100000);
  const isNativeAsset = isNativeDenomInjective(tokenAddress);
  const mk_action: string = payload
    ? "initiate_transfer_with_payload"
    : "initiate_transfer";
  const mk_initiate_transfer = (info: object) =>
    payload
      ? {
          asset: {
            amount,
            info,
          },
          recipient_chain: recipientChainId,
          recipient: Buffer.from(recipientAddress).toString("base64"),
          fee: relayerFee,
          nonce,
          payload: fromUint8Array(payload),
        }
      : {
          asset: {
            amount,
            info,
          },
          recipient_chain: recipientChainId,
          recipient: Buffer.from(recipientAddress).toString("base64"),
          fee: relayerFee,
          nonce,
        };
  return isNativeAsset
    ? [
        MsgExecuteContractInjective.fromJSON({
          contractAddress: tokenBridgeAddress,
          sender: walletAddress,
          exec: {
            msg: {},
            action: "deposit_tokens",
          },
          funds: { denom: tokenAddress, amount },
        }),
        MsgExecuteContractInjective.fromJSON({
          contractAddress: tokenBridgeAddress,
          sender: walletAddress,
          exec: {
            msg: mk_initiate_transfer({
              native_token: { denom: tokenAddress },
            }),
            action: mk_action,
          },
        }),
      ]
    : [
        MsgExecuteContractInjective.fromJSON({
          contractAddress: tokenAddress,
          sender: walletAddress,
          exec: {
            msg: {
              spender: tokenBridgeAddress,
              amount,
              expires: {
                never: {},
              },
            },
            action: "increase_allowance",
          },
        }),
        MsgExecuteContractInjective.fromJSON({
          contractAddress: tokenBridgeAddress,
          sender: walletAddress,
          exec: {
            msg: mk_initiate_transfer({
              token: { contract_addr: tokenAddress },
            }),
            action: mk_action,
          },
        }),
      ];
}

export const updateWrappedOnInjective = submitVAAOnInjective;
