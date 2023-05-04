import {
  JsonRpcProvider,
  SUI_CLOCK_OBJECT_ID,
  TransactionBlock,
} from "@mysten/sui.js";
import {
  Commitment,
  Connection,
  Keypair,
  PublicKey,
  PublicKeyInitData,
  Transaction,
} from "@solana/web3.js";
import { MsgExecuteContract } from "@terra-money/terra.js";
import { MsgExecuteContract as XplaMsgExecuteContract } from "@xpla/xpla.js";
import {
  Algodv2,
  OnApplicationComplete,
  SuggestedParams,
  bigIntToBytes,
  decodeAddress,
  getApplicationAddress,
  makeApplicationCallTxnFromObject,
  makePaymentTxnWithSuggestedParamsFromObject,
} from "algosdk";
import { Types } from "aptos";
import BN from "bn.js";
import { PayableOverrides, ethers } from "ethers";
import { FunctionCallOptions } from "near-api-js/lib/account";
import { Provider } from "near-api-js/lib/providers";
import { getIsWrappedAssetNear } from ".";
import { TransactionSignerPair, getMessageFee, optin } from "../algorand";
import { attestToken as attestTokenAptos } from "../aptos";
import { isNativeDenomXpla } from "../cosmwasm";
import { Bridge__factory } from "../ethers-contracts";
import { createBridgeFeeTransferInstruction } from "../solana";
import { createAttestTokenInstruction } from "../solana/tokenBridge";
import { getPackageId } from "../sui/utils";
import { isNativeDenom } from "../terra";
import {
  ChainId,
  callFunctionNear,
  hashAccount,
  textToHexString,
  textToUint8Array,
  uint8ArrayToHex,
} from "../utils";
import { safeBigIntToNumber } from "../utils/bigint";
import { createNonce } from "../utils/createNonce";

export async function attestFromEth(
  tokenBridgeAddress: string,
  signer: ethers.Signer,
  tokenAddress: string,
  overrides: PayableOverrides & { from?: string | Promise<string> } = {}
): Promise<ethers.ContractReceipt> {
  const bridge = Bridge__factory.connect(tokenBridgeAddress, signer);
  const v = await bridge.attestToken(tokenAddress, createNonce(), overrides);
  const receipt = await v.wait();
  return receipt;
}

export async function attestFromTerra(
  tokenBridgeAddress: string,
  walletAddress: string,
  asset: string
): Promise<MsgExecuteContract> {
  const nonce = Math.round(Math.random() * 100000);
  const isNativeAsset = isNativeDenom(asset);
  return new MsgExecuteContract(walletAddress, tokenBridgeAddress, {
    create_asset_meta: {
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
  });
}

export function attestFromXpla(
  tokenBridgeAddress: string,
  walletAddress: string,
  asset: string
): XplaMsgExecuteContract {
  const nonce = Math.round(Math.random() * 100000);
  const isNativeAsset = isNativeDenomXpla(asset);
  return new XplaMsgExecuteContract(walletAddress, tokenBridgeAddress, {
    create_asset_meta: {
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
  });
}

export async function attestFromSolana(
  connection: Connection,
  bridgeAddress: PublicKeyInitData,
  tokenBridgeAddress: PublicKeyInitData,
  payerAddress: PublicKeyInitData,
  mintAddress: PublicKeyInitData,
  commitment?: Commitment
): Promise<Transaction> {
  const nonce = createNonce().readUInt32LE(0);
  const transferIx = await createBridgeFeeTransferInstruction(
    connection,
    bridgeAddress,
    payerAddress
  );
  const messageKey = Keypair.generate();
  const attestIx = createAttestTokenInstruction(
    tokenBridgeAddress,
    bridgeAddress,
    payerAddress,
    mintAddress,
    messageKey.publicKey,
    nonce
  );
  const transaction = new Transaction().add(transferIx, attestIx);
  const { blockhash } = await connection.getLatestBlockhash(commitment);
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  transaction.partialSign(messageKey);
  return transaction;
}

/**
 * Attest an already created asset
 * If you create a new asset on algorand and want to transfer it elsewhere,
 * you create an attestation for it on algorand... pass that vaa to the target chain..
 * submit it.. then you can transfer from algorand to that target chain
 * @param client An Algodv2 client
 * @param tokenBridgeId The ID of the token bridge
 * @param senderAcct The account paying fees
 * @param assetId The asset index
 * @returns Transaction ID
 */
export async function attestFromAlgorand(
  client: Algodv2,
  tokenBridgeId: bigint,
  bridgeId: bigint,
  senderAddr: string,
  assetId: bigint
): Promise<TransactionSignerPair[]> {
  const tbAddr: string = getApplicationAddress(tokenBridgeId);
  const decTbAddr: Uint8Array = decodeAddress(tbAddr).publicKey;
  const aa: string = uint8ArrayToHex(decTbAddr);
  const txs: TransactionSignerPair[] = [];
  // "attestFromAlgorand::emitterAddr"
  const { addr: emitterAddr, txs: emitterOptInTxs } = await optin(
    client,
    senderAddr,
    bridgeId,
    BigInt(0),
    aa
  );
  txs.push(...emitterOptInTxs);

  let creatorAddr = "";
  let creatorAcctInfo;
  const bPgmName: Uint8Array = textToUint8Array("attestToken");

  if (assetId !== BigInt(0)) {
    const assetInfo = await client
      .getAssetByID(safeBigIntToNumber(assetId))
      .do();
    creatorAcctInfo = await client
      .accountInformation(assetInfo.params.creator)
      .do();
    if (creatorAcctInfo["auth-addr"] === tbAddr) {
      throw new Error("Cannot re-attest wormhole assets");
    }
  }

  const result = await optin(
    client,
    senderAddr,
    tokenBridgeId,
    assetId,
    textToHexString("native")
  );
  creatorAddr = result.addr;
  txs.push(...result.txs);

  const suggParams: SuggestedParams = await client.getTransactionParams().do();

  const firstTxn = makeApplicationCallTxnFromObject({
    from: senderAddr,
    appIndex: safeBigIntToNumber(tokenBridgeId),
    onComplete: OnApplicationComplete.NoOpOC,
    appArgs: [textToUint8Array("nop")],
    suggestedParams: suggParams,
  });
  txs.push({ tx: firstTxn, signer: null });

  const mfee = await getMessageFee(client, bridgeId);
  if (mfee > BigInt(0)) {
    const feeTxn = makePaymentTxnWithSuggestedParamsFromObject({
      from: senderAddr,
      suggestedParams: suggParams,
      to: getApplicationAddress(tokenBridgeId),
      amount: mfee,
    });
    txs.push({ tx: feeTxn, signer: null });
  }

  let accts: string[] = [
    emitterAddr,
    creatorAddr,
    getApplicationAddress(bridgeId),
  ];

  if (creatorAcctInfo) {
    accts.push(creatorAcctInfo["address"]);
  }

  let appTxn = makeApplicationCallTxnFromObject({
    appArgs: [bPgmName, bigIntToBytes(assetId, 8)],
    accounts: accts,
    appIndex: safeBigIntToNumber(tokenBridgeId),
    foreignApps: [safeBigIntToNumber(bridgeId)],
    foreignAssets: [safeBigIntToNumber(assetId)],
    from: senderAddr,
    onComplete: OnApplicationComplete.NoOpOC,
    suggestedParams: suggParams,
  });
  if (mfee > BigInt(0)) {
    appTxn.fee *= 3;
  } else {
    appTxn.fee *= 2;
  }
  txs.push({ tx: appTxn, signer: null });

  return txs;
}

export async function attestTokenFromNear(
  provider: Provider,
  coreBridge: string,
  tokenBridge: string,
  asset: string
): Promise<FunctionCallOptions[]> {
  const options: FunctionCallOptions[] = [];
  const messageFee = await callFunctionNear(
    provider,
    coreBridge,
    "message_fee"
  );
  if (!getIsWrappedAssetNear(tokenBridge, asset)) {
    const { isRegistered } = await hashAccount(provider, tokenBridge, asset);
    if (!isRegistered) {
      // The account has not been registered. The first user to attest a non-wormhole token pays for the space
      options.push({
        contractId: tokenBridge,
        methodName: "register_account",
        args: { account: asset },
        gas: new BN("100000000000000"),
        attachedDeposit: new BN("2000000000000000000000"), // 0.002 NEAR
      });
    }
  }
  options.push({
    contractId: tokenBridge,
    methodName: "attest_token",
    args: { token: asset, message_fee: messageFee },
    attachedDeposit: new BN("3000000000000000000000").add(new BN(messageFee)), // 0.003 NEAR
    gas: new BN("100000000000000"),
  });
  return options;
}

export async function attestNearFromNear(
  provider: Provider,
  coreBridge: string,
  tokenBridge: string
): Promise<FunctionCallOptions> {
  const messageFee =
    (await callFunctionNear(provider, coreBridge, "message_fee")) + 1;
  return {
    contractId: tokenBridge,
    methodName: "attest_near",
    args: { message_fee: messageFee },
    attachedDeposit: new BN(messageFee),
    gas: new BN("100000000000000"),
  };
}

/**
 * Attest given token from Aptos.
 * @param tokenBridgeAddress Address of token bridge
 * @param tokenChain Origin chain ID
 * @param tokenAddress Address of token on origin chain
 * @returns Transaction payload
 */
export function attestFromAptos(
  tokenBridgeAddress: string,
  tokenChain: ChainId,
  tokenAddress: string
): Types.EntryFunctionPayload {
  return attestTokenAptos(tokenBridgeAddress, tokenChain, tokenAddress);
}

export async function attestFromSui(
  provider: JsonRpcProvider,
  coreBridgeStateObjectId: string,
  tokenBridgeStateObjectId: string,
  coinType: string,
  feeAmount: BigInt = BigInt(0),
  coreBridgePackageId?: string,
  tokenBridgePackageId?: string
): Promise<TransactionBlock> {
  const metadata = await provider.getCoinMetadata({ coinType });
  if (metadata === null || metadata.id === null) {
    throw new Error(`Coin metadata ID for type ${coinType} not found`);
  }

  [coreBridgePackageId, tokenBridgePackageId] = await Promise.all([
    coreBridgePackageId
      ? Promise.resolve(coreBridgePackageId)
      : getPackageId(provider, coreBridgeStateObjectId),
    tokenBridgePackageId
      ? Promise.resolve(tokenBridgePackageId)
      : getPackageId(provider, tokenBridgeStateObjectId),
  ]);
  const tx = new TransactionBlock();
  const [feeCoin] = tx.splitCoins(tx.gas, [tx.pure(feeAmount)]);
  const [messageTicket] = tx.moveCall({
    target: `${tokenBridgePackageId}::attest_token::attest_token`,
    arguments: [
      tx.object(tokenBridgeStateObjectId),
      tx.object(metadata.id),
      tx.pure(createNonce().readUInt32LE()),
    ],
    typeArguments: [coinType],
  });
  tx.moveCall({
    target: `${coreBridgePackageId}::publish_message::publish_message`,
    arguments: [
      tx.object(coreBridgeStateObjectId),
      feeCoin,
      messageTicket,
      tx.object(SUI_CLOCK_OBJECT_ID),
    ],
  });
  return tx;
}
