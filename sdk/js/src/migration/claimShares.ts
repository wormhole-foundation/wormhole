import { Connection, PublicKey, Transaction } from "@solana/web3.js";
import { ixFromRust } from "../solana";

export default async function claimShares(
  connection: Connection,
  payerAddress: string,
  program_id: string,
  from_mint: string,
  to_mint: string,
  output_token_account: string,
  lp_share_token_account: string,
  amount: BigInt
) {
  const { claim_shares } = await import(
    "../solana/migration/wormhole_migration"
  );
  const ix = ixFromRust(
    claim_shares(
      program_id,
      from_mint,
      to_mint,
      output_token_account,
      lp_share_token_account,
      amount
    )
  );
  const transaction = new Transaction().add(ix);
  const { blockhash } = await connection.getRecentBlockhash();
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  return transaction;
}
