import { postVaaSolana } from "@certusone/wormhole-sdk";
import { NodeWallet } from "@certusone/wormhole-sdk/lib/cjs/solana";
import { Connection, Keypair } from "@solana/web3.js";
import { CORE_BRIDGE_PROGRAM_ID } from "./consts";

export async function invokeVerifySignaturesAndPostVaa(
  connection: Connection,
  payer: Keypair,
  signedVaa: Buffer
) {
  const wallet = new NodeWallet(payer);
  return postVaaSolana(
    connection,
    wallet.signTransaction,
    CORE_BRIDGE_PROGRAM_ID,
    wallet.key(),
    signedVaa
  );
}
