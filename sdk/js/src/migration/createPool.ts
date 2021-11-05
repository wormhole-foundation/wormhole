import { Connection, PublicKey, Transaction } from "@solana/web3.js";
import { ixFromRust } from "../solana";
import { importMigrationWasm } from "../solana/wasm";

export default async function createPool(
  connection: Connection,
  payerAddress: string,
  program_id: string,
  payer: string,
  from_mint: string,
  to_mint: string
) {
  const { create_pool } = await importMigrationWasm();
  const ix = ixFromRust(create_pool(program_id, payer, from_mint, to_mint));
  const transaction = new Transaction().add(ix);
  const { blockhash } = await connection.getRecentBlockhash();
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  return transaction;
}
