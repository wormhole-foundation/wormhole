import {
  JsonRpcProvider,
  SUI_CLOCK_OBJECT_ID,
  TransactionBlock,
} from "@mysten/sui.js";
import { Overrides, ethers } from "ethers";
import {
  createWrappedOnAlgorand,
  createWrappedOnAptos,
  createWrappedOnNear,
  createWrappedOnSolana,
  createWrappedOnTerra,
  createWrappedOnXpla,
} from ".";
import { Bridge__factory } from "../ethers-contracts";
import { getPackageId, getWrappedCoinType, uint8ArrayToBCS } from "../sui";

export async function updateWrappedOnEth(
  tokenBridgeAddress: string,
  signer: ethers.Signer,
  signedVAA: Uint8Array,
  overrides: Overrides & { from?: string | Promise<string> } = {}
) {
  const bridge = Bridge__factory.connect(tokenBridgeAddress, signer);
  const v = await bridge.updateWrapped(signedVAA, overrides);
  const receipt = await v.wait();
  return receipt;
}

export const updateWrappedOnTerra = createWrappedOnTerra;

export const updateWrappedOnXpla = createWrappedOnXpla;

export const updateWrappedOnSolana = createWrappedOnSolana;

export const updateWrappedOnAlgorand = createWrappedOnAlgorand;

export const updateWrappedOnNear = createWrappedOnNear;

export const updateWrappedOnAptos = createWrappedOnAptos;

export async function updateWrappedOnSui(
  provider: JsonRpcProvider,
  coreBridgeStateObjectId: string,
  tokenBridgeStateObjectId: string,
  coinPackageId: string,
  attestVAA: Uint8Array,
  coreBridgePackageId?: string,
  tokenBridgePackageId?: string
): Promise<TransactionBlock> {
  [coreBridgePackageId, tokenBridgePackageId] = await Promise.all([
    coreBridgePackageId
      ? Promise.resolve(coreBridgePackageId)
      : getPackageId(provider, coreBridgeStateObjectId),
    tokenBridgePackageId
      ? Promise.resolve(tokenBridgePackageId)
      : getPackageId(provider, tokenBridgeStateObjectId),
  ]);

  // Get coin metadata
  const coinType = getWrappedCoinType(coinPackageId);
  const coinMetadataObjectId = (await provider.getCoinMetadata({ coinType }))
    ?.id;
  if (!coinMetadataObjectId) {
    throw new Error(
      `Coin metadata object not found for coin type ${coinType}.`
    );
  }

  // Get verified VAA
  const tx = new TransactionBlock();
  const [vaa] = tx.moveCall({
    target: `${coreBridgePackageId}::vaa::parse_and_verify`,
    arguments: [
      tx.object(coreBridgeStateObjectId),
      tx.pure(uint8ArrayToBCS(attestVAA)),
      tx.object(SUI_CLOCK_OBJECT_ID),
    ],
  });

  // Get TokenBridgeMessage
  const [message] = tx.moveCall({
    target: `${tokenBridgePackageId}::vaa::verify_only_once`,
    arguments: [tx.object(tokenBridgeStateObjectId), vaa],
  });

  // Construct complete registration payload
  tx.moveCall({
    target: `${tokenBridgePackageId}::create_wrapped::update_attestation`,
    arguments: [
      tx.object(tokenBridgeStateObjectId),
      tx.object(coinMetadataObjectId),
      message,
    ],
    typeArguments: [coinType],
  });
  return tx;
}
