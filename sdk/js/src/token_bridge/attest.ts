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
import BN from "bn.js";
import { ethers, PayableOverrides } from "ethers";
import { isNativeDenom } from "..";
import { getMessageFee, optin, TransactionSignerPair } from "../algorand";
import { Bridge__factory } from "../ethers-contracts";
import { getBridgeFeeIx, ixFromRust } from "../solana";
import { importTokenWasm } from "../solana/wasm";
import {
  callFunctionNear,
  hashAccount,
  ChainId,
  textToHexString,
  textToUint8Array,
  uint8ArrayToHex,
} from "../utils";
import { safeBigIntToNumber } from "../utils/bigint";
import { createNonce } from "../utils/createNonce";
import { getIsWrappedAssetNear } from ".";
import { isNativeDenomInjective, isNativeDenomXpla } from "../cosmwasm";
import { Provider } from "near-api-js/lib/providers";
import { FunctionCallOptions } from "near-api-js/lib/account";
import { MsgExecuteContract as XplaMsgExecuteContract } from "@xpla/xpla.js";
import { AptosClient, AptosAccount, Types } from "aptos";
import { WormholeAptosApi } from "../aptos";

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

// TODO: do we want to pass in a single assetAddress (instead of tokenChain and tokenAddress) as
// with other APIs above and let user derive the wrapped asset address themselves?
export async function attestFromAptos(
  client: AptosClient,
  sender: AptosAccount,
  tokenBridgeAddress: string,
  tokenChain: ChainId,
  tokenAddress: string
): Promise<Types.Transaction> {
  const api = new WormholeAptosApi(client, undefined, tokenBridgeAddress);
  return api.tokenBridge.attestToken(sender, tokenChain, tokenAddress);
}
