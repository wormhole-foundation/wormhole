import {
  Commitment,
  Connection,
  PublicKey,
  PublicKeyInitData,
  Transaction,
} from "@solana/web3.js";
import { MsgExecuteContract } from "@terra-money/terra.js";
import { Algodv2 } from "algosdk";
import { Types } from "aptos";
import BN from "bn.js";
import { ethers, Overrides } from "ethers";
import { fromUint8Array } from "js-base64";
import {
  TransactionSignerPair,
  _parseVAAAlgorand,
  _submitVAAAlgorand,
} from "../algorand";
import { Bridge__factory } from "../ethers-contracts";
import { submitVAAOnInjective } from "./redeem";
import { FunctionCallOptions } from "near-api-js/lib/account";
import { Provider } from "near-api-js/lib/providers";
import { callFunctionNear } from "../utils";
import { MsgExecuteContract as XplaMsgExecuteContract } from "@xpla/xpla.js";
import {
  createWrappedCoin as createWrappedCoinAptos,
  createWrappedCoinType as createWrappedCoinTypeAptos,
} from "../aptos";
import { createCreateWrappedInstruction } from "../solana/tokenBridge";
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

export const createWrappedOnInjective = submitVAAOnInjective;

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
