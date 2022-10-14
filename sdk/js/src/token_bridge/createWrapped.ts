import { Connection, PublicKey, Transaction } from "@solana/web3.js";
import { MsgExecuteContract } from "@terra-money/terra.js";
import { Algodv2 } from "algosdk";
import { AptosAccount, AptosClient, Types } from "aptos";
import BN from "bn.js";
import { ethers, Overrides } from "ethers";
import { fromUint8Array } from "js-base64";
import { Account as nearAccount, providers as nearProviders } from "near-api-js";
import { TransactionSignerPair, _submitVAAAlgorand } from "../algorand";
import { WormholeAptosApi } from "../aptos";
import { Bridge__factory } from "../ethers-contracts";
import { ixFromRust } from "../solana";
import { importTokenWasm } from "../solana/wasm";
import { submitVAAOnInjective } from "./redeem";
import BN from "bn.js";
import { FunctionCallOptions } from "near-api-js/lib/account";
import { Provider } from "near-api-js/lib/providers";
import { callFunctionNear, ChainId } from "../utils";
import { MsgExecuteContract as XplaMsgExecuteContract } from "@xpla/xpla.js";

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
  bridgeAddress: string,
  tokenBridgeAddress: string,
  payerAddress: string,
  signedVAA: Uint8Array
): Promise<Transaction> {
  const { create_wrapped_ix } = await importTokenWasm();
  const ix = ixFromRust(
    create_wrapped_ix(tokenBridgeAddress, bridgeAddress, payerAddress, signedVAA)
  );
  const transaction = new Transaction().add(ix);
  const { blockhash } = await connection.getRecentBlockhash();
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
  return await _submitVAAAlgorand(client, tokenBridgeId, bridgeId, attestVAA, senderAddr);
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

export async function createWrappedOnAptos(
  client: AptosClient,
  sender: AptosAccount,
  tokenBridgeAddress: string,
  tokenChain: ChainId,
  tokenAddress: string,
  signedVAA: Uint8Array
): Promise<Types.Transaction> {
  const api = new WormholeAptosApi(client, undefined, tokenBridgeAddress);
  return api.tokenBridge.createWrappedCoin(sender, tokenChain, tokenAddress, signedVAA);
}
