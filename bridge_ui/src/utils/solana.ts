import { WalletContextState } from "@solana/wallet-adapter-react";
import { Connection, Transaction } from "@solana/web3.js";

export async function signSendAndConfirm(
  wallet: WalletContextState,
  connection: Connection,
  transaction: Transaction
) {
  const signed = await wallet.signTransaction(transaction);
  const txid = await connection.sendRawTransaction(signed.serialize());
  await connection.confirmTransaction(txid);
  return txid;
}

export async function signSendConfirmAndGet(
  wallet: WalletContextState,
  connection: Connection,
  transaction: Transaction
) {
  const txid = await signSendAndConfirm(wallet, connection, transaction);
  return await connection.getTransaction(txid);
}
