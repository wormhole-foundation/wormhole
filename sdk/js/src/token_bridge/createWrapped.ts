import {
  JsonRpcProvider,
  SUI_CLOCK_OBJECT_ID,
  TransactionBlock,
} from "@mysten/sui.js";
import {
  Commitment,
  Connection,
  PublicKey,
  PublicKeyInitData,
  Transaction,
} from "@solana/web3.js";
import { MsgExecuteContract } from "@terra-money/terra.js";
import { MsgExecuteContract as XplaMsgExecuteContract } from "@xpla/xpla.js";
import { Algodv2 } from "algosdk";
import { Types } from "aptos";
import BN from "bn.js";
import { Overrides, ethers } from "ethers";
import { fromUint8Array } from "js-base64";
import { FunctionCallOptions } from "near-api-js/lib/account";
import { Provider } from "near-api-js/lib/providers";
import { TransactionSignerPair, _submitVAAAlgorand } from "../algorand";
import {
  createWrappedCoin as createWrappedCoinAptos,
  createWrappedCoinType as createWrappedCoinTypeAptos,
} from "../aptos";
import { Bridge__factory } from "../ethers-contracts";
import { createCreateWrappedInstruction } from "../solana/tokenBridge";
import {
  getOwnedObjectId,
  getPackageId,
  getUpgradeCapObjectId,
  getWrappedCoinType,
  publishCoin,
  uint8ArrayToBCS,
} from "../sui";
import { callFunctionNear } from "../utils";
import { SignedVaa } from "../vaa";

export async function createWrappedOnEth(
  tokenBridgeAddress: string,
  signer: ethers.Signer,
  signedVAA: Uint8Array,
  overrides: Overrides & { from?: string | Promise<string> } = {}
): Promise<ethers.ContractReceipt> {
  const bridge = Bridge__factory.connect(tokenBridgeAddress, signer);
  const v = await bridge.createWrapped(signedVAA, overrides);
  const receipt = await v.wait();
  return receipt;
}

export async function createWrappedOnTerra(
  tokenBridgeAddress: string,
  walletAddress: string,
  signedVAA: Uint8Array
): Promise<MsgExecuteContract> {
  return new MsgExecuteContract(walletAddress, tokenBridgeAddress, {
    submit_vaa: {
      data: fromUint8Array(signedVAA),
    },
  });
}

export function createWrappedOnXpla(
  tokenBridgeAddress: string,
  walletAddress: string,
  signedVAA: Uint8Array
): XplaMsgExecuteContract {
  return new XplaMsgExecuteContract(walletAddress, tokenBridgeAddress, {
    submit_vaa: {
      data: fromUint8Array(signedVAA),
    },
  });
}

export async function createWrappedOnSolana(
  connection: Connection,
  bridgeAddress: PublicKeyInitData,
  tokenBridgeAddress: PublicKeyInitData,
  payerAddress: PublicKeyInitData,
  signedVaa: SignedVaa,
  commitment?: Commitment
): Promise<Transaction> {
  const transaction = new Transaction().add(
    createCreateWrappedInstruction(
      tokenBridgeAddress,
      bridgeAddress,
      payerAddress,
      signedVaa
    )
  );
  const { blockhash } = await connection.getLatestBlockhash(commitment);
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  return transaction;
}

export async function createWrappedOnAlgorand(
  client: Algodv2,
  tokenBridgeId: bigint,
  bridgeId: bigint,
  senderAddr: string,
  attestVAA: Uint8Array
): Promise<TransactionSignerPair[]> {
  return await _submitVAAAlgorand(
    client,
    tokenBridgeId,
    bridgeId,
    attestVAA,
    senderAddr
  );
}

export async function createWrappedOnNear(
  provider: Provider,
  tokenBridge: string,
  attestVAA: Uint8Array
): Promise<FunctionCallOptions[]> {
  const vaa = Buffer.from(attestVAA).toString("hex");
  const res = await callFunctionNear(
    provider,
    tokenBridge,
    "deposit_estimates"
  );
  const msgs = [
    {
      contractId: tokenBridge,
      methodName: "submit_vaa",
      args: { vaa },
      attachedDeposit: new BN(res[1]),
      gas: new BN("150000000000000"),
    },
  ];
  msgs.push({ ...msgs[0] });
  return msgs;
}

/**
 * Constructs payload to create wrapped asset type. The type is of form `{{address}}::coin::T`,
 * where address is `sha256_hash(tokenBridgeAddress | chainID | "::" | originAddress | 0xFF)`.
 *
 * Note that the typical createWrapped call is broken into two parts on Aptos because we must first
 * create the CoinType that is used by `create_wrapped_coin<CoinType>`. Since it's not possible to
 * create a resource and use it in the same transaction, this is broken into separate transactions.
 * @param tokenBridgeAddress Address of token bridge
 * @param attestVAA Bytes of attest VAA
 * @returns Transaction payload
 */
export function createWrappedTypeOnAptos(
  tokenBridgeAddress: string,
  attestVAA: Uint8Array
): Types.EntryFunctionPayload {
  return createWrappedCoinTypeAptos(tokenBridgeAddress, attestVAA);
}

/**
 * Constructs payload to create wrapped asset.
 *
 * Note that this function is typically called in tandem with `createWrappedTypeOnAptos` because
 * we must first create the CoinType that is used by `create_wrapped_coin<CoinType>`. Since it's
 * not possible to create a resource and use it in the same transaction, this is broken into
 * separate transactions.
 * @param tokenBridgeAddress Address of token bridge
 * @param attestVAA Bytes of attest VAA
 * @returns Transaction payload
 */
export function createWrappedOnAptos(
  tokenBridgeAddress: string,
  attestVAA: Uint8Array
): Types.EntryFunctionPayload {
  return createWrappedCoinAptos(tokenBridgeAddress, attestVAA);
}

export async function createWrappedOnSuiPrepare(
  provider: JsonRpcProvider,
  coreBridgeStateObjectId: string,
  tokenBridgeStateObjectId: string,
  decimals: number,
  signerAddress: string
): Promise<TransactionBlock> {
  return publishCoin(
    provider,
    coreBridgeStateObjectId,
    tokenBridgeStateObjectId,
    decimals,
    signerAddress
  );
}

export async function createWrappedOnSui(
  provider: JsonRpcProvider,
  coreBridgeStateObjectId: string,
  tokenBridgeStateObjectId: string,
  signerAddress: string,
  coinPackageId: string,
  wrappedAssetSetupType: string,
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

  // WrappedAssetSetup looks like
  // 0x92d81f28c167d90f84638c654b412fe7fa8e55bdfac7f638bdcf70306289be86::create_wrapped::WrappedAssetSetup<0xa40e0511f7d6531dd2dfac0512c7fd4a874b76f5994985fb17ee04501a2bb050::coin::COIN, 0x4eb7c5bca3759ab3064b46044edb5668c9066be8a543b28b58375f041f876a80::version_control::V__0_1_1>
  const wrappedAssetSetupObjectId = await getOwnedObjectId(
    provider,
    signerAddress,
    wrappedAssetSetupType
  );
  if (!wrappedAssetSetupObjectId) {
    throw new Error(`WrappedAssetSetup not found`);
  }

  // Get coin upgrade capability
  const coinUpgradeCapObjectId = await getUpgradeCapObjectId(
    provider,
    signerAddress,
    coinPackageId
  );
  if (!coinUpgradeCapObjectId) {
    throw new Error(
      `Coin upgrade cap not found for ${coinType} under owner ${signerAddress}. You must call 'createWrappedOnSuiPrepare' first.`
    );
  }

  // Get TokenBridgeMessage
  const tx = new TransactionBlock();
  const [vaa] = tx.moveCall({
    target: `${coreBridgePackageId}::vaa::parse_and_verify`,
    arguments: [
      tx.object(coreBridgeStateObjectId),
      tx.pure(uint8ArrayToBCS(attestVAA)),
      tx.object(SUI_CLOCK_OBJECT_ID),
    ],
  });
  const [message] = tx.moveCall({
    target: `${tokenBridgePackageId}::vaa::verify_only_once`,
    arguments: [tx.object(tokenBridgeStateObjectId), vaa],
  });

  // Construct complete registration payload
  const versionType = wrappedAssetSetupType.split(", ")[1].replace(">", ""); // ugh
  tx.moveCall({
    target: `${tokenBridgePackageId}::create_wrapped::complete_registration`,
    arguments: [
      tx.object(tokenBridgeStateObjectId),
      tx.object(coinMetadataObjectId),
      tx.object(wrappedAssetSetupObjectId),
      tx.object(coinUpgradeCapObjectId),
      message,
    ],
    typeArguments: [coinType, versionType],
  });
  return tx;
}
