import { Connection, PublicKey, Transaction } from "@solana/web3.js";
import { MsgExecuteContract } from "@terra-money/terra.js";
import { Algodv2 } from "algosdk";
import { ethers, Overrides } from "ethers";
import { fromUint8Array } from "js-base64";
import { TransactionSignerPair, _submitVAAAlgorand } from "../algorand";
import { Bridge__factory } from "../ethers-contracts";
import { ixFromRust } from "../solana";
import { importTokenWasm } from "../solana/wasm";
import { Account as nearAccount } from "near-api-js";
const BN = require("bn.js");
const nearApi = require("near-api-js");

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

export async function createWrappedOnSolana(
  connection: Connection,
  bridgeAddress: string,
  tokenBridgeAddress: string,
  payerAddress: string,
  signedVAA: Uint8Array
): Promise<Transaction> {
  const { create_wrapped_ix } = await importTokenWasm();
  const ix = ixFromRust(
    create_wrapped_ix(
      tokenBridgeAddress,
      bridgeAddress,
      payerAddress,
      signedVAA
    )
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
  return await _submitVAAAlgorand(
    client,
    tokenBridgeId,
    bridgeId,
    attestVAA,
    senderAddr
  );
}

export async function createWrappedOnNear(
  client: nearAccount,
  tokenBridge: string,
  attestVAA: Uint8Array
): Promise<string> {
  // Could we just pass in the vaa already as hex?
  let vaa = Buffer.from(attestVAA).toString("hex");

  let res = await client.viewFunction(tokenBridge, "deposit_estimates", {});

  let result = await client.functionCall({
    contractId: tokenBridge,
    methodName: "submit_vaa",
    args: { vaa: vaa },
    attachedDeposit: new BN(res[1]),
    gas: new BN("300000000000000"),
  });

  result = await client.functionCall({
    contractId: tokenBridge,
    methodName: "submit_vaa",
    args: { vaa: vaa },
    attachedDeposit: new BN(res[1]),
    gas: new BN("300000000000000"),
  });

  return nearApi.providers.getTransactionLastResult(result);
}
