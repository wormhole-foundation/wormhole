import {
  Connection,
  Keypair,
  PublicKey,
  Transaction,
  TransactionInstruction,
} from "@solana/web3.js";
import { ixFromRust } from "./rust";

// is there a better pattern for this?
export async function postVaa(
  connection: Connection,
  signTransaction: (transaction: Transaction) => any,
  bridge_id: string,
  payer: string,
  vaa: Buffer
) {
  const {
    guardian_set_address,
    parse_guardian_set,
    verify_signatures_ix,
    post_vaa_ix,
  } = await import("./core/bridge");
  let bridge_state = await getBridgeState(connection, bridge_id);
  let guardian_addr = new PublicKey(
    guardian_set_address(bridge_id, bridge_state.guardian_set_index)
  );
  let acc = await connection.getAccountInfo(guardian_addr);
  if (acc?.data === undefined) {
    return;
  }
  let guardian_data = parse_guardian_set(new Uint8Array(acc?.data));

  let signature_set = Keypair.generate();
  let txs = verify_signatures_ix(
    bridge_id,
    payer,
    bridge_state.guardian_set_index,
    guardian_data,
    signature_set.publicKey.toString(),
    vaa
  );
  // Add transfer instruction to transaction
  for (let tx of txs) {
    let ixs: Array<TransactionInstruction> = tx.map((v: any) => {
      return ixFromRust(v);
    });
    let transaction = new Transaction().add(ixs[0], ixs[1]);
    const { blockhash } = await connection.getRecentBlockhash();
    transaction.recentBlockhash = blockhash;
    transaction.feePayer = new PublicKey(payer);
    transaction.partialSign(signature_set);

    // Sign transaction, broadcast, and confirm
    const signed = await signTransaction(transaction);
    const txid = await connection.sendRawTransaction(signed.serialize());
    await connection.confirmTransaction(txid);
  }

  let ix = ixFromRust(
    post_vaa_ix(bridge_id, payer, signature_set.publicKey.toString(), vaa)
  );
  let transaction = new Transaction().add(ix);
  const { blockhash } = await connection.getRecentBlockhash();
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payer);

  const signed = await signTransaction(transaction);
  const txid = await connection.sendRawTransaction(signed.serialize());
  await connection.confirmTransaction(txid);
}

async function getBridgeState(
  connection: Connection,
  bridge_id: string
): Promise<BridgeState> {
  const { parse_state, state_address } = await import("./core/bridge");
  let bridge_state = new PublicKey(state_address(bridge_id));
  let acc = await connection.getAccountInfo(bridge_state);
  if (acc?.data === undefined) {
    throw new Error("bridge state not found");
  }
  return parse_state(new Uint8Array(acc?.data));
}

interface BridgeState {
  // The current guardian set index, used to decide which signature sets to accept.
  guardian_set_index: number;

  // Lamports in the collection account
  last_lamports: number;

  // Bridge configuration, which is set once upon initialization.
  config: BridgeConfig;
}

interface BridgeConfig {
  // Period for how long a guardian set is valid after it has been replaced by a new one.  This
  // guarantees that VAAs issued by that set can still be submitted for a certain period.  In
  // this period we still trust the old guardian set.
  guardian_set_expiration_time: number;

  // Amount of lamports that needs to be paid to the protocol to post a message
  fee: number;
}
