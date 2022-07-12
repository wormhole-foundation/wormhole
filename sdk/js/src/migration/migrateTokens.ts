import { createApproveInstruction } from "@solana/spl-token";
import {
  Commitment,
  Connection,
  PublicKey,
  Transaction,
} from "@solana/web3.js";
import { ixFromRust } from "../solana";
import { importMigrationWasm } from "../solana/wasm";

export default async function migrateTokens(
  connection: Connection,
  payerAddress: string,
  program_id: string,
  from_mint: string,
  to_mint: string,
  input_token_account: string,
  output_token_account: string,
  amount: BigInt,
  commitment?: Commitment
) {
  const { authority_address, migrate_tokens } = await importMigrationWasm();
  const approvalIx = createApproveInstruction(
    new PublicKey(input_token_account),
    new PublicKey(authority_address(program_id)),
    new PublicKey(payerAddress),
    amount.valueOf()
  );
  const ix = ixFromRust(
    migrate_tokens(
      program_id,
      from_mint,
      to_mint,
      input_token_account,
      output_token_account,
      amount.valueOf()
    )
  );
  const transaction = new Transaction().add(approvalIx, ix);
  const { blockhash } = await connection.getLatestBlockhash(commitment);
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  return transaction;
}
