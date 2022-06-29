import { Connection, Keypair, PublicKey, Transaction } from "@solana/web3.js";
import { MsgExecuteContract } from "@terra-money/terra.js";
import { MsgExecuteContract as MsgExecuteContractInjective } from "@injectivelabs/sdk-ts";
import {
  Algodv2,
  bigIntToBytes,
  decodeAddress,
  getApplicationAddress,
  makeApplicationCallTxnFromObject,
  makePaymentTxnWithSuggestedParamsFromObject,
  OnApplicationComplete,
  SuggestedParams,
} from "algosdk";
import { ethers, PayableOverrides } from "ethers";
import { isNativeDenom, isNativeDenomInjective } from "..";
import { getMessageFee, optin, TransactionSignerPair } from "../algorand";
import { Bridge__factory } from "../ethers-contracts";
import { getBridgeFeeIx, ixFromRust } from "../solana";
import { importTokenWasm } from "../solana/wasm";
import { textToHexString, textToUint8Array, uint8ArrayToHex } from "../utils";
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

// For native assets the asset string is the denomination.
// For foreign assets the asset string is the inj address of the foreign asset
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
    msg: {
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
    },
  });
}

export async function attestFromSolana(
  connection: Connection,
  bridgeAddress: string,
  tokenBridgeAddress: string,
  payerAddress: string,
  mintAddress: string
): Promise<Transaction> {
  const nonce = createNonce().readUInt32LE(0);
  const transferIx = await getBridgeFeeIx(
    connection,
    bridgeAddress,
    payerAddress
  );
  const { attest_ix } = await importTokenWasm();
  const messageKey = Keypair.generate();
  const ix = ixFromRust(
    attest_ix(
      tokenBridgeAddress,
      bridgeAddress,
      payerAddress,
      messageKey.publicKey.toString(),
      mintAddress,
      nonce
    )
  );
  const transaction = new Transaction().add(transferIx, ix);
  const { blockhash } = await connection.getRecentBlockhash();
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
