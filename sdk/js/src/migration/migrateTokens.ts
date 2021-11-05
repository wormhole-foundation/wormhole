import { Token, TOKEN_PROGRAM_ID, u64 } from "@solana/spl-token";
import { Connection, PublicKey, Transaction } from "@solana/web3.js";
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
  amount: BigInt
) {
  const { authority_address, migrate_tokens } = await importMigrationWasm();
  const approvalIx = Token.createApproveInstruction(
    TOKEN_PROGRAM_ID,
    new PublicKey(input_token_account),
    new PublicKey(authority_address(program_id)),
    new PublicKey(payerAddress),
    [],
    new u64(amount.toString(16), 16)
  );
  const ix = ixFromRust(
    migrate_tokens(
      program_id,
      from_mint,
      to_mint,
      input_token_account,
      output_token_account,
      amount
    )
  );
  const transaction = new Transaction().add(approvalIx, ix);
  const { blockhash } = await connection.getRecentBlockhash();
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  return transaction;
}
