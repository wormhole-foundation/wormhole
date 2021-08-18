import Wallet from "@project-serum/sol-wallet-adapter";
import { Connection, Transaction } from "@solana/web3.js";

export async function signSendAndConfirm(
  wallet: Wallet,
  connection: Connection,
  transaction: Transaction
) {
  const signed = await wallet.signTransaction(transaction);
  const txid = await connection.sendRawTransaction(signed.serialize());
  await connection.confirmTransaction(txid);
  return txid;
}

export async function signSendConfirmAndGet(
  wallet: Wallet,
  connection: Connection,
  transaction: Transaction
) {
  const txid = await signSendAndConfirm(wallet, connection, transaction);
  return await connection.getTransaction(txid);
}
